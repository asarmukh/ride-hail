# Распределение задач по доменным сервисам - Ride Hailing System

## Подход к разделению

Проект разделен по **вертикальным доменным сервисам**, где каждый участник получает полный ownership одного микросервиса от базы данных до API и интеграций. Это позволяет команде работать параллельно с минимумом зависимостей.

## Участник 1: Ride Service (Сервис поездок)

**Доменная область:** Управление жизненным циклом поездок и взаимодействие с пассажирами

### Ответственность

- Полный микросервис обработки поездок
- REST API для пассажиров
- WebSocket соединения с пассажирами
- Интеграция с другими сервисами через RabbitMQ
- Расчет тарифов и управление платежами

### Задачи

#### Week 1: Основа сервиса

**База данных:**
- [x] Настроить PostgreSQL подключение с пулом соединений
- [x] Реализовать миграции:
  - Таблица `users` (id, email, role, status, password_hash, attrs)
  - Таблица `roles` (PASSENGER, DRIVER, ADMIN)
  - Таблица `user_status` (ACTIVE, INACTIVE, BANNED)
  - Таблица `rides` (полная схема со всеми полями)
  - Таблица `ride_status` (все статусы поездки)
  - Таблица `vehicle_type` (ECONOMY, PREMIUM, XL)
  - Таблица `coordinates` (для pickup и destination)
  - Таблица `ride_events` (event sourcing)
  - Таблица `ride_event_type` (типы событий)
- [x] Создать индексы: `idx_rides_status`, `idx_coordinates_entity`, `idx_coordinates_current`
- [ ] Написать утилиты для работы с транзакциями

**API - Создание поездки:**
- [x] Реализовать `POST /rides`
  - Валидация входных данных (координаты -90 до 90 lat, -180 до 180 lng)
  - Валидация адресов
  - Проверка существования пассажира
  - Генерация ride_number (формат: RIDE_YYYYMMDD_NNN)
- [x] Реализовать расчет тарифов:
  ```
  ECONOMY: 500₸ base + 100₸/km + 50₸/min
  PREMIUM: 800₸ base + 120₸/km + 60₸/min
  XL: 1000₸ base + 150₸/km + 75₸/min
  ```
- [x] Сохранение поездки в БД со статусом 'REQUESTED'
- [x] Создание записей в coordinates для pickup и destination
- [x] Создание события в ride_events (RIDE_REQUESTED)
- [x] Возврат ответа с ride_id, ride_number, estimated_fare

**RabbitMQ Integration:**
- [x] Подключиться к RabbitMQ с reconnection логикой
- [x] Публиковать сообщения в `ride_topic` exchange:
  - Routing key: `ride.request.{ride_type}`
  - Payload: ride_id, pickup/destination locations, ride_type, estimated_fare, timeout_seconds
  - Добавить correlation_id для трассировки
- [ ] Настроить producer с подтверждением доставки

**Logging:**
- [x] Реализовать структурированное JSON логирование
- [x] Обязательные поля: timestamp, level, service (ride-service), action, message, hostname, request_id, ride_id
- [x] Логировать все ключевые события

#### Week 2: Обработка ответов и статусы

**RabbitMQ Consumer:**
- [x] Создать consumer для очереди `driver_responses`
  - Биндинг: `driver_topic` exchange, routing key `driver.response.*`
  - Обработка ответов водителей (accepted: true/false)
- [x] Обработать acceptance:
  - Обновить статус поездки на 'MATCHED'
  - Сохранить driver_id в таблице rides
  - Записать matched_at timestamp
  - Создать событие DRIVER_MATCHED в ride_events
- [ ] Обработать rejection:
  - Если таймаут не истек, ждать следующего водителя
  - Если таймаут истек, отменить поездку

**Таймер подбора водителя:**
- [x] Запустить таймер на 2 минуты при создании поездки
- [x] При истечении времени:
  - Обновить статус на 'CANCELLED'
  - Установить cancellation_reason: "No drivers available"
  - Уведомить пассажира через WebSocket

