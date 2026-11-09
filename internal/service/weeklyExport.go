package service

import (
	models "avito-easy-report/internal/struct"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"
)

var weeklyFilePeriod = regexp.MustCompile(`(?i)_с_(\d{4}-\d{2}-\d{2})_по_(\d{4}-\d{2}-\d{2})`)

// writeWeeklyReport оформляет только обычный XLSX по еженедельному образцу.
// Никакие данные прошлого периода или комментарии не подставляются автоматически.
func writeWeeklyReport(f *excelize.File, sheet string, row *int, report StoredReport, regularOnly bool) error {
	const (
		navy   = "#23324D"
		teal   = "#217C82"
		pale   = "#F2F5FA"
		yellow = "#FFF4DA"
		white  = "#FFFFFF"
	)
	newStyle := func(color, fontColor string, bold bool, format int) (int, error) {
		style := &excelize.Style{Font: &excelize.Font{Bold: bold, Color: fontColor, Family: "Calibri", Size: 11},
			Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}}
		if color != "" {
			style.Fill = excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1}
		}
		style.NumFmt = format
		return f.NewStyle(style)
	}
	section, err := newStyle(teal, white, true, 0)
	if err != nil {
		return err
	}
	header, err := newStyle(navy, white, true, 0)
	if err != nil {
		return err
	}
	stripe, err := newStyle(pale, navy, false, 0)
	if err != nil {
		return err
	}
	plain, err := newStyle("", navy, false, 0)
	if err != nil {
		return err
	}
	input, err := newStyle(yellow, navy, false, 0)
	if err != nil {
		return err
	}
	count, err := newStyle(yellow, navy, false, 3)
	if err != nil {
		return err
	}
	percent, err := newStyle(yellow, navy, false, 10)
	if err != nil {
		return err
	}
	stripePercent, err := newStyle(pale, navy, false, 10)
	if err != nil {
		return err
	}
	plainPercent, err := newStyle("", navy, false, 10)
	if err != nil {
		return err
	}
	moneyFormat := `#,##0.00" ₽"`
	makeMoney := func(fill string) (int, error) {
		style := &excelize.Style{Font: &excelize.Font{Color: navy, Family: "Calibri", Size: 11}, CustomNumFmt: &moneyFormat}
		if fill != "" {
			style.Fill = excelize.Fill{Type: "pattern", Color: []string{fill}, Pattern: 1}
		}
		return f.NewStyle(style)
	}
	money, err := makeMoney(yellow)
	if err != nil {
		return err
	}
	stripeMoney, err := makeMoney(pale)
	if err != nil {
		return err
	}
	plainMoney, err := makeMoney("")
	if err != nil {
		return err
	}
	cell := func(col, r int) string { c, _ := excelize.CoordinatesToCellName(col, r); return c }
	write := func(col, r int, v interface{}) error { return f.SetCellValue(sheet, cell(col, r), v) }
	setStyle := func(c1, c2, r, style int) error { return f.SetCellStyle(sheet, cell(c1, r), cell(c2, r), style) }
	// Одна общая вкладка: не меняем её ширины и вид для соседнего HR-отчёта.
	if regularOnly {
		if err = f.SetColWidth(sheet, "A", "A", 38); err != nil {
			return err
		}
		if err = f.SetColWidth(sheet, "B", "B", 18); err != nil {
			return err
		}
		if err = f.SetColWidth(sheet, "C", "L", 20); err != nil {
			return err
		}
		showGridLines := false
		if err = f.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: &showGridLines}); err != nil {
			return err
		}
	}
	start := *row
	if err = f.MergeCell(sheet, cell(1, start+1), cell(14, start+1)); err != nil {
		return err
	}
	if err = write(1, start+1, "Еженедельный отчёт — "+report.FileName); err != nil {
		return err
	}
	if err = setStyle(1, 14, start+1, section); err != nil {
		return err
	}
	if err = f.SetRowHeight(sheet, start+1, 28); err != nil {
		return err
	}
	if err = write(1, start+2, "Период отчёта"); err != nil {
		return err
	}
	if period := periodFromFileName(report.FileName); period != "" {
		if err = write(2, start+2, period); err != nil {
			return err
		}
	}
	if err = write(1, start+3, "Лист прошлой недели"); err != nil {
		return err
	}
	if err = setStyle(2, 2, start+2, input); err != nil {
		return err
	}
	if err = setStyle(2, 2, start+3, input); err != nil {
		return err
	}
	headRow := start + 5
	for col, h := range []string{"Показатель", "Текущая неделя", "Прошлая неделя", "Изменение", "Изменение, %"} {
		if err = write(col+1, headRow, h); err != nil {
			return err
		}
		if err = setStyle(col+1, col+1, headRow, header); err != nil {
			return err
		}
	}
	if err = f.SetRowHeight(sheet, headRow, 30); err != nil {
		return err
	}
	for _, item := range []struct {
		row   int
		label string
	}{
		{headRow, "Сделано"}, {headRow + 5, "Планируется"}, {headRow + 10, "Выводы"},
	} {
		if err = f.MergeCell(sheet, cell(7, item.row), cell(14, item.row)); err != nil {
			return err
		}
		if err = write(7, item.row, item.label); err != nil {
			return err
		}
		if err = setStyle(7, 14, item.row, section); err != nil {
			return err
		}
		if err = f.MergeCell(sheet, cell(7, item.row+1), cell(14, item.row+3)); err != nil {
			return err
		}
		if err = setStyle(7, 14, item.row+1, input); err != nil {
			return err
		}
	}
	var shows, views, contacts int
	var expense float64
	for _, o := range report.Offers {
		shows += o.Shows
		views += o.Views
		contacts += o.Contacts

		expense += o.Promotion + o.ViewersCost
	}
	kpis := []struct {
		label  string
		value  interface{}
		format int
	}{
		{"Показы", shows, count}, {"Просмотры", views, count}, {"Контакты", contacts, count},
		// В исходном Avito-файле есть «Посмотрели телефон», но это не число звонков.
		{"Звонки", nil, count}, {"Расход, ₽", expense, money},
		{"CTR: просмотры / показы", ratio(views, shows), percent},
		{"CR: контакты / просмотры", ratio(contacts, views), percent},
		{"Стоимость контакта, ₽", ratioFloat(expense, contacts), money},
	}
	for i, k := range kpis {
		r := headRow + 1 + i
		bg := plain
		if i%2 == 0 {
			bg = stripe
		}
		if err = setStyle(1, 5, r, bg); err != nil {
			return err
		}
		if err = write(1, r, k.label); err != nil {
			return err
		}
		if k.value != nil {
			if err = write(2, r, k.value); err != nil {
				return err
			}
		}
		if err = setStyle(2, 2, r, k.format); err != nil {
			return err
		}
		// C:E remain physically empty (no cross-sheet links or change formulas).
	}
	current := headRow + 17
	regularHeaders := []string{"", "Показы", "CTR", "Просмотры", "CR", "Контакты", "Расход, ₽", "Цена контакта, ₽", "Избранное", "Написали в чат", "Смотрели телефон", "Цена просмотра, ₽"}
	writeSection := func(title, dimension string, stats []models.ResultStats, top bool) error {
		if !top {
			sort.Slice(stats, func(i, j int) bool {
				if stats[i].Contacts != stats[j].Contacts {
					return stats[i].Contacts > stats[j].Contacts
				}
				return stats[i].Key < stats[j].Key
			})
		}
		if err = f.MergeCell(sheet, cell(1, current), cell(len(regularHeaders), current)); err != nil {
			return err
		}
		if err = write(1, current, title); err != nil {
			return err
		}
		if err = setStyle(1, len(regularHeaders), current, section); err != nil {
			return err
		}
		if err = f.SetRowHeight(sheet, current, 25); err != nil {
			return err
		}
		current++
		headers := regularHeaders
		if top {
			headers = []string{"Номер объявления", "Город", "Показы", "CTR", "Просмотры", "CR", "Контакты", "Расход, ₽", "Цена контакта, ₽", "Избранное"}
		}
		for c, h := range headers {
			if c == 0 && !top {
				h = dimension
			}
			if err = write(c+1, current, h); err != nil {
				return err
			}
			if err = setStyle(c+1, c+1, current, header); err != nil {
				return err
			}
		}
		if err = f.SetRowHeight(sheet, current, 32); err != nil {
			return err
		}
		current++
		for i, s := range stats {
			bg := plain
			pctStyle, cashStyle := plainPercent, plainMoney
			if i%2 == 0 {
				bg = stripe
				pctStyle, cashStyle = stripePercent, stripeMoney
			}
			if err = setStyle(1, len(headers), current, bg); err != nil {
				return err
			}
			values := []interface{}{s.Key, s.Shows, ratio(s.Views, s.Shows), s.Views, ratio(s.Contacts, s.Views), s.Contacts, s.Expense, ratioFloat(s.Expense, s.Contacts), s.Favorite, s.ViewWithMessage, s.LookPhone, ratioFloat(s.Expense, s.Views)}
			if top {
				values = []interface{}{s.Key, s.City, s.Shows, ratio(s.Views, s.Shows), s.Views, ratio(s.Contacts, s.Views), s.Contacts, s.Expense, ratioFloat(s.Expense, s.Contacts), s.Favorite}
			}
			for c, v := range values {
				if v != nil {
					if err = write(c+1, current, v); err != nil {
						return err
					}
				}
				style := 0
				if top {
					switch c {
					case 3, 5:
						style = pctStyle
					case 7, 8:
						style = cashStyle
					}
				} else {
					switch c {
					case 2, 4:
						style = pctStyle
					case 6, 7, 11:
						style = cashStyle
					}
				}
				if style != 0 {
					if err = setStyle(c+1, c+1, current, style); err != nil {
						return err
					}
				}
			}
			current++
		}
		current += 2
		return nil
	}
	for _, sec := range []struct{ title, dimension, group string }{
		{"По регионам", "Регион", "region"}, {"По городам", "Город", "city"},
		{"По заголовкам", "Заголовок", "name"}, {"По категориям", "Категория", "category"},
		{"По подкатегориям", "Подкатегория", "subcategory"},
	} {
		if err = writeSection(sec.title, sec.dimension, GetStatsForGroup(report.Offers, sec.group), false); err != nil {
			return err
		}
	}
	if err = writeSection("Топ-10 объявлений по контактам", "", GetTopListings(report.Offers, 10), true); err != nil {
		return err
	}
	*row = current + 1
	return nil
}

func ratio(n, d int) interface{} {
	if d == 0 {
		return nil
	}
	return float64(n) / float64(d)
}
func ratioFloat(n float64, d int) interface{} {
	if d == 0 {
		return nil
	}
	return n / float64(d)
}

func periodFromFileName(name string) string {
	match := weeklyFilePeriod.FindStringSubmatch(name)
	if match == nil {
		return ""
	}
	start, e1 := time.Parse("2006-01-02", match[1])
	end, e2 := time.Parse("2006-01-02", match[2])
	if e1 != nil || e2 != nil || end.Before(start) {
		return ""
	}
	return fmt.Sprintf("%s–%s", start.Format("02.01.2006"), end.Format("02.01.2006"))
}
