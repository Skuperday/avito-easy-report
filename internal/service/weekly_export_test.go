package service

import (
	models "avito-easy-report/internal/struct"
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestRegularExportUsesWeeklyLayoutAndLeavesManualFieldsBlank(t *testing.T) {
	report := StoredReport{FileName: "Статистика_с_2026-10-26_по_2026-11-01.xlsx", ReportType: models.ReportTypeRegular, Offers: []models.Offer{
		{Region: "Московская область", City: "Москва", Name: "Заголовок А", SubCategory: "Подкат А", Category: "Техника", ListingNumber: "1234567890", Shows: 100, Views: 20, Contacts: 4, Favorite: 3, Promotion: 50, ViewersCost: 10, TargetViewers: 12, ViewWithMessage: 1, LookPhone: 2},
		{Region: "Московская область", City: "Химки", Name: "Заголовок Б", SubCategory: "Подкат Б", Category: "Техника", ListingNumber: "1234567891", Shows: 50, Views: 10, Contacts: 2, Promotion: 20, ViewersCost: 5, LookPhone: 1},
	}}
	var buf bytes.Buffer
	if err := ExportXLSX([]StoredReport{report}, &buf); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) != 1 || got[0] != "Сводка" {
		t.Fatalf("листы: %v", got)
	}
	rows, err := f.GetRows("Сводка")
	if err != nil {
		t.Fatal(err)
	}
	find := func(label string) int {
		t.Helper()
		for i, r := range rows {
			for _, cell := range r {
				if cell == label {
					return i + 1
				}
			}
		}
		t.Fatalf("нет строки %q", label)
		return 0
	}
	value := func(col string, row int) string {
		t.Helper()
		v, e := f.GetCellValue("Сводка", col+strconv.Itoa(row), excelize.Options{RawCellValue: true})
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	if value("B", find("Период отчёта")) != "26.10.2026–01.11.2026" {
		t.Errorf("период: %s", value("B", find("Период отчёта")))
	}
	h := find("Показатель")
	for col, want := range map[string]string{"B": "Текущая неделя", "C": "Прошлая неделя", "D": "Изменение", "E": "Изменение, %"} {
		if v := value(col, h); v != want {
			t.Errorf("%s%d=%q", col, h, v)
		}
	}
	for _, test := range []struct{ label, want string }{{"Показы", "150"}, {"Просмотры", "30"}, {"Контакты", "6"}, {"Звонки", ""}, {"Расход, ₽", "85"}} {
		if v := value("B", find(test.label)); v != test.want {
			t.Errorf("%s: %s != %s", test.label, v, test.want)
		}
	}
	for _, label := range []string{"CTR: просмотры / показы", "CR: контакты / просмотры", "Стоимость контакта, ₽"} {
		if v := value("B", find(label)); v == "" {
			t.Errorf("пустой KPI %s", label)
		}
	}
	for _, label := range []string{"Показы", "Просмотры", "Контакты", "Звонки", "Расход, ₽", "CTR: просмотры / показы", "CR: контакты / просмотры", "Стоимость контакта, ₽"} {
		r := find(label)
		for _, col := range []string{"C", "D", "E"} {
			if v := value(col, r); v != "" {
				t.Errorf("%s%d не пусто: %q", col, r, v)
			}
			if form, _ := f.GetCellFormula("Сводка", col+strconv.Itoa(r)); form != "" {
				t.Errorf("формула %s%d: %s", col, r, form)
			}
		}
	}
	for _, label := range []string{"Сделано", "Планируется", "Выводы"} {
		r := find(label)
		if v := value("G", r+1); v != "" {
			t.Errorf("%s заполнено: %q", label, v)
		}
	}
	for _, label := range []string{"По регионам", "По городам", "По заголовкам", "По категориям", "По подкатегориям", "Топ-10 объявлений по контактам"} {
		find(label)
	}
	r := find("По регионам")
	if value("A", r+1) != "Регион" || value("J", r+1) != "Написали в чат" || value("K", r+1) != "Смотрели телефон" || value("L", r+1) != "Цена просмотра, ₽" || value("M", r+1) != "" {
		t.Fatal("некорректная шапка региона")
	}
	if value("A", r+2) != "Московская область" || value("B", r+2) != "150" || value("D", r+2) != "30" || value("F", r+2) != "6" || value("G", r+2) != "85" || value("J", r+2) != "1" || value("K", r+2) != "3" || value("M", r+2) != "" {
		t.Errorf("данные региона: %v", rows[r+1])
	}
	for col, want := range map[string]float64{"C": 0.2, "E": 0.2, "H": 85.0 / 6, "L": 85.0 / 30} {
		v := value(col, r+2)
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || math.Abs(n-want) > 0.001 {
			t.Errorf("%s%d=%q want %.4f", col, r+2, v, want)
		}
	}
	if value("A", find("По подкатегориям")+2) != "Подкат А" {
		t.Error("подкатегория должна браться из поля SubCategory")
	}
	if value("A", find("Топ-10 объявлений по контактам")+2) != "1234567890" {
		t.Error("номер объявления потерян")
	}
	for _, r := range rows {
		for _, v := range r {
			if strings.Contains(v, "Отклики") || strings.Contains(v, "Ср. цена отклика") || v == "Продвижение, ₽" || v == "Целевые просмотры" {
				t.Fatal("лишняя колонка в обычном отчёте")
			}
		}
	}
	styleTitle, _ := f.GetCellStyle("Сводка", "A"+strconv.Itoa(find("По регионам")))
	styleHead, _ := f.GetCellStyle("Сводка", "A"+strconv.Itoa(find("По регионам")+1))
	if styleTitle == 0 || styleHead == 0 || styleTitle == styleHead {
		t.Error("отсутствуют стили заголовка/таблицы")
	}
	for _, check := range []struct {
		cell, color string
	}{
		{"A" + strconv.Itoa(find("По регионам")), "217C82"},
		{"A" + strconv.Itoa(find("По регионам")+1), "23324D"},
		{"B" + strconv.Itoa(find("Показы")), "FFF4DA"},
	} {
		id, err := f.GetCellStyle("Сводка", check.cell)
		if err != nil {
			t.Fatal(err)
		}
		style, err := f.GetStyle(id)
		if err != nil {
			t.Fatal(err)
		}
		if len(style.Fill.Color) == 0 || !strings.EqualFold(strings.TrimPrefix(style.Fill.Color[0], "#"), check.color) {
			t.Errorf("стиль %s: заливка %v вместо %s", check.cell, style.Fill.Color, check.color)
		}
	}
}

func TestWeeklyExportExpandsSectionsAndDoesNotInventPeriod(t *testing.T) {
	offers := make([]models.Offer, 16)
	for i := range offers {
		offers[i] = models.Offer{Region: "Регион " + strconv.Itoa(i), City: "Город", Views: 1, Shows: 2}
	}
	var buf bytes.Buffer
	if err := ExportXLSX([]StoredReport{
		{FileName: "без_дат.xlsx", ReportType: models.ReportTypeRegular, Offers: offers},
		{FileName: "другой.xlsx", ReportType: models.ReportTypeRegular, Offers: offers[:1]},
	}, &buf); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if period, err := f.GetCellValue("Сводка", "B3"); err != nil || period != "" {
		t.Fatalf("период без даты в названии должен быть пустым: %q %v", period, err)
	}
	rows, err := f.GetRows("Сводка")
	if err != nil {
		t.Fatal(err)
	}
	count, titles := 0, 0
	for _, r := range rows {
		if len(r) == 0 {
			continue
		}
		if strings.HasPrefix(r[0], "Еженедельный отчёт — ") {
			titles++
		}
		if strings.HasPrefix(r[0], "Регион ") {
			count++
		}
	}
	if titles != 2 || count != 17 {
		t.Fatalf("потеряны группы или второй отчёт: заголовков %d, регионов %d", titles, count)
	}
}

func TestWeeklyExportUndefinedPricesAreBlank(t *testing.T) {
	var buf bytes.Buffer
	report := StoredReport{ReportType: models.ReportTypeRegular, Offers: []models.Offer{{
		Region: "Москва", City: "Москва", ListingNumber: "123", Shows: 10, Views: 0,
		Contacts: 0, Promotion: 100,
	}}}
	if err := ExportXLSX([]StoredReport{report}, &buf); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, ref := range []string{"B14", "E25", "H25", "N25"} {
		v, err := f.GetCellValue("Сводка", ref, excelize.Options{RawCellValue: true})
		if err != nil || v != "" {
			t.Errorf("неопределённая цена/конверсия %s = %q, err %v", ref, v, err)
		}
	}
	if v, err := f.GetCellValue("Сводка", "G25", excelize.Options{RawCellValue: true}); err != nil || v != "100" {
		t.Errorf("расход не должен пропадать: %q %v", v, err)
	}
}

func TestMixedExportKeepsHRSheetWidths(t *testing.T) {
	hr := StoredReport{FileName: "hr.xlsx", ReportType: models.ReportTypeHR, Offers: []models.Offer{{City: "Город", Employee: "Иван", Shows: 1}}}
	regular := StoredReport{FileName: "regular.xlsx", ReportType: models.ReportTypeRegular, Offers: []models.Offer{{City: "Город", Shows: 2}}}
	widths := func(reports []StoredReport) float64 {
		t.Helper()
		var out bytes.Buffer
		if err := ExportXLSX(reports, &out); err != nil {
			t.Fatal(err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(out.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		w, err := f.GetColWidth("Сводка", "A")
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	want := widths([]StoredReport{hr})
	for _, reports := range [][]StoredReport{{hr, regular}, {regular, hr}} {
		if got := widths(reports); got != want {
			t.Errorf("mixed ширина A = %v, HR-only = %v", got, want)
		}
	}
}