**Consumer для статусов водителя:**
- [ ] Подписаться на `driver_status` очередь (`driver.status.*`)
- [ ] Обрабатывать изменения статусов поездки:
  - EN_ROUTE (водитель едет на место встречи)
  - ARRIVED (водитель прибыл)
  - IN_PROGRESS (поездка началась)
  - COMPLETED (поездка завершена)
- [ ] Обновлять соответствующие timestamps в БД
- [ ] Создавать события в ride_events

**API - Отмена поездки:**
- [x] Реализовать `POST /rides/{ride_id}/cancel`
  - Проверка: только пассажир может отменить свою поездку
  - Проверка: поездка должна быть в статусе REQUESTED или MATCHED
  - Сохранение причины отмены
  - Обновление статуса на 'CANCELLED'
  - Установка cancelled_at timestamp
  - Создание события RIDE_CANCELLED
  - Публикация сообщения в `ride_topic` (ride.status.cancelled)

**Расчет возврата средств:**
- [x] Если отмена до назначения водителя: 100% возврат
- [x] Если отмена после назначения: 90% возврат (10% штраф)
- [x] Если отмена после старта поездки: возврат не производится

#### Week 3: WebSocket и финализация

**WebSocket Server:**
- [x] Настроить Gorilla WebSocket на порту из конфига
- [x] Реализовать эндпоинт `ws://{host}/ws/passengers/{passenger_id}`
- [x] Реализовать аутентификацию:
  - Получить auth сообщение в течение 5 секунд
  - Валидировать JWT токен
  - Извлечь passenger_id из токена
  - Сохранить соединение в map[passenger_id]*websocket.Conn
- [x] Реализовать ping/pong (30s ping, 60s timeout)
- [x] Обработка disconnect: удалить из map

**WebSocket Events для пассажиров:**
- [ ] `ride_status_update` - отправлять при изменении статуса:
  ```json
  {
    "type": "ride_status_update",
    "ride_id": "...",
    "status": "MATCHED",
    "message": "Driver found!"
  }
  ```
- [ ] `ride_match_notification` - отправлять при подборе водителя:
  ```json
  {
    "type": "ride_status_update",
    "ride_id": "...",
    "status": "MATCHED",
    "driver_info": {
      "driver_id": "...",
      "name": "...",
      "rating": 4.8,
      "vehicle": {...}
    }
  }
  ```
- [ ] `driver_location_update` - получать из location consumer:
  ```json
  {
    "type": "driver_location_update",
    "ride_id": "...",
    "driver_location": {"lat": ..., "lng": ...},
    "estimated_arrival": "...",
    "distance_to_pickup_km": 1.2
  }
  ```

**Location Updates Consumer:**
- [ ] Подписаться на `location_updates_ride` очередь (fanout)
- [ ] Фильтровать обновления только для активных поездок
- [ ] Отправлять обновления локации соответствующему пассажиру через WebSocket
- [ ] Рассчитывать ETA на основе расстояния и скорости водителя

**Тестирование:**
- [ ] Unit тесты для расчета тарифов
- [ ] Unit тесты для валидации входных данных
- [ ] Интеграционные тесты для полного цикла поездки
- [ ] Тесты для обработки таймаутов
- [ ] Тесты для WebSocket соединений

**Оптимизация:**
- [ ] Оптимизировать запросы к БД
- [ ] Добавить connection pooling
- [ ] Реализовать graceful shutdown
- [ ] Добавить health check endpoint

### Deliverables

- ✅ Полностью рабочий Ride Service
- ✅ REST API для создания и отмены поездок
- ✅ WebSocket server для пассажиров
- ✅ Интеграция с RabbitMQ (producer и consumers)
- ✅ Полный event sourcing для поездок
- ✅ Тесты с покрытием >80%

---

## Участник 2: Driver Service (Сервис водителей)

**Доменная область:** Управление водителями, подбор водителей, обработка поездок со стороны водителя

### Ответственность

- Полный микросервис управления водителями
- REST API для водителей
- WebSocket соединения с водителями
- Алгоритм подбора водителей (PostGIS)
- Обработка начала и завершения поездок

### Задачи

#### Week 1: Основа сервиса

