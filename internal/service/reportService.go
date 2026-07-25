package service

import (
	models "avito-easy-report/internal/struct"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

// ReportStore — потокобезопасное in-memory хранилище отчётов
type ReportStore struct {
	mu      sync.RWMutex
	reports map[string]*StoredReport
}

// StoredReport — загруженный отчёт со всеми данными
type StoredReport struct {
	ID        string
	FileName  string
	UserID    uint
	CabinetID string
	Offers    []models.Offer
	File      *excelize.File
}

// NewReportStore создаёт хранилище
func NewReportStore() *ReportStore {
	return &ReportStore{
		reports: make(map[string]*StoredReport),
	}
}

// Add добавляет отчёт в хранилище
func (s *ReportStore) Add(id string, report *StoredReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports[id] = report
}

// Get возвращает отчёт по ID
func (s *ReportStore) Get(id string) *StoredReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reports[id]
}

// List возвращает список отчётов пользователя
func (s *ReportStore) List() []StoredReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]StoredReport, 0, len(s.reports))
	for _, r := range s.reports {
		result = append(result, *r)
	}
	return result
}

// ListByUser возвращает отчёты конкретного пользователя
func (s *ReportStore) ListByUser(userID uint) []StoredReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]StoredReport, 0)
	for _, r := range s.reports {
		if r.UserID == userID {
			result = append(result, *r)
		}
	}
	return result
}

// ListByCabinet возвращает отчёты кабинета
func (s *ReportStore) ListByCabinet(userID uint, cabinetID string) []StoredReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]StoredReport, 0)
	for _, r := range s.reports {
		if r.UserID == userID && r.CabinetID == cabinetID {
			result = append(result, *r)
		}
	}
	return result
}

// Remove удаляет отчёт
func (s *ReportStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.reports, id)
}

// Count возвращает количество отчётов
func (s *ReportStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.reports)
}

// ParseReport парсит XLSX из io.Reader и возвращает объявления, файл, предупреждения и найденные колонки
func ParseReport(reader io.Reader, fileName string, objStore *ObjectStore) ([]models.Offer, *excelize.File, []string, []string, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("ошибка открытия файла %s: %w", fileName, err)
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("лист 'Sheet1' не найден в %s: %w", fileName, err)
	}

	if len(rows) < 2 {
		return nil, nil, nil, nil, fmt.Errorf("файл %s пуст или содержит только заголовки", fileName)
	}

	columnIndexMap, warnings := getColumnIndexMap(rows[0])
	offers := make([]models.Offer, 0, len(rows)-1)
	listingColIdx := columnIndexMap["listingNumber"]

	// HR-отчёт: строим маппинг Лист1 (номер объявления → объект)
	objLookup := buildObjectLookup(f)

	for rowIdx, row := range rows[1:] {
		offer := parseRow(row, columnIndexMap)
		// Извлекаем номер объявления из HYPERLINK, если колонка есть
		if listingColIdx >= 0 {
			cellRef, _ := excelize.CoordinatesToCellName(listingColIdx+1, rowIdx+2)
			if formula, err := f.GetCellFormula("Sheet1", cellRef); err == nil {
				offer.ListingNumber = extractListingNumber(formula)
			}
		}
		// Разрешаем Объект: 1) ObjectStore (приоритет), 2) Лист1 из файла
		if offer.ListingNumber != "" {
			if obj, ok := objStore.Get(offer.ListingNumber); ok {
				offer.Object = obj
			} else if obj, ok := objLookup[offer.ListingNumber]; ok && offer.Object == "" {
				offer.Object = obj
			}
		}
		// Чистим #N/A из VLOOKUP-ошибок
		if offer.Object == "#N/A" || offer.Object == "#VALUE!" || offer.Object == "#REF!" {
			offer.Object = ""
		}
		offers = append(offers, offer)
	}

	// Собираем имена найденных колонок
	var foundColumns []string
	for key, idx := range columnIndexMap {
		if idx >= 0 {
			foundColumns = append(foundColumns, key)
		}
	}

	return offers, f, warnings, foundColumns, nil
}

