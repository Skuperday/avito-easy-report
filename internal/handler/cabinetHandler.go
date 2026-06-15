package handler

import (
	"avito-easy-report/internal/middleware"
	"avito-easy-report/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CabinetHandler struct {
	cabinets *service.CabinetStore
	reports  *service.ReportStore
}

func NewCabinetHandler(c *service.CabinetStore, r *service.ReportStore) *CabinetHandler {
	return &CabinetHandler{cabinets: c, reports: r}
}

type createCabinetReq struct {
	Name string `json:"name" binding:"required"`
}

func (h *CabinetHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	cabinets := h.cabinets.ListByUser(claims.UserID)
	if cabinets == nil {
		cabinets = []service.Cabinet{}
	}
	c.JSON(http.StatusOK, cabinets)
}

func (h *CabinetHandler) Create(c *gin.Context) {
	var req createCabinetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "имя кабинета обязательно"})
		return
	}
	claims := middleware.GetClaims(c)
	cab := h.cabinets.Create(req.Name, claims.UserID)
	c.JSON(http.StatusCreated, cab)
}

func (h *CabinetHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	claims := middleware.GetClaims(c)
	if !h.cabinets.Delete(id, claims.UserID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "кабинет не найден"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "удалён"})
}

// ListReports — отчёты в конкретном кабинете
func (h *CabinetHandler) ListReports(c *gin.Context) {
	cabID := c.Param("id")
	cab := h.cabinets.Get(cabID)
	if cab == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "кабинет не найден"})
		return
	}
	claims := middleware.GetClaims(c)
	if cab.UserID != claims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ запрещён"})
		return
	}
	// Пока возвращаем все отчёты пользователя — позже можно привязать к кабинету
	reports := h.reports.ListByUser(claims.UserID)
	type info struct {
		ID       string `json:"id"`
		FileName string `json:"fileName"`
	}
	result := make([]info, len(reports))
	for i, r := range reports {
		result[i] = info{ID: r.ID, FileName: r.FileName}
	}
	c.JSON(http.StatusOK, result)
}
