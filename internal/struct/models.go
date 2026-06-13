package models

// Offer — сырое объявление из отчёта Avito
type Offer struct {
	City            string  `json:"city"`
	Category        string  `json:"category"`
	SubCategory     string  `json:"subCategory"`
	Views           int     `json:"views"`
	Shows           int     `json:"shows"`
	Favorite        int     `json:"favorite"`
	Name            string  `json:"name"`
	Contacts        int     `json:"contacts"`
	Promotion       float64 `json:"promotion"`
	ViewersCost     float64 `json:"viewersCost"`
	ViewWithMessage int     `json:"viewWithMessage"`
	LookPhone       int     `json:"lookPhone"`
	TargetViewers   int     `json:"targetViewers"`
}

// Stats — агрегированная статистика по городу
type Stats struct {
	Views           int     `json:"views"`
	Shows           int     `json:"shows"`
	Favorite        int     `json:"favorite"`
	Contacts        int     `json:"contacts"`
	Promotion       float64 `json:"promotion"`
	ViewWithMessage int     `json:"viewWithMessage"`
	LookPhone       int     `json:"lookPhone"`
	TargetViewers   int     `json:"targetViewers"`
	ViewersCost     float64 `json:"viewersCost"`
}

// ResultStats — финальная строка результата (на фронтенд)
type ResultStats struct {
	City            string  `json:"city"`
	Views           int     `json:"views"`
	Shows           int     `json:"shows"`
	Favorite        int     `json:"favorite"`
	Contacts        int     `json:"contacts"`
	Promotion       float64 `json:"promotion"`
	ViewersCost     float64 `json:"viewersCost"`
	TargetViewers   int     `json:"targetViewers"`
	ViewWithMessage int     `json:"viewWithMessage"`
	LookPhone       int     `json:"lookPhone"`
	PPConversion    float64 `json:"ppConversion"`
	PKConversion    float64 `json:"pkConversion"`
	AvgViewPrice    float64 `json:"avgViewPrice"`
	AvgContactPrice float64 `json:"avgContactPrice"`
}

// --- API-структуры ---

// UploadResponse — ответ на загрузку отчёта
type UploadResponse struct {
	ID       string   `json:"id"`
	FileName string   `json:"fileName"`
	Rows     int      `json:"rows"`
	Warnings []string `json:"warnings,omitempty"`
}

// ReportInfo — краткая информация об отчёте
type ReportInfo struct {
	ID       string `json:"id"`
	FileName string `json:"fileName"`
}

// StatsResponse — ответ со статистикой по городам
type StatsResponse struct {
	ReportID string        `json:"reportId"`
	FileName string        `json:"fileName"`
	Stats    []ResultStats `json:"stats"`
}

// MultiStatsResponse — сводная статистика по нескольким отчётам
type MultiStatsResponse struct {
	Reports []StatsResponse `json:"reports"`
}