**База данных:**
- [ ] Реализовать миграции:
  - Таблица `drivers` (id, license_number, vehicle_type, vehicle_attrs, rating, total_rides, total_earnings, status, is_verified)
  - Таблица `driver_status` (OFFLINE, AVAILABLE, BUSY, EN_ROUTE)
  - Таблица `driver_sessions` (для учета сессий работы)
  - Таблица `location_history` (архив локаций)
- [ ] Создать индекс `idx_drivers_status`
- [ ] Установить PostGIS extension для геопространственных запросов

**API - Управление доступностью:**
- [ ] Реализовать `POST /drivers/{driver_id}/online`
  - Валидация JWT токена (role: DRIVER)
  - Создание новой сессии в driver_sessions
  - Обновление статуса водителя на 'AVAILABLE'
  - Сохранение начальной локации в coordinates
  - Возврат session_id
- [ ] Реализовать `POST /drivers/{driver_id}/offline`
  - Завершение текущей сессии (ended_at)
  - Обновление статуса на 'OFFLINE'
  - Расчет итогов сессии:
    - Длительность работы
    - Количество поездок
    - Общий заработок
  - Возврат session_summary

**API - Обновление локации:**
- [ ] Реализовать `POST /drivers/{driver_id}/location`
  - Проверка: водитель должен быть online
  - Rate limiting: максимум 1 обновление за 3 секунды
  - Обновление coordinates:
    - Установить предыдущую локацию: is_current = false
    - Создать новую запись с is_current = true
  - Архивирование в location_history
  - Валидация координат (lat: -90 до 90, lng: -180 до 180)
  - Валидация accuracy_meters, speed_kmh, heading_degrees

**RabbitMQ Integration:**
- [ ] Подключиться к RabbitMQ
- [ ] Публиковать обновления локации в `location_fanout` exchange:
  ```json
  {
    "driver_id": "...",
    "ride_id": "..." или null,
    "location": {"lat": ..., "lng": ...},
    "speed_kmh": 45.0,
    "heading_degrees": 180.0,
    "timestamp": "..."
  }
  ```

#### Week 2: Алгоритм подбора и обработка предложений

**RabbitMQ Consumer - Запросы на поездки:**
- [ ] Создать consumer для очереди `driver_matching`
  - Биндинг: `ride_topic` exchange, routing key `ride.request.*`
  - Получение ride_id, pickup_location, ride_type, estimated_fare

**Алгоритм подбора водителей:**
- [ ] Реализовать PostGIS запрос для поиска ближайших водителей:
  ```sql
  SELECT d.id, u.email, d.rating, d.vehicle_attrs, 
         c.latitude, c.longitude,
         ST_Distance(
           ST_MakePoint(c.longitude, c.latitude)::geography,
           ST_MakePoint($1, $2)::geography
         ) / 1000 as distance_km
  FROM drivers d
  JOIN users u ON d.id = u.id
  JOIN coordinates c ON c.entity_id = d.id 
    AND c.entity_type = 'driver' 
    AND c.is_current = true
  WHERE d.status = 'AVAILABLE'
    AND d.vehicle_type = $3
    AND ST_DWithin(
      ST_MakePoint(c.longitude, c.latitude)::geography,
      ST_MakePoint($1, $2)::geography,
      5000  -- радиус 5км
    )
  ORDER BY distance_km, d.rating DESC
  LIMIT 10;
  ```
- [ ] Параметры запроса: pickup_lat, pickup_lng, ride_type
- [ ] Сортировка: сначала по расстоянию, потом по рейтингу

**Отправка предложений водителям:**
- [ ] Отправить предложение первому водителю через WebSocket
- [ ] Запустить таймер на 30 секунд
- [ ] Если отказ или таймаут - отправить следующему водителю
- [ ] Максимум 3 попытки подбора
- [ ] Если никто не принял - отправить сообщение в ride service о неудаче

**WebSocket - Обработка ответов:**
- [ ] Получить от водителя `ride_response`:
  ```json
  {
    "type": "ride_response",
    "offer_id": "...",
    "ride_id": "...",
    "accepted": true,
    "current_location": {"latitude": ..., "longitude": ...}
  }
  ```