// GetGroupedStats агрегирует объявления по указанному ключу (city/category/name)
func GetGroupedStats(offers []models.Offer, groupBy string) map[string]models.Stats {
	result := make(map[string]models.Stats)

	for _, offer := range offers {
		key := groupKey(offer, groupBy)
		if key == "" {
			continue
		}
		stats := result[key]
		stats.Contacts += offer.Contacts
		stats.Favorite += offer.Favorite
		stats.Promotion += offer.Promotion
		stats.Views += offer.Views
		stats.Shows += offer.Shows
		stats.LookPhone += offer.LookPhone
		stats.TargetViewers += offer.TargetViewers
		stats.ViewWithMessage += offer.ViewWithMessage
		stats.ViewersCost += offer.ViewersCost
		stats.Response += offer.Response
		result[key] = stats
	}
	return result
}

func groupKey(o models.Offer, groupBy string) string {
	switch groupBy {
	case "category":
		return o.Category
	case "name":
		return o.Name
	case "employee":
		return o.Employee
	case "object":
		return o.Object
	default:
		return o.City
	}
}

// GetTopListings возвращает топ-N индивидуальных объявлений по контактам (без агрегации)
func GetTopListings(offers []models.Offer, limit int) []models.ResultStats {
	result := make([]models.ResultStats, 0, len(offers))
	for _, o := range offers {
		expense := o.Promotion + o.ViewersCost
		result = append(result, models.ResultStats{
			Key:             o.ListingNumber,
			City:            o.City,
			Shows:           o.Shows,
			Views:           o.Views,
			Contacts:        o.Contacts,
			Favorite:        o.Favorite,
			Promotion:       o.Promotion,
			ViewersCost:     o.ViewersCost,
			Expense:         expense,
			PPConversion:    canDivByZero(float64(o.Views), float64(o.Shows)) * 100,
			PKConversion:    canDivByZero(float64(o.Contacts), float64(o.Views)) * 100,
			AvgContactPrice: canDivByZero(expense, float64(o.Contacts)),
			AvgViewPrice:    canDivByZero(expense, float64(o.Views)),
			TargetViewers:   o.TargetViewers,
			ViewWithMessage: o.ViewWithMessage,
			LookPhone:       o.LookPhone,
			Response:        o.Response,
		})
	}

	// Сортировка по убыванию контактов
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Contacts > result[i].Contacts {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	// Нумерация после сортировки и обрезки
	for i := range result {
		result[i].Number = i + 1
		// Фолбек: если номер объявления пуст — порядковый номер как строка
		if result[i].Key == "" {
			result[i].Key = fmt.Sprintf("%d", i+1)
		}
	}

	return result
}

// GetResultStats вычисляет производные метрики
func GetResultStats(stats map[string]models.Stats) []models.ResultStats {
	result := make([]models.ResultStats, 0, len(stats))
	for key, stat := range stats {
		resultStat := models.ResultStats{
			Key:             key,
			Views:           stat.Views,
			Favorite:        stat.Favorite,
			Shows:           stat.Shows,
			Contacts:        stat.Contacts,
			Promotion:       stat.Promotion,
			ViewersCost:     stat.ViewersCost,
			PPConversion:    canDivByZero(float64(stat.Views), float64(stat.Shows)) * 100,
			PKConversion:    canDivByZero(float64(stat.Contacts), float64(stat.Views)) * 100,
			AvgViewPrice:    canDivByZero(stat.Promotion+stat.ViewersCost, float64(stat.Views)),
			AvgContactPrice: canDivByZero(stat.Promotion+stat.ViewersCost, float64(stat.Contacts)),
			Expense:         stat.Promotion + stat.ViewersCost,
			TargetViewers:   stat.TargetViewers,
			ViewWithMessage: stat.ViewWithMessage,
			LookPhone:       stat.LookPhone,
			Response:        stat.Response,
		}
		result = append(result, resultStat)
	}
	return result
}

// ExportXLSX формирует сводный result.xlsx на одном листе: города, категории, подкатегории, топ-10 объявлений
func ExportXLSX(reports []StoredReport, w io.Writer) error {
	file := excelize.NewFile()
	defer file.Close()

	sheet := "Sheet1"
	_ = file.SetColWidth(sheet, "A", "I", 15)

	headers := []string{"", "Показы", "ПП%", "Просмотры", "ПК%", "Контакты", "Расход", "Ср. цена контакта", "Избранное"}
	offerHeaders := []string{"Номер объявления", "Город", "Показы", "ПП%", "Просмотры", "ПК%", "Контакты", "Расход", "Ср. цена контакта"}

	row := 1
	cell := func(col, r int) string {
		c, _ := excelize.CoordinatesToCellName(col, r)
		return c
	}

	writeHeaders := func(hdrs []string, style int) {
		for col, h := range hdrs {
			cellRef := cell(col+1, row)
			_ = file.SetCellValue(sheet, cellRef, h)
			_ = file.SetCellStyle(sheet, cellRef, cellRef, style)
		}
		row++
	}

	// Стили
	titleStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
	})
	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})

	writeSection := func(title string, firstCol string, stats []models.ResultStats, hdrs []string) {
		// Подставляем название первой колонки
		hdrs[0] = firstCol
		// Заголовок секции — жирный
		titleCell := cell(1, row)
		_ = file.SetCellValue(sheet, titleCell, title)
		_ = file.SetCellStyle(sheet, titleCell, titleCell, titleStyle)
		row++
		// Шапка таблицы — жирный + зелёная заливка
		writeHeaders(hdrs, headerStyle)
		// Данные
		for _, s := range stats {
			_ = file.SetCellValue(sheet, cell(1, row), s.Key)
			offset := 0
			if hdrs[0] == "Номер объявления" {
				_ = file.SetCellValue(sheet, cell(2, row), s.City)
				offset = 1
			}
			_ = file.SetCellValue(sheet, cell(2+offset, row), s.Shows)
			_ = file.SetCellValue(sheet, cell(3+offset, row), fmt.Sprintf("%.2f%%", s.PPConversion))
			_ = file.SetCellValue(sheet, cell(4+offset, row), s.Views)
			_ = file.SetCellValue(sheet, cell(5+offset, row), fmt.Sprintf("%.2f%%", s.PKConversion))
			_ = file.SetCellValue(sheet, cell(6+offset, row), s.Contacts)
			_ = file.SetCellValue(sheet, cell(7+offset, row), fmt.Sprintf("%.2f", s.Expense))
			_ = file.SetCellValue(sheet, cell(8+offset, row), fmt.Sprintf("%.2f", s.AvgContactPrice))
			_ = file.SetCellValue(sheet, cell(9+offset, row), s.Favorite)
			row++
		}
		row++ // пустая строка-разделитель
	}

	for _, report := range reports {
		// Название отчёта
		_ = file.SetCellValue(sheet, cell(1, row), "Отчёт: "+report.FileName)
		row += 2

		// Города
		writeSection("По городам", "Город", GetResultStats(GetGroupedStats(report.Offers, "city")), headers)
		// Категории
		writeSection("По категориям", "Категория", GetResultStats(GetGroupedStats(report.Offers, "category")), headers)
		// Подкатегории
		writeSection("По подкатегориям", "Подкатегория", GetResultStats(GetGroupedStats(report.Offers, "name")), headers)
		// Топ-10 объявлений
		writeSection("Топ-10 объявлений по контактам", "Номер объявления", GetTopListings(report.Offers, 10), offerHeaders)

		// HR: сотрудники и объекты (если есть данные)
		empStats := GetResultStats(GetGroupedStats(report.Offers, "employee"))
		if len(empStats) > 0 && empStats[0].Key != "" {
			writeSection("По сотрудникам", "Сотрудник", empStats, headers)
		}
		objStats := GetResultStats(GetGroupedStats(report.Offers, "object"))
		if len(objStats) > 0 && objStats[0].Key != "" {
			writeSection("По объектам", "Объект", objStats, headers)
		}
	}

	file.SetSheetName("Sheet1", "Сводка")
	return file.Write(w)
}

