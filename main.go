package main

import (
	"avito-easy-report/internal/service"
	models "avito-easy-report/internal/struct"
	"path/filepath"
)

func main() {
	reports := service.GetAllReports()
	resultsFileName := make(map[string][]models.ResultStats)

	for _, reportFile := range reports {
		_, fileName := filepath.Split(reportFile.Path)
		offers := service.GetAllOffers(reportFile)
		stats := service.GetSimpleStatMap(offers)
		resultsFileName[fileName] = service.GetResultStats(stats)
	}
	service.SaveResultStats(resultsFileName)
}

// Конверсия просмотры-контакты
// Ср. цена просмотра
// Ср цена контакта
// статистика по категориям (подкатегориям)
// самое сильное объявление
// динамика затрат
// динамика просмотров
// динамика контактов
