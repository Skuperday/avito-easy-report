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

// ParseReport парсит XLSX из io.Reader и возвращает объявления, файл и предупреждения
func ParseReport(reader io.Reader, fileName string) ([]models.Offer, *excelize.File, []string, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ошибка открытия файла %s: %w", fileName, err)
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("лист 'Sheet1' не найден в %s: %w", fileName, err)
	}

	if len(rows) < 2 {
		return nil, nil, nil, fmt.Errorf("файл %s пуст или содержит только заголовки", fileName)
	}

	columnIndexMap, warnings := getColumnIndexMap(rows[0])
	offers := make([]models.Offer, 0, len(rows)-1)

	for _, row := range rows[1:] {
		offer := parseRow(row, columnIndexMap)
		offers = append(offers, offer)
	}

	return offers, f, warnings, nil
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
		return o.SubCategory
	default:
		return o.City
	}
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
			TargetViewers:   stat.TargetViewers,
			ViewWithMessage: stat.ViewWithMessage,
			LookPhone:       stat.LookPhone,
			Response:        stat.Response,
		}
		result = append(result, resultStat)
	}
	return result
}

// ExportXLSX формирует сводный result.xlsx в writer
func ExportXLSX(reports []StoredReport, w io.Writer) error {
	file := excelize.NewFile()
	defer file.Close()

	headers := []string{
		"Город",
		"Показы",
		"Просмотры",
		"Контакты",
		"ПП Конверсия",
		"ПК Конверсия",
		"Избранное",
		"Продвижение",
		"Затраты на просмотры",
		"Целевые просмотры",
		"Написали в чат",
		"Смотрели телефон",
		"Средняя цена просмотра",
		"Средняя цена контакта",
	}

	for i, report := range reports {
		stats := GetGroupedStats(report.Offers, "city")
		resultStats := GetResultStats(stats)
		sheetName := keepNumbersAndUnderscores(report.FileName)

		// Первый лист переименовываем, остальные создаём
		var index int
		var err error
		if i == 0 {
			file.SetSheetName("Sheet1", sheetName)
			index = 0
		} else {
			index, err = file.NewSheet(sheetName)
			if err != nil {
				return fmt.Errorf("ошибка создания листа %s: %w", sheetName, err)
			}
		}

		file.SetActiveSheet(index)
		_ = file.SetColWidth(sheetName, "A", "N", 15)

		for col, header := range headers {
			_ = file.SetCellValue(sheetName, getCellName(col+1, 1), header)
		}

		for rowIdx, stat := range resultStats {
			row := rowIdx + 2
			_ = file.SetCellValue(sheetName, getCellName(1, row), stat.Key)
			_ = file.SetCellValue(sheetName, getCellName(2, row), stat.Shows)
			_ = file.SetCellValue(sheetName, getCellName(3, row), stat.Views)
			_ = file.SetCellValue(sheetName, getCellName(4, row), stat.Contacts)
			_ = file.SetCellValue(sheetName, getCellName(5, row), fmt.Sprintf("%.2f%%", stat.PPConversion))
			_ = file.SetCellValue(sheetName, getCellName(6, row), fmt.Sprintf("%.2f%%", stat.PKConversion))
			_ = file.SetCellValue(sheetName, getCellName(7, row), stat.Favorite)
			_ = file.SetCellValue(sheetName, getCellName(8, row), stat.Promotion)
			_ = file.SetCellValue(sheetName, getCellName(9, row), stat.ViewersCost)
			_ = file.SetCellValue(sheetName, getCellName(10, row), stat.TargetViewers)
			_ = file.SetCellValue(sheetName, getCellName(11, row), stat.ViewWithMessage)
			_ = file.SetCellValue(sheetName, getCellName(12, row), stat.LookPhone)
			_ = file.SetCellValue(sheetName, getCellName(13, row), fmt.Sprintf("%.2f", stat.AvgViewPrice))
			_ = file.SetCellValue(sheetName, getCellName(14, row), fmt.Sprintf("%.2f", stat.AvgContactPrice))
		}
	}

	return file.Write(w)
}

// --- Вспомогательные функции ---

func parseRow(row []string, columnIndex map[string]int) models.Offer {
	return models.Offer{
		City:            safeGet(row, columnIndex["city"]),
		Category:        safeGet(row, columnIndex["category"]),
		SubCategory:     safeGet(row, columnIndex["subCategory"]),
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
	}
}

func safeGet(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
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
		"contacts":        {"Контакты", "Отклики"},
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