- [ ] Если accepted = true:
  - Отменить таймер
  - Обновить статус водителя на 'EN_ROUTE'
  - Опубликовать в `driver_topic` (driver.response.{ride_id})
  - Отправить водителю `ride_details` с информацией о пассажире
- [ ] Если accepted = false:
  - Отправить предложение следующему водителю

**RabbitMQ Producer - Ответы водителей:**
- [ ] Публиковать в `driver_topic` exchange:
  - Routing key: `driver.response.{ride_id}`
  - Payload: driver_id, accepted, estimated_arrival_minutes, driver_info

#### Week 3: Выполнение поездок и финализация

**API - Начало поездки:**
- [ ] Реализовать `POST /drivers/{driver_id}/start`
  - Проверка: водитель должен быть назначен на эту поездку
  - Проверка: статус поездки должен быть 'ARRIVED'
  - Обновление статуса водителя на 'BUSY'
  - Обновление статуса поездки на 'IN_PROGRESS'
  - Сохранение started_at timestamp
  - Сохранение начальной локации
  - Публикация в `ride_topic` (ride.status.in_progress)

**API - Завершение поездки:**
- [ ] Реализовать `POST /drivers/{driver_id}/complete`
  - Проверка: поездка должна быть в статусе 'IN_PROGRESS'
  - Получить actual_distance_km, actual_duration_minutes
  - Пересчитать финальную стоимость на основе фактических данных
  - Рассчитать заработок водителя (80% от стоимости)
  - Обновить total_earnings и total_rides водителя
  - Обновить статус водителя на 'AVAILABLE'
  - Обновить статус поездки на 'COMPLETED'
  - Сохранить completed_at timestamp
  - Публикация в `ride_topic` (ride.status.completed)

**WebSocket Server для водителей:**
- [ ] Реализовать эндпоинт `ws://{host}/ws/drivers/{driver_id}`
- [ ] Аутентификация (5 секунд таймаут, JWT токен, role: DRIVER)
- [ ] Ping/pong keep-alive
- [ ] Хранить соединения в map[driver_id]*websocket.Conn

**WebSocket Events для водителей:**
- [ ] `ride_offer` - отправлять при новом предложении:
  ```json
  {
    "type": "ride_offer",
    "offer_id": "...",
    "ride_id": "...",
    "pickup_location": {...},
    "destination_location": {...},
    "estimated_fare": 1500.0,
    "driver_earnings": 1200.0,
    "distance_to_pickup_km": 2.1,
    "expires_at": "..."
  }
  ```
- [ ] `ride_details` - отправлять после принятия:
  ```json
  {
    "type": "ride_details",
    "ride_id": "...",
    "passenger_name": "...",
    "passenger_phone": "+7-XXX-XXX-XX-XX",
    "pickup_location": {...}
  }
  ```
- [ ] Принимать `ride_response` и `location_update`

**Consumer для статусов поездок:**
- [ ] Подписаться на `ride_status` очередь (ride.status.*)
- [ ] Обрабатывать отмену поездки (CANCELLED):
  - Если водитель был назначен - вернуть в статус 'AVAILABLE'
  - Уведомить водителя через WebSocket

**Тестирование:**
- [ ] Тесты для PostGIS запросов
- [ ] Тесты алгоритма подбора
- [ ] Тесты обработки таймаутов
- [ ] Интеграционные тесты начала/завершения поездки
- [ ] Тесты WebSocket соединений

### Deliverables

- ✅ Полностью рабочий Driver Service
- ✅ REST API для управления водителями
- ✅ WebSocket server для водителей
- ✅ Алгоритм подбора с PostGIS
- ✅ Интеграция с RabbitMQ
- ✅ Тесты с покрытием >80%

---

## Участник 3: Location Service (Сервис локаций)

**Доменная область:** Real-time отслеживание и трансляция локаций всем участникам системы

### Ответственность

- Прием обновлений локаций от водителей
- Broadcast локаций всем заинтересованным сервисам
- WebSocket трансляция для real-time обновлений
- Оптимизация производительности (rate limiting, caching)

