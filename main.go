package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"weather-aggregator/aggregator"
	"weather-aggregator/config"
	"weather-aggregator/models"
	"weather-aggregator/providers"
)

var (
	cfg *config.Config
	agg *aggregator.Aggregator
)

func main() {
	// Загружаем конфигурацию
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Создаем агрегатор
	agg = aggregator.NewAggregator(cfg.CacheDuration)

	// Добавляем провайдеры
	if cfg.OpenWeatherAPIKey != "" {
		agg.AddProvider(providers.NewOpenWeatherProvider(cfg.OpenWeatherAPIKey))
		log.Printf("Провайдер OpenWeatherMap добавлен")
	}

	if cfg.WeatherAPIKey != "" {
		agg.AddProvider(providers.NewWeatherAPIProvider(cfg.WeatherAPIKey))
		log.Printf("Провайдер WeatherAPI добавлен")
	}

	// Создаем CLI команды
	var rootCmd = &cobra.Command{
		Use:   "weather",
		Short: "Погодный агрегатор",
		Long:  "Получает погоду из нескольких источников и агрегирует данные",
	}

	// Команда для запуска сервера
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Запуск HTTP сервера",
		Run: func(cmd *cobra.Command, args []string) {
			startServer()
		},
	}

	// Команда для запроса погоды через CLI
	var getCmd = &cobra.Command{
		Use:   "get [город]",
		Short: "Получить погоду для города",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			city := args[0]
			country, _ := cmd.Flags().GetString("country")
			output, _ := cmd.Flags().GetString("output")

			getWeatherCLI(city, country, output)
		},
	}

	getCmd.Flags().StringP("country", "c", "RU", "Код страны (например, RU, US)")
	getCmd.Flags().StringP("output", "o", "text", "Формат вывода (text, json)")

	// Команда для проверки провайдеров
	var providersCmd = &cobra.Command{
		Use:   "providers",
		Short: "Показать список доступных провайдеров",
		Run: func(cmd *cobra.Command, args []string) {
			showProviders()
		},
	}

	// Команда для очистки кеша
	var clearCacheCmd = &cobra.Command{
		Use:   "clear-cache",
		Short: "Очистить кеш",
		Run: func(cmd *cobra.Command, args []string) {
			clearCache()
		},
	}

	rootCmd.AddCommand(serverCmd, getCmd, providersCmd, clearCacheCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// startServer запускает HTTP сервер
func startServer() {
	mux := http.NewServeMux()

	// Маршруты API
	mux.HandleFunc("/api/weather", weatherHandler)
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/", homeHandler)

	// Статические файлы (опционально)
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Настройка сервера
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Сервер запущен на порту %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	<-quit
	log.Println("Завершение работы сервера...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при завершении работы сервера: %v", err)
	}

	log.Println("Сервер остановлен")
}

// weatherHandler обработчик запроса погоды
func weatherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	city := r.URL.Query().Get("city")
	country := r.URL.Query().Get("country")

	if city == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error: "Не указан город",
		})
		return
	}

	if country == "" {
		country = "RU"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	weather, err := agg.GetWeather(ctx, city, country)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "Не удалось получить погоду",
			Details: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(weather)
}

// healthHandler проверка здоровья сервиса
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"timestamp":      time.Now().Format(time.RFC3339),
		"providers":      agg.GetProviderCount(),
		"provider_names": agg.GetProvidersInfo(),
	})
}

// homeHandler главная страница
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Погодный агрегатор</title>
        <style>
            body { font-family: Arial, sans-serif; margin: 40px; }
            .container { max-width: 800px; margin: 0 auto; }
            .api-link { background: #f0f0f0; padding: 20px; border-radius: 5px; margin: 20px 0; }
            code { background: #eee; padding: 2px 4px; }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>🌤️ Погодный агрегатор</h1>
            <p>Сервис агрегирует данные о погоде из нескольких источников.</p>
            
            <div class="api-link">
                <h3>API Endpoints:</h3>
                <ul>
                    <li><code>GET /api/weather?city=Москва&country=RU</code> - получить погоду</li>
                    <li><code>GET /api/health</code> - проверка здоровья сервиса</li>
                </ul>
            </div>
            
            <p>Пример запроса:</p>
            <pre><code>curl "http://localhost:%s/api/weather?city=Москва&country=RU"</code></pre>
        </div>
    </body>
    </html>
    `, cfg.ServerPort)
}

// getWeatherCLI получает погоду через CLI
func getWeatherCLI(city, country, output string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	weather, err := agg.GetWeather(ctx, city, country)
	if err != nil {
		log.Fatalf("Ошибка: %v", err)
	}

	if output == "json" {
		data, _ := json.MarshalIndent(weather, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Текстовый вывод
	fmt.Printf("🌤️  Погода в %s\n", weather.Location)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Температура: %.1f°C (мин: %.1f°C, макс: %.1f°C)\n",
		weather.Temperature.Average, weather.Temperature.Min, weather.Temperature.Max)
	fmt.Printf("Ощущается как: %.1f°C\n", weather.FeelsLike.Average)
	fmt.Printf("Влажность: %.0f%%\n", weather.Humidity.Average)
	fmt.Printf("Давление: %.0f hPa\n", weather.Pressure.Average)
	fmt.Printf("Скорость ветра: %.1f м/с\n", weather.WindSpeed.Average)
	fmt.Printf("Описание: %s\n", weather.Description)
	fmt.Printf("Источники: %s\n", strings.Join(weather.Providers, ", "))
	fmt.Printf("Обновлено: %s\n", weather.LastUpdated.Format("15:04:05"))
}

// showProviders показывает список доступных провайдеров
func showProviders() {
	fmt.Println("📡 Доступные провайдеры погоды:")
	fmt.Println(strings.Repeat("-", 30))

	// Временное решение - отображаем на основе конфигурации
	if cfg.OpenWeatherAPIKey != "" {
		fmt.Println("✓ OpenWeatherMap")
	} else {
		fmt.Println("✗ OpenWeatherMap (не настроен)")
	}

	if cfg.WeatherAPIKey != "" {
		fmt.Println("✓ WeatherAPI")
	} else {
		fmt.Println("✗ WeatherAPI (не настроен)")
	}
}

// clearCache очищает кеш
func clearCache() {
	agg.ClearCache()
	fmt.Println("✅ Кеш очищен")
}
