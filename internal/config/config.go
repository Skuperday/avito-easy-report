package config

import "os"

type Config struct {
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	CORSOrigin string
}

func Load() *Config {
	return &Config{
		Port:       envDefault("PORT", "8080"),
		DBHost:     envDefault("DB_HOST", "postgres"),
		DBPort:     envDefault("DB_PORT", "5432"),
		DBUser:     envDefault("DB_USER", "avito"),
		DBPassword: envDefault("DB_PASSWORD", "avito"),
		DBName:     envDefault("DB_NAME", "avito"),
		JWTSecret:  envDefault("JWT_SECRET", "avito-secret-change-in-production"),
		CORSOrigin: envDefault("CORS_ORIGIN", "*"),
	}
}

func (c *Config) DatabaseDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=disable TimeZone=Europe/Moscow"
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