### Задачи

#### Week 1: Основа сервиса

**База данных:**
- [ ] Использовать существующие таблицы coordinates и location_history
- [ ] Создать дополнительные индексы для оптимизации:
  - `idx_location_history_driver_ride` на (driver_id, ride_id, recorded_at)
  - `idx_location_history_recorded_at` на (recorded_at) для аналитики

**API - Прием локаций:**
- [ ] Реализовать `POST /locations/driver/{driver_id}`
  - Аутентификация (JWT, role: DRIVER)
  - Rate limiting: максимум 1 запрос за 3 секунды на водителя
  - Валидация coordinates
  - Сохранение в БД (если необходимо)
- [ ] Реализовать `POST /locations/passenger/{passenger_id}`
  - Для отслеживания локации пассажира (опционально)

**RabbitMQ Integration:**
- [ ] Настроить producer для `location_fanout` exchange
- [ ] Broadcast всех обновлений локаций:
  ```json
  {
    "entity_type": "driver",
    "entity_id": "...",
    "ride_id": "..." или null,
    "location": {"lat": ..., "lng": ...},
    "speed_kmh": 45.0,
    "heading_degrees": 180.0,
    "accuracy_meters": 5.0,
    "timestamp": "..."
  }
  ```

**Rate Limiting:**
- [ ] Реализовать in-memory rate limiter
- [ ] Использовать token bucket или sliding window
- [ ] Отклонять запросы с HTTP 429 Too Many Requests

#### Week 2: Real-time трансляция

**WebSocket Infrastructure:**
- [ ] Реализовать WebSocket endpoints для трансляции:
  - `ws://{host}/ws/locations/ride/{ride_id}` - для конкретной поездки
  - `ws://{host}/ws/locations/driver/{driver_id}` - для конкретного водителя
- [ ] Аутентификация через JWT
- [ ] Ping/pong keep-alive

**Consumer из RabbitMQ:**
- [ ] Подписаться на свою очередь от `location_fanout` exchange
- [ ] Получать все обновления локаций
- [ ] Фильтровать и транслировать через WebSocket:
  - Пассажирам - локация их водителя
  - Admin dashboard - все активные локации

**Кеширование:**
- [ ] Реализовать in-memory cache последних локаций водителей
- [ ] TTL: 30 секунд
- [ ] Использовать при запросах к API
- [ ] Структура: map[driver_id]LocationData

**API - Получение текущих локаций:**
- [ ] Реализовать `GET /locations/driver/{driver_id}/current`
  - Сначала проверить cache
  - Если нет в cache - запрос к БД
  - Возврат последней локации с is_current = true
- [ ] Реализовать `GET /locations/ride/{ride_id}/driver`
  - Получить driver_id для поездки
  - Вернуть текущую локацию водителя

#### Week 3: Оптимизация и аналитика

**Расчет ETA:**
- [ ] Реализовать функцию расчета ETA:
  - Использовать текущую локацию водителя
  - Рассчитать расстояние до точки назначения (PostGIS)
  - Учесть текущую скорость водителя
  - Добавить поправку на трафик (фиксированный множитель 1.2)
  - Формула: `ETA = (distance_km / average_speed_kmh) * 1.2 * 60` (минуты)

**API - Аналитика:**
- [ ] Реализовать `GET /locations/history/driver/{driver_id}`
  - Параметры: start_date, end_date, ride_id (опционально)
  - Возврат истории перемещений из location_history
  - Пагинация
- [ ] Реализовать `GET /locations/history/ride/{ride_id}`
  - Полный трек поездки
  - Для отладки и разрешения споров

**Batch Updates:**
- [ ] Реализовать batch обработку обновлений:
  - Собирать обновления в буфер (100ms window)
  - Отправлять batch в БД
  - Broadcast через WebSocket
  - Оптимизация для high-load сценариев

**Геозоны (Geofencing):**
- [ ] Определить важные геозоны (аэропорты, ТЦ, вокзалы)
- [ ] Проверять вхождение водителя в геозону
- [ ] Отправлять события при входе/выходе
- [ ] Использовать для surge pricing (будущая фича)

