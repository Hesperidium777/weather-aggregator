package providers

import (
	"context"
	"weather-aggregator/models"
)

// Provider интерфейс для всех погодных провайдеров
type Provider interface {
	Name() string
	GetWeather(ctx context.Context, city, country string) (*models.WeatherData, error)
	IsAvailable() bool
}
