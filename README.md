# Task Service

REST API для управления задачами внутри команд: роли (owner/admin/member),
история изменений, комментарии, кеширование списка задач в Redis и
SQL-отчёт по команде.

## Стек

- Go 1.23, стандартный `net/http` (роутинг через `http.ServeMux` с
  method+path паттернами, без стороннего роутера)
- MySQL 8 (`database/sql` + `sqlx`, без ORM)
- Redis (`go-redis/v9`)
- `golang-migrate` (миграции встроены в бинарник через `go:embed`,
  применяются автоматически при старте)
- JWT (`golang-jwt/jwt/v5`), пароли — `bcrypt`
- Docker / docker-compose

## Запуск

```bash
cp .env.example .env      # при необходимости поменяйте JWT_SECRET
docker compose up --build
```

Поднимутся `mysql`, `redis` и `api` (порт 8080). Миграции применяются
автоматически при старте `api` (`internal/repository/db.go: Migrate`) —
отдельный шаг/контейнер не нужен.

Swagger UI: http://localhost:8080/docs
OpenAPI-спека: http://localhost:8080/openapi.yaml

### Локально без Docker

```bash
go run ./cmd/api
```

Нужны локальные MySQL и Redis; переменные окружения — см. `.env.example`
(`config.Load()` читает их напрямую из окружения, без хардкода).

## Примеры запросов

```bash
# регистрация
curl -X POST localhost:8080/api/v1/register \
  -d '{"email":"owner@example.com","password":"password123","name":"Owner"}'

# логин -> JWT
TOKEN=$(curl -s -X POST localhost:8080/api/v1/login \
  -d '{"email":"owner@example.com","password":"password123"}' | jq -r .token)

# создать команду
curl -X POST localhost:8080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" -d '{"name":"Backend"}'

# пригласить участника
curl -X POST localhost:8080/api/v1/teams/1/invite \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"member@example.com","role":"member"}'

# создать задачу
curl -X POST localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"team_id":1,"title":"Fix bug","assignee_id":2}'

# список задач с фильтрами
curl "localhost:8080/api/v1/tasks?team_id=1&status=todo&limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"

# обновить задачу (version обязателен - защита от конкурентной перезаписи)
curl -X PUT localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"version":1,"status":"in_progress"}'

# отчёт по команде (только owner/admin)
curl localhost:8080/api/v1/teams/1/stats -H "Authorization: Bearer $TOKEN"
```

## Права доступа

| Действие | owner | admin | member (создатель задачи) | member (исполнитель) | member (прочий) |
|---|---|---|---|---|---|
| Создать задачу в команде | ✅ | ✅ | ✅ | ✅ | ✅ |
| Редактировать любое поле любой задачи | ✅ | ✅ | ✅ (только свою) | ❌ | ❌ |
| Менять только статус своей назначенной задачи | ✅ | ✅ | — | ✅ | ❌ |
| Переназначать задачу | ✅ | ✅ | ✅ (своя, новый исполнитель — участник команды) | ❌ | ❌ |
| Комментировать | ✅ | ✅ | ✅ | ✅ | ✅ (любой участник команды) |
| Приглашать / менять роль | ✅ | ✅ (кроме owner) | ❌ | ❌ | ❌ |
| Смотреть `/stats` | ✅ | ✅ | ❌ | ❌ | ❌ |

Решение о праве редактирования — чистая функция `taskEditAccess` в
`internal/service/task_access.go`, без обращения к БД, покрыта unit-тестами
(`internal/service/task_access_test.go`).

## Почему так

**Оптимистичная блокировка через `version`, а не пессимистичная (`SELECT
... FOR UPDATE`).** Задачи редактируются нечасто и не одновременно одним
и тем же человеком, а держать транзакцию открытой на время сетевого
round-trip клиента — плохая идея. `PUT /tasks/{id}` требует поле
`version`, полученное из последнего `GET`/`PUT`. `UPDATE ... WHERE id = ?
AND version = ?` атомарно проверяет и применяет изменение за один запрос;
если `version` разошлась — 0 строк обновлено, клиент получает `409` и
обязан перечитать задачу. Один UPDATE, без блокировок, без гонок.

**История собирается только по изменившимся полям, а не полным
снапшотом.** `PUT` сравнивает старое и новое состояние задачи
поле-за-полем (`diffTask`) и пишет в `task_history.changes` JSON только с
теми ключами, что реально изменились, с `{old, new}`. Пустой diff (patch
не менял ничего по факту) не создаёт запись в истории.

**Инвалидация Redis-кеша через счётчик поколений, а не `DEL` по
паттерну.** Список задач кешируется на комбинацию `team_id + status +
assignee_id + limit + offset` — вариантов ключей много. Вместо
`SCAN`/`KEYS` по маске и точечного удаления (дорого, не атомарно) кеш-ключ
включает `generation` — число, которое инкрементируется в Redis одной
атомарной командой `INCR` при любом изменении задач в команде. Старые
ключи со старым `generation` просто больше никогда не запрашиваются и
сами вымирают по TTL. Это `internal/cache/task_list_cache.go`.

**Транзакции строго там, где два write обязаны быть неделимы.** Создание
команды + добавление владельца в `team_members` — одна транзакция
(`TeamService.CreateTeam`): без этого возможна команда без владельца.
Обновление задачи + запись в `task_history` — тоже одна транзакция
(`TaskService.UpdateTask`): история не должна разойтись с фактическим
состоянием задачи.

**Права владельца нельзя обойти через `/invite`.** Эндпоинт приглашения
задваивает роль "добавить нового" и "сменить роль существующему" (upsert),
но явно отказывает, если целевой пользователь уже `owner`, и никогда не
выдаёт роль `owner` — ни владельцу, ни администратору, приглашающему
кого-то.

## Кеширование

`GET /tasks` кеширует результат в Redis на `TASK_CACHE_TTL_SECONDS`
(по умолчанию 300s = 5 минут), ключ учитывает команду и все фильтры.
Любое создание/обновление задачи инвалидирует кеш всей команды (см. выше
про generation-counter).

## Миграции

Лежат в `internal/repository/migrationsfs/*.sql`, встроены в бинарник
через `go:embed` и применяются автоматически при старте (`golang-migrate`,
таблица `schema_migrations` отслеживает уже применённые версии — повторный
запуск безопасен).

## Тестирование

```bash
go test ./...                                  # unit-тесты (RBAC-логика), без Docker
go test -tags=integration ./internal/repository/... -run Stats -v  # интеграционный тест SQL-отчёта, нужен Docker
```

Обязательный интеграционный тест — `internal/repository/stats_repo_integration_test.go`.
Поднимает реальный MySQL через `testcontainers-go`, накатывает миграции,
засеивает фикстуру (в т.ч. задачу, закрытую 60 дней назад — чтобы
проверить, что она НЕ попадает в метрику "топ-3 исполнителя за 30 дней")
и проверяет результат SQL-отчёта. Исключён из обычного `go test ./...`
через build tag `integration`, чтобы остальной набор тестов не зависел от
Docker.

## Известные упрощения (не покрыто из-за таймбокса)

- Явного `unassign` (снять исполнителя, сделать `assignee_id = NULL`)
  через `PUT /tasks/{id}` нет — можно только переназначить на другого
  участника команды.
- Пагинация — offset/limit, не курсорная.
- Rate limiting и circuit breaker не реализованы (в ТЗ отмечены как
  "будет плюсом").
