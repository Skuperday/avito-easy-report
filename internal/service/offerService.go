package service

import (
	models "avito-easy-report/internal/struct"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// Функция для получения всех объявлений из Excel
// возвращает слайс объявлений
// получает на вход объект содержащий файл отчета
func GetAllOffers(report *excelize.File) []models.Offer {
	var result []models.Offer
	rows, _ := report.GetRows("Sheet1")
	for _, row := range rows {
		nextOffer := parseRow(row)
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

// Функция разбора строки из XML файла
func parseRow(row []string) models.Offer {

	views, _ := strconv.Atoi(row[12])                 // Просмотры объявления
	favorite, _ := strconv.Atoi(row[21])              // Добавлено в избранное
	contacts, _ := strconv.Atoi(row[15])              // Контакты
	promotion, _ := strconv.ParseFloat(row[30], 64)   // Затраты на продвижение
	city := row[2]                                    // Город
	category := row[4]                                // Категория
	subCategory := row[5]                             // Подкатегория
	name := row[7]                                    // Название
	viewierCost, _ := strconv.ParseFloat(row[29], 64) // Затраты на просмотры
	viewWithMessage, _ := strconv.Atoi(row[16])       // Написали в чат
	lookPhone, _ := strconv.Atoi(row[17])             // Смотрели телефон
	targetViewers, _ := strconv.Atoi(row[33])         // Целевые просмотры

	return models.Offer{
		City:            city,
		Category:        category,
		SubCategory:     subCategory,
		Views:           views,
		Favorite:        favorite,
		Name:            name,
		Contacts:        contacts,
		Promotion:       promotion,
		ViewersCost:     viewierCost,
		ViewWithMessage: viewWithMessage,
		LookPhone:       lookPhone,
		TargetViewers:   targetViewers,
	}
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