// --- Вспомогательные функции ---

func parseRow(row []string, columnIndex map[string]int) models.Offer {
	return models.Offer{
		City:            safeGet(row, columnIndex["city"]),
		Category:        safeGet(row, columnIndex["category"]),
		SubCategory:     safeGet(row, columnIndex["subCategory"]),
		ListingNumber:   safeGet(row, columnIndex["listingNumber"]),
		Shows:           getIntegerCell(safeGet(row, columnIndex["shows"])),
		Views:           getIntegerCell(safeGet(row, columnIndex["views"])),
		Favorite:        getIntegerCell(safeGet(row, columnIndex["favorite"])),
		Name:            safeGet(row, columnIndex["name"]),
		Contacts:        getIntegerCell(safeGet(row, columnIndex["contacts"])),
		Promotion:       getDoubleCell(safeGet(row, columnIndex["promotion"])),
		ViewersCost:     getDoubleCell(safeGet(row, columnIndex["viewierCost"])),
		ViewWithMessage: getIntegerCell(safeGet(row, columnIndex["viewWithMessage"])),
		LookPhone:       getIntegerCell(safeGet(row, columnIndex["lookPhone"])),
		TargetViewers:   getIntegerCell(safeGet(row, columnIndex["targetViewers"])),
		Response:        getIntegerCell(safeGet(row, columnIndex["response"])),
		Object:          safeGet(row, columnIndex["object"]),
		Employee:        safeGet(row, columnIndex["employee"]),
	}
}

