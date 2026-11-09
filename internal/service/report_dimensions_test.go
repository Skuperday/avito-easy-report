package service

import (
	models "avito-easy-report/internal/struct"
	"bytes"
	"strconv"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseReportUsesHyperlinkTitleAndDetailedSubcategory(t *testing.T) {
	input := excelize.NewFile()
	defer input.Close()
	for col, header := range []string{"Номер объявления", "Категория", "Подкатегория", "Параметр", "Название объявления", "Показы", "Контакты"} {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := input.SetCellValue("Sheet1", cell, header); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		index                int
		id, parameter, title string
		shows, contacts      int
	}{
		{2, "8398860610", "Приём вторсырья", "Скупка АКБ Прием аккумуляторов", 12, 2},
		{3, "8398860611", "Вывоз мусора", "Вывоз мусора в Севастополе", 7, 1},
	} {
		if err := input.SetCellFormula("Sheet1", "A"+strconv.Itoa(row.index), `HYPERLINK("https://www.avito.ru/item","`+row.id+`")`); err != nil {
			t.Fatal(err)
		}
		if err := input.SetCellValue("Sheet1", "B"+strconv.Itoa(row.index), "Услуги"); err != nil {
			t.Fatal(err)
		}
		if err := input.SetCellValue("Sheet1", "C"+strconv.Itoa(row.index), "Предложение услуг"); err != nil {
			t.Fatal(err)
		}
		if err := input.SetCellValue("Sheet1", "D"+strconv.Itoa(row.index), row.parameter); err != nil {
			t.Fatal(err)
		}
		if err := input.SetCellFormula("Sheet1", "E"+strconv.Itoa(row.index), `HYPERLINK("https://www.avito.ru/item","`+row.title+`")`); err != nil {
			t.Fatal(err)
		}
		if err := input.SetCellInt("Sheet1", "F"+strconv.Itoa(row.index), row.shows); err != nil {
			t.Fatal(err)
		}
		if err := input.SetCellInt("Sheet1", "G"+strconv.Itoa(row.index), row.contacts); err != nil {
			t.Fatal(err)
		}
	}
	var data bytes.Buffer
	if err := input.Write(&data); err != nil {
		t.Fatal(err)
	}
	offers, file, _, _, err := ParseReport(bytes.NewReader(data.Bytes()), "sample.xlsx", NewObjectStore())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if len(offers) != 2 {
		t.Fatalf("offers: %d", len(offers))
	}
	if offers[0].Name != "Скупка АКБ Прием аккумуляторов" || offers[0].SubCategory != "Приём вторсырья" || offers[0].Category != "Услуги" || offers[0].ListingNumber != "8398860610" {
		t.Fatalf("неверные поля первого объявления: %#v", offers[0])
	}
	if offers[1].Name != "Вывоз мусора в Севастополе" || offers[1].SubCategory != "Вывоз мусора" {
		t.Fatalf("неверные поля второго объявления: %#v", offers[1])
	}
	if got := GetGroupedStats(offers, "name"); len(got) != 2 || got["Скупка АКБ Прием аккумуляторов"].Shows != 12 {
		t.Fatalf("неверная группировка заголовков: %#v", got)
	}
	if got := GetGroupedStats(offers, "subcategory"); len(got) != 2 || got["Приём вторсырья"].Contacts != 2 {
		t.Fatalf("неверная группировка подкатегорий: %#v", got)
	}
}

func TestHRExportSubcategoryUsesSubcategoryNotTitle(t *testing.T) {
	report := StoredReport{ReportType: models.ReportTypeHR, Offers: []models.Offer{{
		Name: "Скупка АКБ", Category: "Услуги", SubCategory: "Приём вторсырья", Shows: 12,
	}}}
	var out bytes.Buffer
	if err := ExportXLSX([]StoredReport{report}, &out); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("Сводка")
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range rows {
		if len(row) > 0 && row[0] == "По подкатегориям" {
			if got := rows[i+2][0]; got != "Приём вторсырья" {
				t.Fatalf("HR-подкатегория %q вместо «Приём вторсырья»", got)
			}
			return
		}
	}
	t.Fatal("отсутствует секция HR-подкатегорий")
}

func TestParseReportHyperlinkTitleWithEscapedQuotes(t *testing.T) {
	input := excelize.NewFile()
	defer input.Close()
	if err := input.SetCellValue("Sheet1", "A1", "Название объявления"); err != nil {
		t.Fatal(err)
	}
	if err := input.SetCellValue("Sheet1", "B1", "Подкатегория"); err != nil {
		t.Fatal(err)
	}
	if err := input.SetCellFormula("Sheet1", "A2", `HYPERLINK("https://example.com/a,b","Продам ""Ромашка"" дешево")`); err != nil {
		t.Fatal(err)
	}
	if err := input.SetCellValue("Sheet1", "B2", "Растения"); err != nil {
		t.Fatal(err)
	}
	if err := input.SetCellFormula("Sheet1", "A3", `HYPERLINK("https://example.com/a",B2)`); err != nil {
		t.Fatal(err)
	}
	if err := input.SetCellValue("Sheet1", "B3", "Растения"); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := input.Write(&data); err != nil {
		t.Fatal(err)
	}
	offers, f, _, _, err := ParseReport(bytes.NewReader(data.Bytes()), "quoted.xlsx", NewObjectStore())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if len(offers) != 2 {
		t.Fatalf("offers: %d", len(offers))
	}
	if offers[0].Name != `Продам "Ромашка" дешево` {
		t.Errorf("кавычки заголовка: %q", offers[0].Name)
	}
	if offers[1].Name != "" {
		t.Errorf("невычисленный аргумент не должен становиться формулой: %q", offers[1].Name)
	}
}

func TestParseReportKeepsLegacyParameterWithoutTitleColumn(t *testing.T) {
	input := excelize.NewFile()
	defer input.Close()
	for col, label := range []string{"Категория", "Подкатегория", "Параметр"} {
		ref, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := input.SetCellValue("Sheet1", ref, label); err != nil {
			t.Fatal(err)
		}
	}
	for col, value := range []string{"Услуги", "Предложение услуг", "Приём вторсырья"} {
		ref, _ := excelize.CoordinatesToCellName(col+1, 2)
		if err := input.SetCellValue("Sheet1", ref, value); err != nil {
			t.Fatal(err)
		}
	}
	var data bytes.Buffer
	if err := input.Write(&data); err != nil {
		t.Fatal(err)
	}
	offers, f, _, _, err := ParseReport(bytes.NewReader(data.Bytes()), "legacy.xlsx", NewObjectStore())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if len(offers) != 1 || offers[0].Name != "Приём вторсырья" || offers[0].SubCategory != "Приём вторсырья" {
		t.Fatalf("legacy fallback: %#v", offers)
	}
}
