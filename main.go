package main

import (
	"avito-easy-report/internal/service"
	models "avito-easy-report/internal/struct"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	input := bufio.NewScanner(os.Stdin)
	defer input.Scan()
	reports := service.GetAllReports()
	resultsFileName := make(map[string][]models.ResultStats)

	for _, reportFile := range reports {
		_, fileName := filepath.Split(reportFile.Path)
		offers := service.GetAllOffers(reportFile)
		stats := service.GetSimpleStatMap(offers)
		resultsFileName[fileName] = service.GetResultStats(stats)
	}
	service.SaveResultStats(resultsFileName)
	fmt.Println("Готово")
}

// Конверсия просмотры-контакты
// Ср. цена просмотра
// Ср цена контакта
// статистика по категориям (подкатегориям)
// самое сильное объявление
// динамика затрат
// динамика просмотров
// динамика контактов