func safeGet(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

// extractListingNumber вытаскивает номер объявления из HYPERLINK-формулы Avito
// Формат: =HYPERLINK("url","номер") → "номер"
func extractListingNumber(raw string) string {
	if raw == "" {
		return ""
	}
	// Ищем последний аргумент в кавычках: ,"число")
	lastQuote := strings.LastIndex(raw, "\"")
	if lastQuote < 0 {
		return raw // не формула — возвращаем как есть
	}
	prevQuote := strings.LastIndex(raw[:lastQuote], "\"")
	if prevQuote < 0 {
		return raw
	}
	inner := raw[prevQuote+1 : lastQuote]
	// Если внутри только цифры — это номер объявления
	if len(inner) > 0 {
		return inner
	}
	return raw
}

// buildObjectLookup читает Лист1 (номер объявления → объект) для HR-отчётов
func buildObjectLookup(f *excelize.File) map[string]string {
	lookup := make(map[string]string)
	rows, err := f.GetRows("Лист1")
	if err != nil {
		return lookup // Лист1 нет — не HR-отчёт
	}
	// Лист1: Col A=пусто, Col B=Номер объявления, Col C=Объект
	for i, row := range rows {
		if i == 0 {
			continue // пропускаем заголовок (или пустую строку)
		}
		if len(row) < 3 {
			continue
		}
		num := strings.TrimSpace(row[1])
		obj := strings.TrimSpace(row[2])
		if num != "" && obj != "" {
			lookup[num] = obj
		}
	}
	return lookup
}

func getColumnIndexMap(row []string) (map[string]int, []string) {
	columnIndex := make(map[string]int)
	var warnings []string

	// Маппинг ключ → список возможных названий колонок
	mappings := map[string][]string{
		"city":            {"Город"},
		"category":        {"Категория"},
		"subCategory":     {"Подкатегория"},
		"shows":           {"Показы"},
		"views":           {"Просмотры"},
		"favorite":        {"Добавили в избранное", "Добавили в\u00a0избранное"},
		"name":            {"Название объявления", "Параметр"},
		"listingNumber":   {"Номер объявления", "Номер\u00a0объявления", "ID объявления", "№ объявления", "ID", "Номер"},
		"contacts":        {"Контакты"},
		"employee":        {"Сотрудник"},
		"object":          {"Объект"},
		"promotion":       {"Расходы на продвижение", "Расходы на\u00a0продвижение"},
		"viewierCost":     {"Расходы на размещение и целевые действия", "Расходы на\u00a0размещение и\u00a0целевые\u00a0действия", "Расходы на объявления", "Расходы на\u00a0объявления"},
		"viewWithMessage": {"Написали в чат", "Написали в\u00a0чат"},
		"lookPhone":       {"Посмотрели телефон", "Посмотрели\u00a0телефон"},
		"targetViewers":   {"Целевые просмотры", "Целевые отклики", "Откликнулись на скидку в чате", "Откликнулись на\u00a0скидку в\u00a0чате"},
		"response":        {"Отклики"},
	}

	for key, names := range mappings {
		idx := findColumnIndex(row, names)
		columnIndex[key] = idx
		if idx < 0 {
			warnings = append(warnings, fmt.Sprintf("Колонка «%s» не найдена в отчёте", names[0]))
		}
	}

	return columnIndex, warnings
}

func findColumnIndex(row []string, columnNames []string) int {
	for i, cell := range row {
		cellClean := strings.ReplaceAll(cell, "\u00a0", " ")
		for _, name := range columnNames {
			nameClean := strings.ReplaceAll(name, "\u00a0", " ")
			if cellClean == nameClean {
				return i
			}
		}
	}
	return -1
}

func getDoubleCell(cell string) float64 {
	result, _ := strconv.ParseFloat(cell, 64)
	return result
}

func getIntegerCell(cell string) int {
	result, _ := strconv.Atoi(cell)
	return result
}

func canDivByZero(first float64, second float64) float64 {
	if second == 0 {
		return 0.0
	}
	return first / second
}

func getCellName(col int, row int) string {
	result, _ := excelize.CoordinatesToCellName(col, row)
	return result
}

func keepNumbersAndUnderscores(input string) string {
	re := regexp.MustCompile(`[^0-9_]`)
	return re.ReplaceAllString(input, "")
}

// GetSummary строит краткую сводку: топ-5 объявлений, заголовков, городов
func GetSummary(offers []models.Offer) models.StatsSummary {
	shows, views, contacts := 0, 0, 0
	var expense float64
	offerContacts := make(map[string]int)
	titleContacts := make(map[string]int)
	cityContacts := make(map[string]int)

	for _, o := range offers {
		shows += o.Shows
		views += o.Views
		contacts += o.Contacts
		expense += o.Promotion + o.ViewersCost
		offerContacts[o.Name] += o.Contacts
		titleContacts[o.Name] += o.Contacts
		cityContacts[o.City] += o.Contacts
	}

	return models.StatsSummary{
		TotalShows:    shows,
		TotalViews:    views,
		TotalContacts: contacts,
		TotalExpense:  expense,
		TopOffers:     topN(offerContacts, 5),
		TopTitles:     topN(titleContacts, 5),
		TopCities:     topN(cityContacts, 5),
	}
}

func topN(m map[string]int, n int) []models.TopItem {
	type kv struct {
		k string
		v int
	}
	list := make([]kv, 0, len(m))
	for k, v := range m {
		list = append(list, kv{k, v})
	}
	// Сортировка по убыванию
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].v > list[i].v {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	if n > len(list) {
		n = len(list)
	}
	result := make([]models.TopItem, n)
	for i := 0; i < n; i++ {
		result[i] = models.TopItem{Name: list[i].k, Value: list[i].v}
	}
	return result
}

// ComparePeriods сравнивает два периода и возвращает дельту
func ComparePeriods(early, late []models.Offer, groupBy string) models.CompareResponse {
	earlyStats := GetGroupedStats(early, groupBy)
	lateStats := GetGroupedStats(late, groupBy)
	earlyResult := GetResultStats(earlyStats)
	lateResult := GetResultStats(lateStats)

	// Считаем дельту: проходим по позднему периоду, ищем соответствие в раннем
	earlyMap := make(map[string]models.ResultStats)
	for _, s := range earlyResult {
		earlyMap[s.Key] = s
	}

	delta := make([]models.ResultStats, 0, len(lateResult))
	for _, s := range lateResult {
		e, ok := earlyMap[s.Key]
		if !ok {
			// Новый город/категория — вся статистика как прирост
			delta = append(delta, models.ResultStats{
				Key: s.Key, Shows: s.Shows, Views: s.Views, Contacts: s.Contacts,
				PPConversion: s.PPConversion, PKConversion: s.PKConversion,
				AvgViewPrice: s.AvgViewPrice, AvgContactPrice: s.AvgContactPrice,
				Expense: s.Expense,
			})
			continue
		}
		delta = append(delta, models.ResultStats{
			Key:             s.Key,
			Shows:           s.Shows - e.Shows,
			Views:           s.Views - e.Views,
			Contacts:        s.Contacts - e.Contacts,
			Favorite:        s.Favorite - e.Favorite,
			Promotion:       s.Promotion - e.Promotion,
			ViewersCost:     s.ViewersCost - e.ViewersCost,
			TargetViewers:   s.TargetViewers - e.TargetViewers,
			ViewWithMessage: s.ViewWithMessage - e.ViewWithMessage,
			LookPhone:       s.LookPhone - e.LookPhone,
			PPConversion:    s.PPConversion - e.PPConversion,
			PKConversion:    s.PKConversion - e.PKConversion,
			AvgViewPrice:    s.AvgViewPrice - e.AvgViewPrice,
			AvgContactPrice: s.AvgContactPrice - e.AvgContactPrice,
			Expense:         s.Expense - e.Expense,
		})
	}

	// Перестраиваем earlyResult в том же порядке что и delta (по ключам lateResult)
	syncedEarly := make([]models.ResultStats, 0, len(delta))
	for _, s := range delta {
		if e, ok := earlyMap[s.Key]; ok {
			syncedEarly = append(syncedEarly, e)
		} else {
			syncedEarly = append(syncedEarly, models.ResultStats{Key: s.Key})
		}
	}

	return models.CompareResponse{
		Early: models.PeriodStats{
			Summary: GetSummary(early),
			Stats:   syncedEarly,
		},
		Late: models.PeriodStats{
			Summary: GetSummary(late),
			Stats:   lateResult,
		},
		Delta: delta,
	}
}
