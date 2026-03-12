# CLAUDE.md

Инструкции для Claude Code при работе с этим репозиторием.

## Обзор проекта

Jigsaw — онлайн-игра с разрезными картинками (пазлами) для детей с РАС. Бэкенд — Go (REST API). Фронтенд — Preact + Tailwind, раздаётся как статика через Traefik (отдельно от API). Подробная постановка задачи — `docs/PRODUCT.md`, план реализации — `docs/PLAN.md`.

## Команды

```bash
# Все тесты
go test ./...

# Один тест
go test ./pkg/slicer -run TestDucksPuzzle

# Тесты с выводом (полезно для testdata)
go test ./pkg/slicer -v

# Docker-окружение
docker compose -f .docker/base.yml up -d
```

## Архитектура

### Пакеты

| Пакет | Назначение |
|-------|-----------|
| `pkg/slicer` | Нарезка изображений на пазл-куски (4 режима: Grid, Merge, Geometry, Puzzle) |
| `pkg/s3` | S3-клиент (SeaweedFS, AWS-совместимый) |
| `pkg/pgx` | Пул соединений PostgreSQL (pgxpool) |
| `pkg/logger` | Zap-логгер |
| `pkg/di` | DI-контейнер (samber/do): логгер, БД, S3, HTTP-сервер |
| `internal/migrate` | Миграции БД (golang-migrate + embed.FS) |
| `web/` | Фронтенд (Preact + Tailwind + Vite), отдельный проект |

### Ключевые типы (`pkg/slicer`)
- `Piece` — результат нарезки: `*image.RGBA`, `Path`, `Bounds`, `GridPos`
- `Path` / `PathCmd` — векторный путь (MoveTo/LineTo/CubicTo/Close), экспорт в SVG
- `Point` — float64 2D точка

### Конвейер нарезки
Каждый режим строит `Path` → `clipPiece()` → flatten → bounds → rasterize (ray-casting `pointInPolygon`).

### Экспорт (`export.go`)
`ExportMeta`/`ExportMetaJSON` — JSON для клиента. `Silhouette` — SVG с контурами.

## Инфраструктура

Docker-манифесты в `.docker/`:
- `base.yml` — PostgreSQL 18, SeaweedFS, Traefik
- `.postgres/init.sql` — инициализация БД
- `.traefik/` — конфигурация обратного прокси
- `seaweedfs/s3.json` — конфиг S3

## Тесты

JPEG-фикстуры (`ducks.jpg`, `cosmic.jpg`) в `pkg/slicer/`. Результаты сохраняются в `pkg/slicer/testdata/`. Хелперы `loadJPEG` и `savePieces` в `cosmic_test.go`.

## Языки

- **Общение с пользователем**: русский
- **Документация** (CLAUDE.md, docs/, коммиты, PR): русский
- **Комментарии в коде, имена переменных, API**: английский
