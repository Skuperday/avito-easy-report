package handler

import (
	"avito-easy-report/internal/service"
	models "avito-easy-report/internal/struct"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func TestParseReportTypeCanonicalValues(t *testing.T) {
	tests := []struct {
		input string
		want  models.ReportType
		ok    bool
	}{
		{input: "", want: models.ReportTypeRegular, ok: true},
		{input: "regular", want: models.ReportTypeRegular, ok: true},
		{input: "avito", want: models.ReportTypeRegular, ok: true},
		{input: "hr", want: models.ReportTypeHR, ok: true},
		{input: "unknown", ok: false},
	}
	for _, test := range tests {
		got, ok := parseReportType(test.input)
		if ok != test.ok || got != test.want {
			t.Errorf("parseReportType(%q) = %q, %v; ожидалось %q, %v", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestUploadReportRejectsUnknownType(t *testing.T) {
	store := service.NewReportStore()
	handler := NewHandler(store, service.NewObjectStore())
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = reportUploadRequest(t, "unknown")

	handler.UploadReport(context)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("неизвестный тип должен вернуть 400, получено %d: %s", response.Code, response.Body.String())
	}
	if store.Count() != 0 {
		t.Fatalf("отчёт с неизвестным типом не должен сохраняться")
	}
}

func TestMultiStatsReturnsReportType(t *testing.T) {
	store := service.NewReportStore()
	store.Add("regular-report", &service.StoredReport{
		ID:         "regular-report",
		FileName:   "regular.xlsx",
		ReportType: models.ReportTypeRegular,
		UserID:     42,
	})
	handler := NewHandler(store, service.NewObjectStore())
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/reports/multi?ids=regular-report", nil)
	context.Set("claims", &service.Claims{UserID: 42})

	handler.MultiStats(context)

	var body struct {
		Reports []map[string]any `json:"reports"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode multi stats: %v", err)
	}
	if len(body.Reports) != 1 || body.Reports[0]["reportType"] != "regular" {
		t.Fatalf("multi stats должны вернуть regular-тип: %#v", body.Reports)
	}
}

func TestStatsResponseReturnsReportType(t *testing.T) {
	store := service.NewReportStore()
	store.Add("hr-report", &service.StoredReport{
		ID:         "hr-report",
		FileName:   "hr.xlsx",
		ReportType: models.ReportTypeHR,
		UserID:     42,
	})
	handler := NewHandler(store, service.NewObjectStore())
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Params = gin.Params{{Key: "id", Value: "hr-report"}}
	context.Set("claims", &service.Claims{UserID: 42})

	handler.GetStats(context)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if body["reportType"] != "hr" {
		t.Fatalf("stats должны вернуть HR-тип: %#v", body)
	}
}

func TestCabinetReportsReturnsReportType(t *testing.T) {
	cabinetStore := service.NewCabinetStore()
	cabinet := cabinetStore.Create("Основной", 42)
	reportStore := service.NewReportStore()
	reportStore.Add("regular-report", &service.StoredReport{
		ID:         "regular-report",
		FileName:   "regular.xlsx",
		ReportType: models.ReportTypeRegular,
		UserID:     42,
		CabinetID:  cabinet.ID,
	})
	handler := NewCabinetHandler(cabinetStore, reportStore)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Params = gin.Params{{Key: "id", Value: cabinet.ID}}
	context.Set("claims", &service.Claims{UserID: 42})

	handler.ListReports(context)

	var reports []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &reports); err != nil {
		t.Fatalf("decode cabinet reports: %v", err)
	}
	if len(reports) != 1 || reports[0]["reportType"] != "regular" {
		t.Fatalf("список кабинета должен вернуть regular-тип: %#v", reports)
	}
}

func TestListReportsReturnsReportType(t *testing.T) {
	store := service.NewReportStore()
	store.Add("hr-report", &service.StoredReport{
		ID:         "hr-report",
		FileName:   "hr.xlsx",
		ReportType: models.ReportTypeHR,
		UserID:     42,
	})
	handler := NewHandler(store, service.NewObjectStore())
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("claims", &service.Claims{UserID: 42})

	handler.ListReports(context)

	var reports []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &reports); err != nil {
		t.Fatalf("decode reports: %v", err)
	}
	if len(reports) != 1 || reports[0]["reportType"] != "hr" {
		t.Fatalf("список должен вернуть HR-тип отчёта: %#v", reports)
	}
}

func TestUploadReportStoresAndReturnsHRType(t *testing.T) {
	request := reportUploadRequest(t, "hr")
	store := service.NewReportStore()
	handler := NewHandler(store, service.NewObjectStore())

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request
	handler.UploadReport(context)

	if response.Code != http.StatusOK {
		t.Fatalf("HR upload должен вернуть 200, получено %d: %s", response.Code, response.Body.String())
	}
	reports := store.List()
	if len(reports) != 1 {
		t.Fatalf("ожидался один сохранённый отчёт, получено %d", len(reports))
	}
	if reports[0].ReportType != models.ReportTypeHR {
		t.Fatalf("выбранный HR-тип не сохранён: %#v", reports[0])
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["reportType"] != "hr" {
		t.Fatalf("API должен вернуть reportType=hr, получено %#v", body["reportType"])
	}
}

func reportUploadRequest(t *testing.T, reportType string) *http.Request {
	t.Helper()
	workbook := excelize.NewFile()
	if err := workbook.SetSheetRow("Sheet1", "A1", &[]any{"Город"}); err != nil {
		t.Fatalf("header: %v", err)
	}
	if err := workbook.SetSheetRow("Sheet1", "A2", &[]any{"Москва"}); err != nil {
		t.Fatalf("row: %v", err)
	}
	var xlsx bytes.Buffer
	if err := workbook.Write(&xlsx); err != nil {
		t.Fatalf("workbook: %v", err)
	}
	_ = workbook.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("type", reportType); err != nil {
		t.Fatalf("type field: %v", err)
	}
	part, err := writer.CreateFormFile("file", "report.xlsx")
	if err != nil {
		t.Fatalf("file field: %v", err)
	}
	if _, err := part.Write(xlsx.Bytes()); err != nil {
		t.Fatalf("file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
