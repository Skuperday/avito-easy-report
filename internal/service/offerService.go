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
func GetAllOffers(report *excelize.File) []models.Offer {
	var result []models.Offer
	rows, _ := report.GetRows("Sheet1")
	for _, row := range rows {
		nextOffer := parseRow(row)
		result = append(result, nextOffer)
	}
	return result
}

// Функция для получения чтения всех отчетов (статистики) из директории /reports
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
	views, _ := strconv.Atoi(row[10])               // assuming column index for views is 10
	favorite, _ := strconv.Atoi(row[16])            // assuming column index for favorite is 16
	contacts, _ := strconv.Atoi(row[11])            // assuming column index for contacts is 11
	promotion, _ := strconv.ParseFloat(row[13], 64) // assuming column index for promotion is 13

	return models.Offer{
		City:        row[2], // assuming column index for city is 2
		Category:    row[4], // assuming column index for category is 4
		SubCategory: row[5], // assuming column index for subcategory is 5
		Views:       views,
		Favorite:    favorite,
		Name:        row[7], // assuming column index for name is 7
		Contacts:    contacts,
		Promotion:   promotion,
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
			PKConversion:    canDivByZero(float64(stat.Contacts), float64(stat.Views)) * 100,
			AvgViewPrice:    canDivByZero(stat.Promotion, float64(stat.Views)),
			AvgContactPrice: canDivByZero(stat.Promotion, float64(stat.Contacts)),
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
	headers := []string{"Город", "Просмотры", "Избранное", "Контакты", "Продвижение", "PK Конверсия", "Средняя цена просмотра", "Средняя цена контакта"}

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
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Favorite)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Contacts)
			col++
			file.SetCellValue(corrSheetName, getCellName(col, row), stat.Promotion)
			col++
			pkConvValue := fmt.Sprintf("%.2f", stat.PKConversion)
			pkConvValue += "%"
			file.SetCellValue(corrSheetName, getCellName(col, row), pkConvValue)
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
