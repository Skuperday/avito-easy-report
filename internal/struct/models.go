package models

// Объявление
type Offer struct {
	City            string  // Город объявления
	Category        string  // Категория
	SubCategory     string  // Подкатегория
	Views           int     // Просмотры
	Shows           int     // Показы
	Favorite        int     // Избранное
	Name            string  // Название
	Contacts        int     // Контакты
	Promotion       float64 // Цена продвижения
	ViewersCost     float64 // Затраты на просмотры
	ViewWithMessage int     // Написали в чат
	LookPhone       int     // Смотрели телефон
	TargetViewers   int     // Целевые просмотры
}

// Статистика
type Stats struct {
	Views           int     // Просмотры
	Shows           int     // Показы
	Favorite        int     // Избранное
	Contacts        int     // Контакты
	Promotion       float64 // Цена продвижения
	ViewWithMessage int     // Написали в чат
	LookPhone       int     // Смотрели телефон
	TargetViewers   int     // Целевые просмотры
	ViewersCost     float64 // Затраты на просмотры
}

// Результат по статистике
type ResultStats struct {
	City            string  // Город
	Views           int     // Просмотры
	Shows           int     // Показы
	Favorite        int     // Избранное
	Contacts        int     // Контакты
	Promotion       float64 // Затрачено средств
	PPConversion    float64 // Конверсия показы-просмотры
	PKConversion    float64 // Конверсия просмотры-контакты
	ViewersCost     float64 // Затраты на просмотры
	TargetViewers   int     // Целевые просмотры
	ViewWithMessage int     // Написали в чат
	LookPhone       int     // Смотрели телефон
	AvgViewPrice    float64 // Ср. цена просмотра
	AvgContactPrice float64 // Ср. цена контакта
}
