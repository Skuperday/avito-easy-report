package service

import (
	models "avito-easy-report/internal/struct"
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExportXLSXRegularContainsOnlyRegularSections(t *testing.T) {
	reports := []StoredReport{{
		FileName:   "regular.xlsx",
		ReportType: models.ReportTypeRegular,
		Offers: []models.Offer{{
			City: "Москва", Category: "Работа", Name: "Водитель", ListingNumber: "123",
			Employee: "Анна", Object: "Склад", Shows: 10,
		}},
	}}

	titles := exportSectionTitles(t, reports)
	for _, want := range []string{
		"По городам",
		"По категориям",
		"По подкатегориям",
		"Топ-10 объявлений по контактам",
	} {
		if !titles[want] {
			t.Errorf("в обычном XLSX отсутствует секция %q", want)
		}
	}
	for title := range titles {
		if title == "По сотрудникам" || title == "По объектам" || strings.HasPrefix(title, "Объекты сотрудника:") {
			t.Errorf("обычный XLSX не должен содержать HR-секцию %q", title)
		}
	}
}

func TestGetEmployeeObjectStatsSeparatesSameObjectByEmployee(t *testing.T) {
	got := GetEmployeeObjectStats([]models.Offer{
		{Employee: "Анна", Object: "Склад", Views: 20, Response: 2},
		{Employee: "Борис", Object: "Склад", Views: 10, Response: 3},
	})

	if len(got) != 2 {
		t.Fatalf("одинаковый объект разных сотрудников не должен смешиваться: %#v", got)
	}
	if got[0].Employee != "Анна" || got[0].Object != "Склад" || got[0].ResponseConversion != 10 {
		t.Fatalf("неверная HR-строка Анна/Склад: %#v", got[0])
	}
	if got[1].Employee != "Борис" || got[1].Object != "Склад" || got[1].ResponseConversion != 30 {
		t.Fatalf("неверная HR-строка Борис/Склад: %#v", got[1])
	}
}

func TestExportXLSXHRContainsAdditionalSections(t *testing.T) {
	reports := []StoredReport{{
		FileName:   "hr.xlsx",
		ReportType: models.ReportTypeHR,
		Offers: []models.Offer{{
			City: "Москва", Category: "Работа", Name: "Водитель", ListingNumber: "123",
			Employee: "Анна", Object: "Склад", Shows: 10,
		}},
	}}

	titles := exportSectionTitles(t, reports)
	for _, want := range []string{"По сотрудникам", "По объектам", "Объекты сотрудника: Анна"} {
		if !titles[want] {
			t.Errorf("в HR XLSX отсутствует секция %q", want)
		}
	}
}

func TestResponseConversionUsesResponsesPerView(t *testing.T) {
	grouped := GetResultStats(map[string]models.Stats{"Анна": {Views: 20, Response: 2}})
	if len(grouped) != 1 || grouped[0].ResponseConversion != 10 {
		t.Fatalf("групповая конверсия должна быть 2/20 = 10%%, получено %#v", grouped)
	}
	listings := GetTopListings([]models.Offer{{ListingNumber: "1", Views: 20, Response: 2}}, 10)
	if len(listings) != 1 || listings[0].ResponseConversion != 10 {
		t.Fatalf("конверсия объявления должна быть 2/20 = 10%%, получено %#v", listings)
	}
}

func TestComparePeriodsIncludesResponseMetricsInDelta(t *testing.T) {
	early := []models.Offer{{City: "Москва", Views: 20, Response: 2, Promotion: 30, ViewersCost: 10}}
	late := []models.Offer{
		{City: "Москва", Views: 40, Response: 8, Promotion: 60, ViewersCost: 20},
		{City: "Казань", Views: 10, Response: 5, Promotion: 20, ViewersCost: 5},
	}

	result := ComparePeriods(early, late, "city")
	delta := make(map[string]models.ResultStats, len(result.Delta))
	for _, row := range result.Delta {
		delta[row.Key] = row
	}
	moscow := delta["Москва"]
	if moscow.Response != 6 || moscow.ResponseConversion != 10 || moscow.AvgResponsePrice != -10 {
		t.Fatalf("дельта существующей группы должна включать HR-метрики: %#v", moscow)
	}
	kazan := delta["Казань"]
	if kazan.Response != 5 || kazan.ResponseConversion != 50 || kazan.AvgResponsePrice != 5 {
		t.Fatalf("дельта новой группы должна включать HR-метрики: %#v", kazan)
	}
}

func TestEmployeeObjectStatsDistinguishesMissingAndLiteralValues(t *testing.T) {
	got := GetEmployeeObjectStats([]models.Offer{
		{Employee: "", Object: "Склад"},
		{Employee: "Сотрудник не указан", Object: "Склад"},
		{Employee: "Анна", Object: ""},
		{Employee: "Анна", Object: "Объект не указан"},
	})
	if len(got) != 4 {
		t.Fatalf("пустые значения нельзя смешивать с буквальными placeholder: %#v", got)
	}
	missingEmployees, missingObjects := 0, 0
	for _, row := range got {
		if row.EmployeeMissing {
			missingEmployees++
		}
		if row.ObjectMissing {
			missingObjects++
		}
		if !row.EmployeeMissing && row.Employee == "Сотрудник не указан" {
			t.Errorf("буквальный placeholder сотрудника должен визуально отличаться: %#v", row)
		}
		if !row.ObjectMissing && row.Object == "Объект не указан" {
			t.Errorf("буквальный placeholder объекта должен визуально отличаться: %#v", row)
		}
	}
	if missingEmployees != 1 || missingObjects != 1 {
		t.Fatalf("missing-флаги должны однозначно различать значения: %#v", got)
	}
}

func TestEmployeeAndObjectGroupsIncludeMissingSeparately(t *testing.T) {
	tests := []struct {
		name         string
		groupBy      string
		missingLabel string
		offers       []models.Offer
	}{
		{
			name: "employee", groupBy: "employee", missingLabel: "Сотрудник не указан",
			offers: []models.Offer{{Employee: "", Views: 1}, {Employee: "Сотрудник не указан", Views: 2}},
		},
		{
			name: "object", groupBy: "object", missingLabel: "Объект не указан",
			offers: []models.Offer{{Object: "", Views: 1}, {Object: "Объект не указан", Views: 2}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats := GetGroupedStats(test.offers, test.groupBy)
			if len(stats) != 2 {
				t.Fatalf("missing и буквальный placeholder должны быть отдельными группами: %#v", stats)
			}
			if stats[test.missingLabel].Views != 1 || stats[test.missingLabel+" (значение из отчёта)"].Views != 2 {
				t.Fatalf("группы должны иметь разные пользовательские ключи: %#v", stats)
			}
		})
	}
}

func TestExportXLSXHRWritesObjectsForMissingEmployee(t *testing.T) {
	titles := exportSectionTitles(t, []StoredReport{{
		FileName:   "hr.xlsx",
		ReportType: models.ReportTypeHR,
		Offers:     []models.Offer{{Object: "Склад", Shows: 10}},
	}})
	if !titles["Объекты сотрудника: Сотрудник не указан"] {
		t.Fatalf("HR XLSX должен содержать секцию объектов отсутствующего сотрудника: %#v", titles)
	}
}

func exportSectionTitles(t *testing.T, reports []StoredReport) map[string]bool {
	t.Helper()
	var output bytes.Buffer
	if err := ExportXLSX(reports, &output); err != nil {
		t.Fatalf("ExportXLSX: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("открыть экспорт: %v", err)
	}
	defer file.Close()
	rows, err := file.GetRows("Сводка")
	if err != nil {
		t.Fatalf("прочитать сводку: %v", err)
	}
	titles := make(map[string]bool)
	for _, row := range rows {
		if len(row) > 0 {
			titles[row[0]] = true
		}
	}
	return titles
}
