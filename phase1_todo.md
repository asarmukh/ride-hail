
# PHASE 1: RIDE REQUEST INITIATION

- [ ] **1) Создать структуру проекта**
  - [-] `cmd/ride-service/main.go`
  - [ ] `cmd/ride-service/Dockerfile`
  - [ ] `internal/shared/config/config.go`
  - [ ] `internal/shared/db/postgres.go`
  - [ ] `internal/shared/mq/rabbitmq.go`
  - [ ] `internal/shared/util/logger.go`
  - [ ] `internal/ride/r/postgres.go`
  - [ ] `internal/ride/app/service.go`
  - [ ] `internal/ride/api/handlers.go`
  - [ ] `migrations/`
  - [ ] `docker-compose.yml`
  - [ ] `Makefile` (опционально)

- [ ] **2) Добавить миграции**
  - [ ] Создать `001_roles_and_users.up.sql` и `.down.sql`
  - [ ] Создать `002_rides.up.sql` и `.down.sql`
  - [ ] Создать `003_drivers.up.sql` и `.down.sql`

- [ ] **3) docker-compose — поднять infra**
  - [ ] Добавить сервисы `postgres`, `rabbitmq`, `migrate`
  - [ ] Запустить: `docker-compose up -d postgres rabbitmq migrate`
  - [ ] Проверить миграции: `docker logs ridehail_migrate`
  - [ ] Проверить таблицы: `docker exec -it ridehail_postgres psql ... -c "\dt"`

- [ ] **4) Реализация shared/config**
  - [ ] Написать структуру `Config`
  - [ ] Функцию `Load()` для env переменных

- [ ] **5) Реализация shared/db и shared/mq**
  - [ ] `db.NewPostgresPool(cfg)`
  - [ ] `mq.NewRabbitMQ(cfg)`

- [ ] **6) Реализовать util/logger**
  - [ ] JSON-логгер
  - [ ] Методы: `Info`, `Error`, `Fatal`

- [ ] **7) Реализовать репозиторий RideRepository**
  - [ ] Интерфейс `CreateRide(...)`
  - [ ] Реализация через pgxpool

- [ ] **8) Реализовать бизнес-логику RideService**
  - [ ] Валидация входа
  - [ ] Расчёт `fare`
  - [ ] Генерация `ride_id`, `ride_number`
  - [ ] Сохранение в БД
  - [ ] Публикация события в RabbitMQ

- [ ] **9) Функция расчёта расстояния и fare**
  - [ ] Реализовать Haversine
  - [ ] Добавить тарифы (ECONOMY, PREMIUM, XL)

- [ ] **10) Реализовать API**
  - [ ] `POST /rides`
  - [ ] Валидация входа
  - [ ] Вызов `RideService.CreateRide`
  - [ ] Ответ JSON

- [ ] **11) Main: запуск сервера**
  - [ ] Загрузить конфиг
  - [ ] Инициализировать logger, db, rmq
  - [ ] Зарегистрировать handlers
  - [ ] Запустить http.Server
  - [ ] Добавить graceful shutdown

- [ ] **12) Dockerfile и docker-compose**
  - [ ] Написать `cmd/ride-service/Dockerfile`
  - [ ] Добавить сервис `ride-service` в `docker-compose.yml`

- [ ] **13) Локальное тестирование**
  - [ ] Поднять infra: `docker-compose up -d postgres rabbitmq migrate`
  - [ ] Запустить сервис: `go run ./cmd/ride-service`
  - [ ] Отправить `POST /rides`
  - [ ] Проверить запись в Postgres
  - [ ] Проверить событие в RabbitMQ

- [ ] **14) Unit/Integration tests**
  - [ ] Unit-тесты для haversine и fare
  - [ ] Мок для `RideRepository`
  - [ ] Тест логики `CreateRide`

- [ ] **15) Makefile (опционально)**
  - [ ] Добавить команды `up`, `down`, `run`, `build`

- [ ] **16) Следующие шаги**
  - [ ] Consumer в `driver-service`
  - [ ] WebSocket пассажир/водитель
  - [ ] Ошибки MQ и retries
  - [ ] Метрики и трейсинг
