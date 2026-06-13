package service

import (
	models "avito-easy-report/internal/struct"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
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
	ID       string
	FileName string
	Offers   []models.Offer
	File     *excelize.File // оригинальный файл для экспорта
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

// List возвращает список всех отчётов
func (s *ReportStore) List() []StoredReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]StoredReport, 0, len(s.reports))
	for _, r := range s.reports {
		result = append(result, *r)
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

// GetSimpleStatMap агрегирует объявления по городам
func GetSimpleStatMap(offers []models.Offer) map[string]models.Stats {
	result := make(map[string]models.Stats)

	for _, offer := range offers {
		stats := result[offer.City]
		stats.Contacts += offer.Contacts
		stats.Favorite += offer.Favorite
		stats.Promotion += offer.Promotion
		stats.Views += offer.Views
		stats.Shows += offer.Shows
		stats.LookPhone += offer.LookPhone
		stats.TargetViewers += offer.TargetViewers
		stats.ViewWithMessage += offer.ViewWithMessage
		stats.ViewersCost += offer.ViewersCost
		result[offer.City] = stats
	}
	return result
}

// GetResultStats вычисляет производные метрики
func GetResultStats(stats map[string]models.Stats) []models.ResultStats {
	result := make([]models.ResultStats, 0, len(stats))
	for city, stat := range stats {
		resultStat := models.ResultStats{
			City:            city,
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
		stats := GetSimpleStatMap(report.Offers)
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
			_ = file.SetCellValue(sheetName, getCellName(1, row), stat.City)
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
		"favorite":        {"Добавили в избранное"},
		"name":            {"Название объявления"},
		"contacts":        {"Контакты"},
		"promotion":       {"Расходы на продвижение"},
		"viewierCost":     {"Расходы на размещение и целевые действия"},
		"viewWithMessage": {"Написали в чат"},
		"lookPhone":       {"Посмотрели телефон"},
		"targetViewers":   {"Целевые просмотры", "Целевые отклики"},
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
		if slices.Contains(columnNames, cell) {
			return i
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
