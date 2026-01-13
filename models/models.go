package models

import (
	"time"
)

// WeatherData содержит данные о погоде
type WeatherData struct {
	Provider      string    `json:"provider"`
	Location      string    `json:"location"`
	Temperature   float64   `json:"temperature"`    // в градусах Цельсия
	FeelsLike     float64   `json:"feels_like"`     // ощущается как
	Humidity      int       `json:"humidity"`       // влажность %
	Pressure      int       `json:"pressure"`       // давление в hPa
	WindSpeed     float64   `json:"wind_speed"`     // скорость ветра м/с
	WindDirection int       `json:"wind_direction"` // направление ветра в градусах
	Description   string    `json:"description"`
	Icon          string    `json:"icon"`
	Sunrise       time.Time `json:"sunrise,omitempty"`
	Sunset        time.Time `json:"sunset,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Units         string    `json:"units"` // метрическая или имперская
}

// AggregatedWeather содержит агрегированные данные
type AggregatedWeather struct {
	Location    string          `json:"location"`
	Temperature AggregatedValue `json:"temperature"`
	FeelsLike   AggregatedValue `json:"feels_like"`
	Humidity    AggregatedValue `json:"humidity"`
	Pressure    AggregatedValue `json:"pressure"`
	WindSpeed   AggregatedValue `json:"wind_speed"`
	Description string          `json:"description"`
	Providers   []string        `json:"providers"`
	LastUpdated time.Time       `json:"last_updated"`
}

// AggregatedValue содержит агрегированное значение
type AggregatedValue struct {
	Average float64   `json:"average"`
	Min     float64   `json:"min"`
	Max     float64   `json:"max"`
	Values  []float64 `json:"values,omitempty"`
}

// WeatherRequest запрос на получение погоды
type WeatherRequest struct {
	City    string `json:"city"`
	Country string `json:"country,omitempty"`
	Units   string `json:"units,omitempty"` // metric, imperial
	Lang    string `json:"lang,omitempty"`  // язык ответа
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
