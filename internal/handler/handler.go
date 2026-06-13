package handler

import (
	models "avito-easy-report/internal/struct"
	"avito-easy-report/internal/service"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler — обработчики HTTP-запросов
type Handler struct {
	store *service.ReportStore
}

// NewHandler создаёт новый Handler
func NewHandler(store *service.ReportStore) *Handler {
	return &Handler{store: store}
}

// UploadReport — POST /api/upload
// Принимает multipart/form-data с полем "file"
func (h *Handler) UploadReport(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл не найден в запросе: " + err.Error()})
		return
	}
	defer file.Close()

	offers, excelFile, warnings, err := service.ParseReport(file, header.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	h.store.Add(id, &service.StoredReport{
		ID:       id,
		FileName: header.Filename,
		Offers:   offers,
		File:     excelFile,
	})

	c.JSON(http.StatusOK, models.UploadResponse{
		ID:       id,
		FileName: header.Filename,
		Rows:     len(offers),
		Warnings: warnings,
	})
}

// ListReports — GET /api/reports
func (h *Handler) ListReports(c *gin.Context) {
	reports := h.store.List()
	result := make([]models.ReportInfo, len(reports))
	for i, r := range reports {
		result[i] = models.ReportInfo{
			ID:       r.ID,
			FileName: r.FileName,
		}
	}
	c.JSON(http.StatusOK, result)
}

// GetStats — GET /api/reports/:id/stats
func (h *Handler) GetStats(c *gin.Context) {
	id := c.Param("id")
	report := h.store.Get(id)
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "отчёт не найден"})
		return
	}

	stats := service.GetSimpleStatMap(report.Offers)
	resultStats := service.GetResultStats(stats)

	c.JSON(http.StatusOK, models.StatsResponse{
		ReportID: report.ID,
		FileName: report.FileName,
		Stats:    resultStats,
	})
}

// DeleteReport — DELETE /api/reports/:id
func (h *Handler) DeleteReport(c *gin.Context) {
	id := c.Param("id")
	if h.store.Get(id) == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "отчёт не найден"})
		return
	}
	h.store.Remove(id)
	c.JSON(http.StatusOK, gin.H{"status": "удалён"})
}

// ExportAll — GET /api/export
func (h *Handler) ExportAll(c *gin.Context) {
	reports := h.store.List()
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

// MultiStats — GET /api/reports/multi?ids=id1,id2
func (h *Handler) MultiStats(c *gin.Context) {
	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "укажите ids отчётов через запятую"})
		return
	}
	ids := strings.Split(idsStr, ",")

	var result []models.StatsResponse
	for _, id := range ids {
		id = strings.TrimSpace(id)
		report := h.store.Get(id)
		if report == nil {
			continue
		}
		statsMap := service.GetSimpleStatMap(report.Offers)
		resultStats := service.GetResultStats(statsMap)
		result = append(result, models.StatsResponse{
			ReportID: report.ID,
			FileName: report.FileName,
			Stats:    resultStats,
		})
	}

	c.JSON(http.StatusOK, models.MultiStatsResponse{Reports: result})
}
