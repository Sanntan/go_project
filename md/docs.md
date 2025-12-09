## Обзор папки docs

В `docs` хранятся артефакты Swagger/OpenAPI, сгенерированные утилитой `swag` из комментариев в коде (в основном в `cmd/ingestion-service/main.go`).

- `docs/docs.go` — автогенерируемый Go-файл с шаблоном swagger JSON; редактировать не нужно.
- `docs/swagger.yaml` — OpenAPI/Swagger 2.0 в YAML.
- `docs/swagger.json` — тот же контракт в JSON.

Эти файлы обслуживаются эндпоинтом Swagger UI в ingestion-service: `GET /swagger/*any` (см. подключение `ginSwagger.WrapHandler`).

## Как генерируется Swagger в проекте

1) Установить CLI:
```
go install github.com/swaggo/swag/cmd/swag@latest
```

2) Из корня репо выполнить генерацию:
```
swag init -g cmd/ingestion-service/main.go -o docs
```
- `-g` указывает главный файл с аннотациями (`@title`, `@BasePath`, `@Summary`, `@Param`, `@Router` и т.д.).
- `-o docs` кладёт сгенерированные `docs.go`, `swagger.yaml`, `swagger.json` в папку `docs`.

После этого Swagger UI (маршрут `/swagger/*any`) будет отдавать актуальный `swagger.json`.

