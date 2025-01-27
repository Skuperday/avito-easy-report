package service

import (
	models "avito-easy-report/internal/struct"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Функция для получения всех объявлений из Excel
// возвращает слайс объявлений
// получает на вход объект содержащий файл отчета
func GetAllOffers(report *excelize.File) []models.Offer {
	var result []models.Offer
	rows, _ := report.GetRows("Sheet1")
	var columnIndexMap = GetColumnIndexMap(rows[0])
	for _, row := range rows[1:] {
		nextOffer := parseRow(row, columnIndexMap)
		result = append(result, nextOffer)
	}
	return result
}

// Функция для получения всех отчетов (статистики) из директории /reports
func GetAllReports() []*excelize.File {
	files, err := os.ReadDir("./reports")
	if err != nil {
		log.Fatal(err)
	}

	var result []*excelize.File

	for _, file := range files {
		f, err := excelize.OpenFile("./reports/" + file.Name())
		if err != nil {
			log.Println(err)
			return nil
		}
		result = append(result, f)
	}
	return result
}

// Функция разбора строки из XLSX файла
func parseRow(row []string, columnIndex map[string]int) models.Offer {
	return models.Offer{
		City:            row[columnIndex["city"]],
		Category:        row[columnIndex["category"]],
		SubCategory:     row[columnIndex["subCategory"]],
		Views:           GetIntegerCell(row[columnIndex["views"]]),
		Favorite:        GetIntegerCell(row[columnIndex["favorite"]]),
		Name:            row[columnIndex["name"]],
		Contacts:        GetIntegerCell(row[columnIndex["contacts"]]),
		Promotion:       GetDoubleCell(row[columnIndex["promotion"]]),
		ViewersCost:     GetDoubleCell(row[columnIndex["viewierCost"]]),
		ViewWithMessage: GetIntegerCell(row[columnIndex["viewWithMessage"]]),
		LookPhone:       GetIntegerCell(row[columnIndex["lookPhone"]]),
		TargetViewers:   GetIntegerCell(row[columnIndex["targetViewers"]]),
	}
}

func GetDoubleCell(cell string) float64 {
	result, _ := strconv.ParseFloat(cell, 64)
	return result
}

func GetIntegerCell(cell string) int {
	result, _ := strconv.Atoi(cell)
	return result
}

func GetColumnIndexMap(row []string) map[string]int {
	columnIndex := make(map[string]int)

	columnIndex["city"] = FindColumnIndex(row, []string{"Город"})
	columnIndex["category"] = FindColumnIndex(row, []string{"Категория"})
	columnIndex["subCategory"] = FindColumnIndex(row, []string{"Подкатегория"})
	columnIndex["views"] = FindColumnIndex(row, []string{"Просмотры"})
	columnIndex["favorite"] = FindColumnIndex(row, []string{"Добавили в избранное"})
	columnIndex["name"] = FindColumnIndex(row, []string{"Название объявления"})
	columnIndex["contacts"] = FindColumnIndex(row, []string{"Контакты"})
	columnIndex["promotion"] = FindColumnIndex(row, []string{"Расходы на продвижение"})
	columnIndex["viewierCost"] = FindColumnIndex(row, []string{"Расходы на размещение и целевые действия"})
	columnIndex["viewWithMessage"] = FindColumnIndex(row, []string{"Написали в чат"})
	columnIndex["lookPhone"] = FindColumnIndex(row, []string{"Посмотрели телефон"})
	columnIndex["targetViewers"] = FindColumnIndex(row, []string{"Целевые просмотры"})

	return columnIndex
}

func FindColumnIndex(row []string, columnNames []string) int {
	for i, cell := range row {
		if slices.Contains(columnNames, cell) {
			return i
		}
	}
	println("Не найдено совпадений для колонок: " + strings.Join(columnNames, ", "))
	return -1
}

// Функция для отбора статистики из данных по объявлениям.
// необходимые поля определены в модели stats
func GetSimpleStatMap(offers []models.Offer) map[string]models.Stats {

	result := make(map[string]models.Stats)

	for _, offer := range offers {
		stats := result[offer.City]
		stats.Contacts += offer.Contacts
		stats.Favorite += offer.Favorite
		stats.Promotion += offer.Promotion
		stats.Views += offer.Views
		stats.LookPhone += offer.LookPhone
		stats.TargetViewers += offer.TargetViewers
		stats.ViewWithMessage += offer.ViewWithMessage
		stats.ViewersCost += offer.ViewersCost
		result[offer.City] = stats
	}
	return result
}

// Конверсия просмотры-контакты
// Ср. цена просмотра
// Ср цена контакта
func GetResultStats(stats map[string]models.Stats) []models.ResultStats {
	var result []models.ResultStats
	for city, stat := range stats {
		resultStat := models.ResultStats{
			City:            city,
			Views:           stat.Views,
			Favorite:        stat.Favorite,
			Contacts:        stat.Contacts,
			Promotion:       stat.Promotion,
			ViewersCost:     stat.ViewersCost,
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

func canDivByZero(first float64, second float64) float64 {
	if second == 0 {
		return 0.0
	}
	return first / second
}

// самое сильное объявление TODO
// динамика затрат
// динамика просмотров
// динамика контактов
func SaveResultStats(resultStatsMap map[string][]models.ResultStats) {
	file := excelize.NewFile()
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Создаем заголовки на русском языке
	headers := []string{
		"Город",
		"Просмотры",
		"Контакты",
		"Избранное",
		"Продвижение",
		"PK Конверсия",
		"Затраты на просмотры",
		"Целевые просмотры",
		"Написали в чат",
		"Смотрели телефон",
		"Средняя цена просмотра",
		"Средняя цена контакта",
	}

	for sheetName, stats := range resultStatsMap {
		if file.SheetCount > 1 {
			file.DeleteSheet("Sheet1")
		}
		corrSheetName := keepNumbersAndUnderscores(sheetName)
		// Создаем новый лист с указанным именем
		index, err := file.NewSheet(corrSheetName)
		if err != nil {
			fmt.Println(err.Error())
		}
		// Устанавливаем активный лист
		file.SetActiveSheet(index)
		error := file.SetColWidth(corrSheetName, "A", "I", 15)
		if error != nil {
			fmt.Println(error.Error())
		}
		// Заполняем заголовки
		for i, header := range headers {
			file.SetCellValue(corrSheetName, getCellName(i+1, 1), header)
		}

		// Заполняем ячейки данными из структуры ResultStats
		for i, stat := range stats {
			col := 1     // Начальная колонка
			row := i + 2 // Начальная строка
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.City)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Views)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Contacts)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Favorite)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Promotion)
			col++
			pkConvValue := fmt.Sprintf("%.2f", stat.PKConversion)
			pkConvValue += "%"
			file.SetCellValue(corrSheetName, getCellName(col, row), pkConvValue)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.ViewersCost)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.TargetViewers)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.ViewWithMessage)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.LookPhone)
			col++
			avgViewPrice := fmt.Sprintf("%.2f", stat.AvgViewPrice)
			file.SetCellValue(corrSheetName, getCellName(col, row), avgViewPrice)
			col++
			avgContactPrice := fmt.Sprintf("%.2f", stat.AvgContactPrice)
			file.SetCellValue(corrSheetName, getCellName(col, row), avgContactPrice)
		}

	}
	// Сохраняем файл Excel
	if err := file.SaveAs("result.xlsx"); err != nil {
		println("Error saving Excel file:", err.Error())
	}
}

// Функция для получения имени ячейки на основе номера колонки и строки
func getCellName(col int, row int) string {
	result, _ := excelize.CoordinatesToCellName(col, row)
	return result
}

func findBestOffer(offers []models.Offer) models.Offer {
	bestOffer := offers[0]
	for _, offer := range offers {
		if offer.Contacts > bestOffer.Contacts {
			bestOffer = offer
		}
	}
	return bestOffer
}

func keepNumbersAndUnderscores(input string) string {
	// Создаем регулярное выражение, которое соответствует цифрам и символам "_"
	re := regexp.MustCompile(`[^0-9_]`)
	// Удаляем все символы, кроме цифр и "_"
	result := re.ReplaceAllString(input, "")

	return result
}