**Мониторинг:**
- [ ] Метрики:
  - Количество обновлений локаций в секунду
  - Количество активных WebSocket соединений
  - Latency обновлений (от получения до broadcast)
  - Hit rate cache
- [ ] Health check endpoint

**Тестирование:**
- [ ] Нагрузочные тесты (1000+ водителей одновременно)
- [ ] Тесты rate limiting
- [ ] Тесты WebSocket трансляции
- [ ] Тесты расчета ETA

### Deliverables

- ✅ Полностью рабочий Location Service
- ✅ REST API для локаций
- ✅ WebSocket real-time трансляция
- ✅ RabbitMQ fanout broadcast
- ✅ Rate limiting и кеширование
- ✅ Расчет ETA
- ✅ Высокая производительность

---

## Участник 4: Infrastructure & Admin Service

**Доменная область:** Инфраструктура, общие библиотеки, мониторинг, администрирование

### Ответственность

- Настройка RabbitMQ (exchanges, queues, bindings)
- Admin Service (мониторинг, аналитика)
- Общие библиотеки (logging, config, JWT)
- Docker Compose для локальной разработки
- CI/CD и деплоймент
- Интеграционные тесты всей системы

### Задачи

#### Week 1: Инфраструктура RabbitMQ

**Setup RabbitMQ:**
- [ ] Создать скрипт инициализации RabbitMQ
- [ ] Создать exchanges:
  ```
  ride_topic (type: topic, durable: true)
  driver_topic (type: topic, durable: true)
  location_fanout (type: fanout, durable: true)
  ```
- [ ] Создать queues:
  ```
  ride_requests (durable: true)
  ride_status (durable: true)
  driver_matching (durable: true)
  driver_responses (durable: true)
  driver_status (durable: true)
  location_updates_ride (durable: true)
  location_updates_admin (durable: true)
  ```
- [ ] Настроить bindings:
  ```
  ride_requests <- ride_topic (ride.request.*)
  ride_status <- ride_topic (ride.status.*)
  driver_matching <- ride_topic (ride.request.*)
  driver_responses <- driver_topic (driver.response.*)
  driver_status <- driver_topic (driver.status.*)
  location_updates_ride <- location_fanout
  location_updates_admin <- location_fanout
  ```
- [ ] Настроить Dead Letter Exchanges для всех очередей
- [ ] Документировать все routing keys и форматы сообщений

**Общие библиотеки:**
- [ ] Создать пакет `pkg/rabbitmq`:
  - Функции для подключения с reconnection
  - Producer утилиты с confirmation
  - Consumer утилиты с auto-ack/manual-ack
  - Retry логика с exponential backoff
- [ ] Создать пакет `pkg/logger`:
  - Структурированное JSON логирование
  - Обязательные поля
  - Levels: INFO, DEBUG, ERROR
  - Контекстные логеры с correlation_id
- [ ] Создать пакет `pkg/config`:
  - Чтение YAML конфигурации
  - Поддержка env переменных
  - Валидация конфига
- [ ] Создать пакет `pkg/auth`:
  - Генерация JWT токенов
  - Валидация JWT
  - Middleware для HTTP и WebSocket

**Docker Compose:**
- [ ] Создать docker-compose.yml:
  - PostgreSQL с PostGIS
  - RabbitMQ с management plugin
  - Redis (для будущего кеширования)
  - Все микросервисы
- [ ] Создать .env.example с переменными окружения
- [ ] Создать Makefile с командами:
  - `make up` - запуск всех сервисов
  - `make down` - остановка
  - `make logs` - просмотр логов
  - `make migrate` - запуск миграций
  - `make test` - запуск тестов

#### Week 2: Admin Service

**База данных:**
- [ ] Использовать существующие таблицы для аналитики
- [ ] Создать materialized views для оптимизации:
  - `mv_daily_stats` - дневная статистика
  - `mv_driver_performance` - производительность водителей
  - Refresh каждые 5 минут

**API - System Overview:**
- [ ] Реализовать `GET /admin/overview`
  - Аутентификация (JWT, role: ADMIN)
  - Подсчет метрик:
    ```sql