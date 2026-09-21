package handler

import (
	"avito-easy-report/internal/service"
	"log"
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

type mappingWorkbook interface {
	GetSheetList() []string
	GetRows(sheet string, opts ...excelize.Options) ([][]string, error)
}

func readMappingRows(workbook mappingWorkbook) ([][]string, error) {
	var rows [][]string
	for _, sheet := range workbook.GetSheetList() {
		sheetRows, err := workbook.GetRows(sheet)
		if err != nil {
			return nil, err
		}
		if len(sheetRows) == 0 {
			continue
		}
		// Пропускаем заголовок, если первый столбец — "Номер объявления".
		start := 0
		if len(sheetRows[0]) >= 1 && strings.Contains(sheetRows[0][0], "Номер") {
			start = 1
		}
		for i := start; i < len(sheetRows); i++ {
			if len(sheetRows[i]) < 2 {
				continue
			}
			num := strings.TrimSpace(sheetRows[i][0])
			obj := strings.TrimSpace(sheetRows[i][1])
			if num != "" && obj != "" {
				rows = append(rows, []string{num, obj})
			}
		}
	}
	return rows, nil
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

	rows, err := readMappingRows(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать XLSX"})
		return
	}

	count, err := h.store.LoadFromRows(rows)
	if err != nil {
		log.Printf("Ошибка сохранения маппинга объектов: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось сохранить маппинг"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "загружено",
		"count":  count,
		"total":  h.store.Count(),
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
	if err := h.store.Remove(listingNumber); err != nil {
		log.Printf("Ошибка удаления маппинга объекта %q: %v", listingNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось удалить маппинг"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "удалён"})
}
