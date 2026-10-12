# Task Tracker

Небольшое клиент-серверное приложение для управления проектами и задачами.
Бэкенд написан на Go с использованием `net/http` и `pgx`, данные хранятся в
PostgreSQL, а миграции встроены в бинарный файл через `embed.FS`.
По умолчанию интерфейс показывает задачи из всех проектов с названием проекта;
выбор проекта в боковой панели включает фильтр по нему.

## Быстрый запуск

Требуются Docker и Docker Compose.

```bash
cp .env.example .env
make compose-up
```

После запуска интерфейс доступен по адресу <http://localhost:8080>.

Остановить приложение:

```bash
make compose-down
```

## Локальная разработка

Запустите только PostgreSQL:

```bash
docker compose up -d db
```

Запустите сервер. Команда `make run` включает применение миграций при старте:

```bash
make run
```

Для сценария, близкого к production, миграции можно выполнить отдельным
процессом, а сервер запустить с выключенным `MIGRATE_ON_START`.

Основные команды можно посмотреть через `make help`.

## API

```text
GET    /api/projects
POST   /api/projects
GET    /api/projects/{id}
PUT    /api/projects/{id}
DELETE /api/projects/{id}

GET    /api/tasks?project_id={id}&status={status}
POST   /api/tasks
GET    /api/tasks/{id}
PUT    /api/tasks/{id}
DELETE /api/tasks/{id}

GET    /health/live
GET    /health/ready
```

Допустимые статусы задачи: `todo`, `in_progress`, `done`.

## Конфигурация

| Переменная | Назначение | Значение по умолчанию |
|---|---|---|
| `DATABASE_URL` | Строка подключения к PostgreSQL | обязательна |
| `PORT` | HTTP-порт приложения | `8080` |
| `DB_MAX_CONNS` | Максимальный размер пула соединений | `10` |
| `SHUTDOWN_TIMEOUT` | Таймаут корректного завершения | `10s` |
| `MIGRATE_ON_START` | Применять миграции при старте HTTP-сервиса | `false` |

## Миграции

```bash
make migrate-up
make migrate-down
make migrate-version
```

В Docker Compose миграции выполняются отдельным одноразовым контейнером до
запуска приложения, поэтому `MIGRATE_ON_START` по умолчанию выключен. Режим
автоматических миграций оставлен для удобного локального запуска одного
экземпляра приложения.
