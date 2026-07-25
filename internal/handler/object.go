package handler

import (
	"avito-easy-report/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ObjectHandler struct {
	store *service.ObjectStore
}

func NewObjectHandler(store *service.ObjectStore) *ObjectHandler {
	return &ObjectHandler{store: store}
}

// UploadMapping загружает файл маппинга (XLSX с колонками: номер объявления, объект)
func (h *ObjectHandler) UploadMapping(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл не найден"})
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось открыть XLSX: " + err.Error()})
		return
	}
	defer f.Close()

	// Пробуем все листы
	var rows [][]string
	for _, sheet := range f.GetSheetList() {
		r, _ := f.GetRows(sheet)
		if len(r) > 0 {
			// Пропускаем заголовок, если первый столбец — "Номер объявления"
			start := 0
			if len(r[0]) >= 1 && strings.Contains(r[0][0], "Номер") {
				start = 1
			}
			for i := start; i < len(r); i++ {
				if len(r[i]) >= 2 {
					num := strings.TrimSpace(r[i][0])
					obj := strings.TrimSpace(r[i][1])
					if num != "" && obj != "" {
						rows = append(rows, []string{num, obj})
					}
				}
			}
		}
	}

	count := h.store.LoadFromRows(rows)
	c.JSON(http.StatusOK, gin.H{
		"status":  "загружено",
		"count":   count,
		"total":   h.store.Count(),
	})
}

func (h *ObjectHandler) ListMappings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"mappings": h.store.List(),
		"total":    h.store.Count(),
	})
}

func (h *ObjectHandler) DeleteMapping(c *gin.Context) {
	listingNumber := c.Param("listingNumber")
	if listingNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "укажите номер объявления"})
		return
	}
	h.store.Remove(listingNumber)
	c.JSON(http.StatusOK, gin.H{"status": "удалён"})
}
