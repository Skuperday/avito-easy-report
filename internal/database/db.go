package database

import (
	"avito-easy-report/internal/config"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init открывает подключение к PostgreSQL с ретраями
func Init(cfg *config.Config) error {
	dsn := cfg.DatabaseDSN()

	var err error
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("Подключено к PostgreSQL")
			return nil
		}
		log.Printf("Попытка %d: PostgreSQL не готов (%v), ждём...", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("ошибка подключения к PostgreSQL: %w", err)
}

func Migrate(models ...interface{}) error {
	return DB.AutoMigrate(models...)
}

func SeedAdmin() error {
	var count int64
	DB.Model(&User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	admin := User{
		Username: "admin",
		Password: string(hash),
		Role:     "admin",
	}

	if err := DB.Create(&admin).Error; err != nil {
		return fmt.Errorf("ошибка создания admin: %w", err)
	}

	log.Println("Создана учётка по умолчанию: admin / admin")
	return nil
}

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password string `gorm:"not null" json:"-"`
	Role     string `gorm:"size:20;not null;default:guest" json:"role"`
}
