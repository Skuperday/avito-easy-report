package handler

import (
	models "avito-easy-report/internal/struct"
	"avito-easy-report/internal/middleware"
	"avito-easy-report/internal/service"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	store *service.ReportStore
}

func NewHandler(store *service.ReportStore) *Handler { return &Handler{store: store} }

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

	claims := middleware.GetClaims(c)
	userID := uint(0)
	if claims != nil {
		userID = claims.UserID
	}
	cabinetID := c.PostForm("cabinetId")

	id := uuid.New().String()
	h.store.Add(id, &service.StoredReport{
		ID:        id,
		FileName:  header.Filename,
		UserID:    userID,
		CabinetID: cabinetID,
		Offers:    offers,
		File:      excelFile,
	})

	c.JSON(http.StatusOK, models.UploadResponse{
		ID:       id,
		FileName: header.Filename,
		Rows:     len(offers),
		Warnings: warnings,
	})
}

func (h *Handler) ListReports(c *gin.Context) {
	claims := middleware.GetClaims(c)
	reports := h.store.ListByUser(claims.UserID)
	result := make([]models.ReportInfo, len(reports))
	for i, r := range reports {
		result[i] = models.ReportInfo{ID: r.ID, FileName: r.FileName}
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
	statsMap := service.GetGroupedStats(report.Offers, groupBy)
	resultStats := service.GetResultStats(statsMap)
	summary := service.GetSummary(report.Offers)

	c.JSON(http.StatusOK, models.StatsResponse{
		ReportID: report.ID,
		FileName: report.FileName,
		Stats:    resultStats,
		Summary:  summary,
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
	reports := h.store.ListByUser(claims.UserID)
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
		statsMap := service.GetGroupedStats(report.Offers, groupBy)
		resultStats := service.GetResultStats(statsMap)
		summary := service.GetSummary(report.Offers)
		result = append(result, models.StatsResponse{
			ReportID: report.ID, FileName: report.FileName,
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

	type indexedReport struct {
		id      string
		offers  []models.Offer
		created time.Time
	}
	var reports []indexedReport
	for _, id := range ids {
		id = strings.TrimSpace(id)
		r := h.store.Get(id)
		if r == nil || !h.ownsReport(c, r) {
			continue
		}
		t := parseDateFromFilename(r.FileName)
		reports = append(reports, indexedReport{id: r.ID, offers: r.Offers, created: t})
	}

	if len(reports) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "недостаточно отчётов для сравнения"})
		return
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].created.Before(reports[j].created) })

	early := reports[0].offers
	late := reports[len(reports)-1].offers

	result := service.ComparePeriods(early, late, groupBy)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ownsReport(c *gin.Context, report *service.StoredReport) bool {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return false
	}
	return report.UserID == claims.UserID
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
