package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenWeatherAPIKey string
	WeatherAPIKey     string
	ServerPort        string
	CacheDuration     int // минуты
	LogLevel          string
}

func Load() (*Config, error) {
	// Загружаем .env файл если существует
	godotenv.Load()

	config := &Config{
		OpenWeatherAPIKey: getEnv("OPENWEATHER_API_KEY", ""),
		WeatherAPIKey:     getEnv("WEATHERAPI_API_KEY", ""),
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		CacheDuration:     getEnvAsInt("CACHE_DURATION", 10),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}

	// Проверяем наличие хотя бы одного API ключа
	if config.OpenWeatherAPIKey == "" && config.WeatherAPIKey == "" {
		return nil, fmt.Errorf("необходим хотя бы один API ключ (OpenWeather или WeatherAPI)")
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
