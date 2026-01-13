# weather-aggregator

Сервис для получения и агрегации данных о погоде из двух API, написанный на Go.

## Функциональность

- Получение погоды из OpenWeatherMap и WeatherAPI
- Агрегация данных от разных провайдеров
- Кеширование результатов
- REST API и CLI интерфейс

## Установка

1. Клонируйте репозиторий:
git clone https://github.com/yourusername/weather-aggregator.git
cd weather-aggregator

2.Настройте .env файл
Получите ключи для беслпатных версий двух выше указанных API и укажите их в .env файле следующим образом:
# API ключи
OPENWEATHER_API_KEY=ваш_ключ_openweather
WEATHERAPI_API_KEY=ваш_ключ_weatherapi
LOG_LEVEL=info

3. Соберите проект
go mod tidy
go build -o weather

4. Для работы сервера добавьте в .env файл его настройки:
# Настройки сервера
SERVER_PORT=8080
CACHE_DURATION=10
LOG_LEVEL=info
