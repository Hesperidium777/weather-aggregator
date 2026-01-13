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

type OpenWeatherProvider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

func NewOpenWeatherProvider(apiKey string) *OpenWeatherProvider {
	return &OpenWeatherProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.openweathermap.org/data/2.5/weather",
	}
}

func (p *OpenWeatherProvider) Name() string {
	return "OpenWeatherMap"
}

func (p *OpenWeatherProvider) IsAvailable() bool {
	return p.apiKey != ""
}

func (p *OpenWeatherProvider) GetWeather(ctx context.Context, city, country string) (*models.WeatherData, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("провайдер %s не настроен", p.Name())
	}

	// Формируем запрос
	query := url.Values{}
	query.Set("q", fmt.Sprintf("%s,%s", city, country))
	query.Set("appid", p.apiKey)
	query.Set("units", "metric") // метрическая система
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
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("город не найден")
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("неверный API ключ")
		}
		return nil, fmt.Errorf("ошибка API: статус %d", resp.StatusCode)
	}

	// Парсим ответ
	var result struct {
		Name string `json:"name"`
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Humidity  int     `json:"humidity"`
			Pressure  int     `json:"pressure"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
		Weather []struct {
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
		Sys struct {
			Sunrise int64 `json:"sunrise"`
			Sunset  int64 `json:"sunset"`
		} `json:"sys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if len(result.Weather) == 0 {
		return nil, fmt.Errorf("нет данных о погоде")
	}

	weather := &models.WeatherData{
		Provider:      p.Name(),
		Location:      fmt.Sprintf("%s, %s", result.Name, country),
		Temperature:   result.Main.Temp,
		FeelsLike:     result.Main.FeelsLike,
		Humidity:      result.Main.Humidity,
		Pressure:      result.Main.Pressure,
		WindSpeed:     result.Wind.Speed,
		WindDirection: result.Wind.Deg,
		Description:   result.Weather[0].Description,
		Icon:          result.Weather[0].Icon,
		Sunrise:       time.Unix(result.Sys.Sunrise, 0),
		Sunset:        time.Unix(result.Sys.Sunset, 0),
		Timestamp:     time.Now(),
		Units:         "metric",
	}

	return weather, nil
}
