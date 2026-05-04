## Запуск локально

Требования:
- Go 1.25+
- Docker + Docker Compose
- `golang-migrate` CLI: `brew install golang-migrate` или `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- (опционально) `golangci-lint` для `make lint`: `brew install golangci-lint`
- (опционально) `sqlc` для `make sqlc`: `brew install sqlc`

Шаги:

```bash
# 1. Скопировать шаблон env
cp .env.example .env

# 2. Поднять PostgreSQL в Docker
make dc-up

# 3. Накатить миграции
make migrate-up

# 4. Запустить сервер
make run

# В другом терминале:
curl http://localhost:8080/api/v1/health
# {"status":"ok"}
```

Для остановки: `Ctrl+C` (graceful shutdown ≤5с) и `make dc-down`.

Список всех команд: `make help`.

См. также:
- Архитектура и решения: [docs/project-structure/](docs/project-structure/)
- Стандарты кода: [prompts/](prompts/)

## процесс запуска
1. просим агента /research_codebase  выполнить запрос, команда запускает агентов по ресерчу кодовой базы
пример
```/research_codebase add-calls(либо вставить из заранее подготовленного файла промпт для фичи)```

2. После проведения ресерча, запускаем команду /design_feature с такой семантикой (имя фичи, путь до файла с ресерчем )
пример:
```/design_feature add_calls .thoughts/research/2020-09-09-audio-calls.md```
3. После утверждения дизайна создать детальный план
пример: ```напиши план```
4. начинаем с чистого контекстного окна, вызываем команду (/implement_backend + путь к плану + кол-во агентов)
пример:
```/implement_backend docs/audio-calls/plan```