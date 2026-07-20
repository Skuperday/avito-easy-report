package models

// Offer — сырое объявление из отчёта Avito
type Offer struct {
	City            string  `json:"city"`
	Category        string  `json:"category"`
	SubCategory     string  `json:"subCategory"`
	ListingNumber   string  `json:"listingNumber"`
	Object          string  `json:"object,omitempty"`
	Employee        string  `json:"employee,omitempty"`
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
	Response        int     `json:"response"`
}

// Stats — агрегированная статистика по ключу группировки
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
	Response        int     `json:"response"`
}

// ResultStats — строка результата с группировкой
type ResultStats struct {
	Number         int     `json:"number,omitempty"`
	Key            string  `json:"key"`
	City           string  `json:"city,omitempty"`
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
	Expense         float64 `json:"expense"`
	Response        int     `json:"response"`
}

// TopItem — элемент топ-N
type TopItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// StatsSummary — краткая сводка
type StatsSummary struct {
	TotalShows    int       `json:"totalShows"`
	TotalViews    int       `json:"totalViews"`
	TotalContacts int       `json:"totalContacts"`
	TotalExpense  float64   `json:"totalExpense"`
	TopOffers     []TopItem `json:"topOffers"`
	TopTitles     []TopItem `json:"topTitles"`
	TopCities     []TopItem `json:"topCities"`
}

// PeriodStats — статистика периода
type PeriodStats struct {
	Summary StatsSummary  `json:"summary"`
	Stats   []ResultStats `json:"stats"`
}

// CompareResponse — сравнение двух периодов
type CompareResponse struct {
	Early  PeriodStats   `json:"early"`
	Late   PeriodStats   `json:"late"`
	Delta  []ResultStats `json:"delta"`
}

// --- API-структуры ---

type UploadResponse struct {
	ID       string   `json:"id"`
	FileName string   `json:"fileName"`
	Rows     int      `json:"rows"`
	Warnings []string `json:"warnings,omitempty"`
	Columns  []string `json:"columns,omitempty"`
}

type ReportInfo struct {
	ID       string `json:"id"`
	FileName string `json:"fileName"`
}

type StatsResponse struct {
	ReportID string        `json:"reportId"`
	FileName string        `json:"fileName"`
	Stats    []ResultStats `json:"stats"`
	Summary  StatsSummary  `json:"summary"`
}

type MultiStatsResponse struct {
	Reports []StatsResponse `json:"reports"`
}
