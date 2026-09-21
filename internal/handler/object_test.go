package handler

import (
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

type unreadableMappingWorkbook struct{}

func (unreadableMappingWorkbook) GetSheetList() []string {
	return []string{"Повреждённый лист"}
}

func (unreadableMappingWorkbook) GetRows(string, ...excelize.Options) ([][]string, error) {
	return nil, errors.New("worksheet XML is unreadable")
}

func TestReadMappingRowsReturnsWorksheetError(t *testing.T) {
	rows, err := readMappingRows(unreadableMappingWorkbook{})
	if err == nil {
		t.Fatalf("ошибка чтения листа должна прерывать импорт, строки: %#v", rows)
	}
	if len(rows) != 0 {
		t.Fatalf("при ошибке нельзя возвращать частичный маппинг: %#v", rows)
	}
}
