package models

type Offer struct {
	City        string
	Category    string
	SubCategory string
	Views       int
	Favorite    int
	Name        string
	Contacts    int
	Promotion   float64
}

type Stats struct {
	Views     int
	Favorite  int
	Contacts  int
	Promotion float64
}

// Конверсия просмотры-контакты
// Ср. цена просмотра
// Ср цена контакта
type ResultStats struct {
	City            string
	Views           int
	Favorite        int
	Contacts        int
	Promotion       float64
	PKConversion    float64
	AvgViewPrice    float64
	AvgContactPrice float64
}
