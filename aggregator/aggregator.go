package aggregator

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"weather-aggregator/models"
	"weather-aggregator/providers"
)

type Aggregator struct {
	providers []providers.Provider
	cache     map[string]cacheEntry
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
}

type cacheEntry struct {
	data      *models.AggregatedWeather
	timestamp time.Time
}

func NewAggregator(cacheDurationMinutes int) *Aggregator {
	return &Aggregator{
		providers: make([]providers.Provider, 0),
		cache:     make(map[string]cacheEntry),
		cacheTTL:  time.Duration(cacheDurationMinutes) * time.Minute,
	}
}

// AddProvider добавляет провайдера
func (a *Aggregator) AddProvider(provider providers.Provider) {
	if provider.IsAvailable() {
		a.providers = append(a.providers, provider)
	}
}

// GetWeather получает погоду из всех провайдеров и агрегирует
func (a *Aggregator) GetWeather(ctx context.Context, city, country string) (*models.AggregatedWeather, error) {
	cacheKey := fmt.Sprintf("%s,%s", city, country)

	// Пробуем получить из кеша
	if cached, found := a.getFromCache(cacheKey); found {
		return cached, nil
	}

	if len(a.providers) == 0 {
		return nil, fmt.Errorf("нет доступных провайдеров")
	}

	var wg sync.WaitGroup
	results := make(chan *models.WeatherData, len(a.providers))
	errors := make(chan error, len(a.providers))

	// Запускаем запросы ко всем провайдерам параллельно
	for _, provider := range a.providers {
		wg.Add(1)
		go func(p providers.Provider) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				errors <- ctx.Err()
				return
			default:
				weather, err := p.GetWeather(ctx, city, country)
				if err != nil {
					errors <- fmt.Errorf("%s: %w", p.Name(), err)
					return
				}
				results <- weather
			}
		}(provider)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Собираем результаты
	var weatherData []*models.WeatherData
	for weather := range results {
		weatherData = append(weatherData, weather)
	}

	// Собираем ошибки
	var errs []string
	for err := range errors {
		errs = append(errs, err.Error())
	}

	// Если ни один запрос не удался
	if len(weatherData) == 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("все провайдеры вернули ошибки: %v", errs)
		}
		return nil, fmt.Errorf("не удалось получить данные от провайдеров")
	}

	// Агрегируем данные
	aggregated := a.aggregateWeather(weatherData, city, country)

	// Сохраняем в кеш
	a.saveToCache(cacheKey, aggregated)

	return aggregated, nil
}

// aggregateWeather агрегирует данные от разных провайдеров
func (a *Aggregator) aggregateWeather(data []*models.WeatherData, city, country string) *models.AggregatedWeather {
	aggregated := &models.AggregatedWeather{
		Location:    fmt.Sprintf("%s, %s", city, country),
		LastUpdated: time.Now(),
		Providers:   make([]string, 0, len(data)),
	}

	// Собираем значения для агрегации
	var temps, feelsLike, humidity, pressure, windSpeed []float64
	var descriptions []string

	for _, d := range data {
		aggregated.Providers = append(aggregated.Providers, d.Provider)
		temps = append(temps, d.Temperature)
		feelsLike = append(feelsLike, d.FeelsLike)
		humidity = append(humidity, float64(d.Humidity))
		pressure = append(pressure, float64(d.Pressure))
		windSpeed = append(windSpeed, d.WindSpeed)
		descriptions = append(descriptions, d.Description)
	}

	// Агрегируем температуру
	aggregated.Temperature = aggregateValues(temps)
	aggregated.FeelsLike = aggregateValues(feelsLike)
	aggregated.Humidity = aggregateValues(humidity)
	aggregated.Pressure = aggregateValues(pressure)
	aggregated.WindSpeed = aggregateValues(windSpeed)

	// Выбираем наиболее частую погоду
	aggregated.Description = mostFrequent(descriptions)

	return aggregated
}

// aggregateValues вычисляет среднее, мин и макс
func aggregateValues(values []float64) models.AggregatedValue {
	if len(values) == 0 {
		return models.AggregatedValue{}
	}

	sum := 0.0
	min := math.MaxFloat64
	max := -math.MaxFloat64

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return models.AggregatedValue{
		Average: sum / float64(len(values)),
		Min:     min,
		Max:     max,
		Values:  values,
	}
}

// mostFrequent находит наиболее частое значение
func mostFrequent(values []string) string {
	freq := make(map[string]int)
	for _, v := range values {
		freq[v]++
	}

	maxFreq := 0
	var result string
	for v, f := range freq {
		if f > maxFreq {
			maxFreq = f
			result = v
		}
	}

	return result
}

// getFromCache получает данные из кеша
func (a *Aggregator) getFromCache(key string) (*models.AggregatedWeather, bool) {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	entry, found := a.cache[key]
	if !found {
		return nil, false
	}

	// Проверяем TTL
	if time.Since(entry.timestamp) > a.cacheTTL {
		return nil, false
	}

	return entry.data, true
}

// saveToCache сохраняет данные в кеш
func (a *Aggregator) saveToCache(key string, data *models.AggregatedWeather) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	a.cache[key] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// ClearCache очищает кеш
func (a *Aggregator) ClearCache() {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	a.cache = make(map[string]cacheEntry)
}

func (a *Aggregator) GetProviderCount() int {
	return len(a.providers)
}

// GetProvidersInfo возвращает информацию о провайдерах
func (a *Aggregator) GetProvidersInfo() []string {
	info := make([]string, len(a.providers))
	for i, provider := range a.providers {
		info[i] = provider.Name()
	}
	return info
}
