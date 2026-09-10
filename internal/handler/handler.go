package handler

import (
	"avito-easy-report/internal/middleware"
	"avito-easy-report/internal/service"
	models "avito-easy-report/internal/struct"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	store       *service.ReportStore
	objectStore *service.ObjectStore
}

func NewHandler(store *service.ReportStore, objectStore *service.ObjectStore) *Handler {
	return &Handler{store: store, objectStore: objectStore}
}

func parseReportType(value string) (models.ReportType, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "regular", "avito":
		return models.ReportTypeRegular, true
	case "hr":
		return models.ReportTypeHR, true
	default:
		return "", false
	}
}

func (h *Handler) UploadReport(c *gin.Context) {
	reportType, ok := parseReportType(c.PostForm("type"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неизвестный тип отчёта"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл не найден в запросе: " + err.Error()})
		return
	}
	defer file.Close()

	offers, excelFile, warnings, foundColumns, err := service.ParseReport(file, header.Filename, h.objectStore)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims := middleware.GetClaims(c)
	userID := uint(0)
	if claims != nil {
		userID = claims.UserID
	}
	cabinetID := c.PostForm("cabinetId")

	id := uuid.New().String()
	h.store.Add(id, &service.StoredReport{
		ID:         id,
		FileName:   header.Filename,
		ReportType: reportType,
		UserID:     userID,
		CabinetID:  cabinetID,
		Offers:     offers,
		File:       excelFile,
	})

	c.JSON(http.StatusOK, models.UploadResponse{
		ID:         id,
		FileName:   header.Filename,
		ReportType: reportType,
		Rows:       len(offers),
		Warnings:   warnings,
		Columns:    foundColumns,
	})
}

func (h *Handler) ListReports(c *gin.Context) {
	claims := middleware.GetClaims(c)
	reports := h.store.ListByUser(claims.UserID)
	result := make([]models.ReportInfo, len(reports))
	for i, r := range reports {
		result[i] = models.ReportInfo{ID: r.ID, FileName: r.FileName, ReportType: r.ReportType}
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetStats(c *gin.Context) {
	id := c.Param("id")
	report := h.store.Get(id)
	if report == nil || !h.ownsReport(c, report) {
		c.JSON(http.StatusNotFound, gin.H{"error": "отчёт не найден"})
		return
	}

	groupBy := c.DefaultQuery("groupBy", "city")
	if !isGroupAllowed(report.ReportType, groupBy) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "группировка недоступна для выбранного типа отчёта"})
		return
	}
	resultStats := service.GetStatsForGroup(report.Offers, groupBy)
	summary := service.GetSummary(report.Offers)

	c.JSON(http.StatusOK, models.StatsResponse{
		ReportID:   report.ID,
		FileName:   report.FileName,
		ReportType: report.ReportType,
		Stats:      resultStats,
		Summary:    summary,
	})
}

func (h *Handler) DeleteReport(c *gin.Context) {
	id := c.Param("id")
	report := h.store.Get(id)
	if report == nil || !h.ownsReport(c, report) {
		c.JSON(http.StatusNotFound, gin.H{"error": "отчёт не найден"})
		return
	}
	h.store.Remove(id)
	c.JSON(http.StatusOK, gin.H{"status": "удалён"})
}

func (h *Handler) ExportAll(c *gin.Context) {
	claims := middleware.GetClaims(c)
	allReports := h.store.ListByUser(claims.UserID)

	// Фильтр по ids, если передан
	var reports []service.StoredReport
	if idsStr := c.Query("ids"); idsStr != "" {
		idSet := make(map[string]bool)
		for _, id := range strings.Split(idsStr, ",") {
			idSet[strings.TrimSpace(id)] = true
		}
		for _, r := range allReports {
			if idSet[r.ID] {
				reports = append(reports, r)
			}
		}
	} else {
		reports = allReports
	}

	if len(reports) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нет загруженных отчётов"})
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=result.xlsx")
	if err := service.ExportXLSX(reports, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ошибка экспорта: %v", err)})
	}
}

func (h *Handler) MultiStats(c *gin.Context) {
	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "укажите ids отчётов через запятую"})
		return
	}
	ids := strings.Split(idsStr, ",")
	groupBy := c.DefaultQuery("groupBy", "city")

	var result []models.StatsResponse
	for _, id := range ids {
		id = strings.TrimSpace(id)
		report := h.store.Get(id)
		if report == nil || !h.ownsReport(c, report) {
			continue
		}
		if !isGroupAllowed(report.ReportType, groupBy) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "группировка недоступна для выбранного типа отчёта"})
			return
		}
		resultStats := service.GetStatsForGroup(report.Offers, groupBy)
		summary := service.GetSummary(report.Offers)
		result = append(result, models.StatsResponse{
			ReportID: report.ID, FileName: report.FileName, ReportType: report.ReportType,
			Stats: resultStats, Summary: summary,
		})
	}

	c.JSON(http.StatusOK, models.MultiStatsResponse{Reports: result})
}

func (h *Handler) CompareReports(c *gin.Context) {
	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "укажите ids отчётов через запятую"})
		return
	}
	ids := strings.Split(idsStr, ",")
	if len(ids) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нужно минимум 2 отчёта для сравнения"})
		return
	}

	groupBy := c.DefaultQuery("groupBy", "city")

	// Для режима "объявления" сравнение идёт по названию объявления
	compareGroupBy := groupBy
	if groupBy == "offers" {
		compareGroupBy = "name"
	}

	type indexedReport struct {
		id         string
		offers     []models.Offer
		reportType models.ReportType
		created    time.Time
	}
	var reports []indexedReport
	for _, id := range ids {
		id = strings.TrimSpace(id)
		r := h.store.Get(id)
		if r == nil || !h.ownsReport(c, r) {
			continue
		}
		if !isCompareGroupAllowed(r.ReportType, groupBy) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "группировка недоступна для выбранного типа отчёта"})
			return
		}
		t := parseDateFromFilename(r.FileName)
		reports = append(reports, indexedReport{id: r.ID, offers: r.Offers, reportType: r.ReportType, created: t})
	}

	if len(reports) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "недостаточно отчётов для сравнения"})
		return
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].created.Before(reports[j].created) })

	early := reports[0].offers
	late := reports[len(reports)-1].offers

	result := service.ComparePeriods(early, late, compareGroupBy)
	result.ReportTypes = make([]models.ReportType, len(reports))
	for i, report := range reports {
		result.ReportTypes[i] = report.reportType
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ownsReport(c *gin.Context, report *service.StoredReport) bool {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return false
	}
	return report.UserID == claims.UserID
}

func isGroupAllowed(reportType models.ReportType, groupBy string) bool {
	switch groupBy {
	case "city", "category", "name", "offers":
		return true
	case "employee", "object", "employee-object":
		return reportType == models.ReportTypeHR
	default:
		return false
	}
}

func isCompareGroupAllowed(reportType models.ReportType, groupBy string) bool {
	return groupBy != "employee-object" && isGroupAllowed(reportType, groupBy)
}

func parseDateFromFilename(name string) time.Time {
	for i := 0; i < len(name)-9; i++ {
		if len(name)-i >= 10 && name[i] >= '0' && name[i] <= '9' {
			if t, err := time.Parse("2006-01-02", name[i:i+10]); err == nil {
				return t
			}
		}
	}
	return time.Now()
}
