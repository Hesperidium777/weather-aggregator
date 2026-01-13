package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"weather-aggregator/models"
)

type WeatherAPIProvider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

func NewWeatherAPIProvider(apiKey string) *WeatherAPIProvider {
	return &WeatherAPIProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.weatherapi.com/v1/current.json",
	}
}

func (p *WeatherAPIProvider) Name() string {
	return "WeatherAPI"
}

func (p *WeatherAPIProvider) IsAvailable() bool {
	return p.apiKey != ""
}

func (p *WeatherAPIProvider) GetWeather(ctx context.Context, city, country string) (*models.WeatherData, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("провайдер %s не настроен", p.Name())
	}

	// Формируем запрос
	query := url.Values{}
	query.Set("key", p.apiKey)
	query.Set("q", fmt.Sprintf("%s,%s", city, country))
	query.Set("lang", "ru")

	reqURL := fmt.Sprintf("%s?%s", p.baseURL, query.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiError struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&apiError); err == nil && apiError.Error.Message != "" {
			return nil, fmt.Errorf("ошибка WeatherAPI: %s", apiError.Error.Message)
		}

		return nil, fmt.Errorf("ошибка API: статус %d", resp.StatusCode)
	}

	// Парсим ответ
	var result struct {
		Location struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"location"`
		Current struct {
			TempC      float64 `json:"temp_c"`
			FeelsLikeC float64 `json:"feelslike_c"`
			Humidity   int     `json:"humidity"`
			PressureMB float64 `json:"pressure_mb"`
			WindKph    float64 `json:"wind_kph"`
			WindDeg    int     `json:"wind_degree"`
			Condition  struct {
				Text string `json:"text"`
				Icon string `json:"icon"`
			} `json:"condition"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	// Конвертируем скорость ветра из км/ч в м/с
	windSpeedMS := result.Current.WindKph / 3.6

	weather := &models.WeatherData{
		Provider:      p.Name(),
		Location:      fmt.Sprintf("%s, %s", result.Location.Name, result.Location.Country),
		Temperature:   result.Current.TempC,
		FeelsLike:     result.Current.FeelsLikeC,
		Humidity:      result.Current.Humidity,
		Pressure:      int(result.Current.PressureMB),
		WindSpeed:     windSpeedMS,
		WindDirection: result.Current.WindDeg,
		Description:   result.Current.Condition.Text,
		Icon:          "https:" + result.Current.Condition.Icon,
		Timestamp:     time.Now(),
		Units:         "metric",
	}

	return weather, nil
}
