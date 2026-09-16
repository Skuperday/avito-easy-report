package handler

import (
	"avito-easy-report/internal/service"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func reportTestRouter() (*gin.Engine, *service.ReportStore) {
	gin.SetMode(gin.TestMode)
	store := service.NewReportStore()
	h := NewHandler(store, service.NewObjectStore())
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("claims", &service.Claims{UserID: 1}) })
	r.POST("/upload", h.UploadReport)
	r.GET("/reports/multi", h.MultiStats)
	r.GET("/reports/compare", h.CompareReports)
	r.GET("/reports/:id/stats", h.GetStats)
	r.GET("/export", h.ExportAll)
	return r, store
}

func uploadTestReport(t *testing.T, r http.Handler, reportType, name string, count int) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	headers := []any{"Сотрудник", "Объект", "Город", "Категория", "Название объявления", "Показы", "Просмотры", "Контакты", "Добавили в избранное"}
	if err := f.SetSheetRow("Sheet1", "A1", &headers); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		row := []any{"Анна", "Офис", "Москва", "Работа", "Вакансия", 10, 4, 2, 3}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow("Sheet1", cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Write(part); err != nil {
		t.Fatal(err)
	}
	if reportType != "" {
		if err := writer.WriteField("type", reportType); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.WriteField("cabinetId", "cabinet"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}
	var result struct {
		ID   string `json:"id"`
		Rows int    `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Rows != count {
		t.Fatalf("rows=%d, want %d", result.Rows, count)
	}
	return result.ID
}

func getTestJSON(t *testing.T, r http.Handler, path string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	if w.Code != 200 {
		t.Fatalf("GET %s: %d %s", path, w.Code, w.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestCompareHRListingCounts(t *testing.T) {
	for _, types := range [][2]string{{"hr", "hr"}, {"hr", "avito"}, {"avito", "hr"}, {"avito", "avito"}} {
		for _, group := range []string{"employee", "object", "city"} {
			t.Run(types[0]+"/"+types[1]+"/"+group, func(t *testing.T) {
				r, store := reportTestRouter()
				earlyID := uploadTestReport(t, r, types[0], "2026-01-01.xlsx", 2)
				lateID := uploadTestReport(t, r, types[1], "2026-02-01.xlsx", 3)
				// Disappeared and newly appearing groups, with fixtures otherwise parsed via HTTP.
				gone := store.Get(earlyID).Offers[0]
				gone.Employee = "Ушёл"
				gone.Object = "Закрыт"
				gone.City = "Ушёл"
				added := store.Get(lateID).Offers[0]
				added.Employee = "Новый"
				added.Object = "Новый"
				added.City = "Новый"
				store.Get(earlyID).Offers = append(store.Get(earlyID).Offers, gone)
				store.Get(lateID).Offers = append(store.Get(lateID).Offers, added)
				result := getTestJSON(t, r, "/reports/compare?ids="+lateID+","+earlyID+"&groupBy="+group)
				early := result["early"].(map[string]any)
				late := result["late"].(map[string]any)
				if early["reportType"] != types[0] || late["reportType"] != types[1] {
					t.Errorf("period types = %v / %v", early["reportType"], late["reportType"])
				}
				deltas := result["delta"].([]any)
				es := early["stats"].([]any)
				ls := late["stats"].([]any)
				fullHR := types[0] == "hr" && types[1] == "hr" && group != "city"
				wantLen := 2
				if fullHR {
					wantLen = 3
				}
				if len(deltas) != wantLen {
					t.Errorf("groups=%d want %d", len(deltas), wantLen)
				}
				for i, item := range deltas {
					d := item.(map[string]any)
					e := es[i].(map[string]any)
					l := ls[i].(map[string]any)
					if e["key"] != d["key"] || l["key"] != d["key"] {
						t.Fatal("period keys misaligned")
					}
					if d["listingCount"] != l["listingCount"].(float64)-e["listingCount"].(float64) {
						t.Errorf("bad count delta: %v / %v / %v", e["listingCount"], l["listingCount"], d["listingCount"])
					}
					if d["key"] == "Ушёл" || d["key"] == "Закрыт" {
						if l["listingCount"] != float64(0) || d["listingCount"] != float64(-1) {
							t.Error("missing negative delta")
						}
					}
				}
			})
		}
	}
}

func TestHRExportColumnPlacement(t *testing.T) {
	r, _ := reportTestRouter()
	hr := uploadTestReport(t, r, "hr", "HR.xlsx", 2)
	avito := uploadTestReport(t, r, "avito", "Avito.xlsx", 1)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/export?ids="+hr+","+avito, nil))
	if w.Code != 200 {
		t.Fatalf("export: %d %s", w.Code, w.Body.String())
	}
	f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("Сводка")
	if err != nil {
		t.Fatal(err)
	}
	report := ""
	counts := 0
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		if row[0] == "Отчёт: HR.xlsx" || row[0] == "Отчёт: Avito.xlsx" {
			report = row[0]
		}
		if row[0] != "Сотрудник" && row[0] != "Объект" && row[0] != "Город" && row[0] != "Категория" && row[0] != "Подкатегория" && row[0] != "Номер объявления" {
			continue
		}
		hrGroup := report == "Отчёт: HR.xlsx" && (row[0] == "Сотрудник" || row[0] == "Объект")
		if hrGroup {
			counts++
			if len(row) != 13 || row[1] != "Количество объявлений" || row[2] != "Показы" {
				t.Errorf("HR header: %v", row)
				continue
			}
			data := rows[i+1]
			if len(data) != 13 || data[1] != "2" || data[2] != "20" || data[4] != "8" || data[6] != "4" || data[12] != "6" {
				t.Errorf("misaligned HR data: %v", data)
			}
		} else {
			for _, h := range row {
				if h == "Количество объявлений" {
					t.Errorf("count leaked into %s %v", report, row)
				}
			}
			shows := 1
			if row[0] == "Номер объявления" {
				shows = 2
			}
			if row[shows] != "Показы" {
				t.Errorf("shows shifted: %v", row)
			}
		}
	}
	if counts != 2 {
		t.Fatalf("HR sections=%d, want 2", counts)
	}
}

func TestUploadReportTypeAndGroupedStats(t *testing.T) {
	for _, typ := range []string{"hr", "avito", "", "unknown"} {
		t.Run("type="+typ, func(t *testing.T) {
			r, store := reportTestRouter()
			id := uploadTestReport(t, r, typ, "2026-01-01.xlsx", 2)
			wantType := "avito"
			if typ == "hr" {
				wantType = "hr"
			}
			if store.Get(id).CabinetID != "cabinet" {
				t.Fatal("cabinet lost")
			}
			for _, group := range []string{"employee", "object"} {
				single := getTestJSON(t, r, "/reports/"+id+"/stats?groupBy="+group)
				multi := getTestJSON(t, r, "/reports/multi?ids="+id+"&groupBy="+group)
				for _, result := range []map[string]any{single, multi["reports"].([]any)[0].(map[string]any)} {
					if result["reportType"] != wantType {
						t.Errorf("reportType=%v, want %s", result["reportType"], wantType)
					}
					row := result["stats"].([]any)[0].(map[string]any)
					if row["listingCount"] != float64(2) || row["shows"] != float64(20) {
						t.Fatalf("unexpected stats: %v", row)
					}
				}
			}
		})
	}
}
