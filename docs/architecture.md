# Архитектура Jigsaw

## Общая схема

```
                         Traefik (:80 / :443)
                         ┌──────────────────────────────┐
   /api/*  ─────────────►│  jigsaw-api  :8080           │  Go REST API
   /*      ─────────────►│  статика (volume)            │  Preact-бандл
   s3.*    ─────────────►│  seaweedfs   :8333           │  S3-объекты
                         └──────────────────────────────┘
                                      │
                              PostgreSQL 18
```

Фронт и API на одном домене → cookie без `SameSite=None`, нет CORS.

---

## Стек

| Слой | Технология |
|------|-----------|
| API | Go, net/http (Go 1.22 pattern routing) |
| БД | PostgreSQL 18 + pgx/v5 |
| S3 | SeaweedFS (AWS-совместимый) |
| DI | samber/do/v2 |
| Конфигурация | spf13/viper (env) |
| Фронтенд | Preact + TypeScript + Tailwind CSS + Vite |
| Роутинг | wouter |
| Миграции | golang-migrate + embed.FS |

---

## Go-пакеты

| Пакет | Назначение |
|-------|-----------|
| `pkg/slicer` | Нарезка изображений (4 режима: Grid, Merge, Geometry, Puzzle) |
| `pkg/s3` | S3-клиент (SeaweedFS) |
| `pkg/pgx` | Пул соединений (pgxpool) |
| `pkg/logger` | Zap-логгер |
| `pkg/di` | DI-контейнер (samber/do) |
| `internal/migrate` | Миграции БД |
| `internal/store` | Все запросы к БД |
| `internal/handler` | HTTP-хендлеры |
| `internal/middleware` | Auth, ChildAuth, RequireAdmin, Locale |
| `internal/worker` | Фоновый воркер (process_image, generate_tts stub, process_video stub) |
| `internal/crypto` | AES-256-GCM шифрование/дешифрование |

---

## Аутентификация

Три типа сессий:

**Родитель/Администратор** — cookie-сессия (bcrypt пароль, таблица `sessions`).
Middleware `Auth` читает cookie `session_token`, наполняет контекст `User`.

**Ребёнок** — child_token в `sessionStorage` браузера, передаётся в заголовке
`X-Child-Token`. Таблица `child_sessions`. Middleware `ChildAuth`. Вход через
PIN (4 цифры), хэш хранится в `children.pin_hash` (bcrypt).

**Публичный доступ** — каталог и медиафайлы доступны без авторизации.

---

## Фоновый воркер

Очередь задач на PostgreSQL (`SELECT ... FOR UPDATE SKIP LOCKED`).
Воркер запускается горутиной рядом с HTTP-сервером.

```
poll() → ClaimTask (SKIP LOCKED) → dispatch() → CompleteTask / RetryOrFailTask
```

Типы задач:
- `process_image`: download S3 → decode → resize (≤2048px) → slicer → upload pieces → DB → difficulty → status=ready
- `generate_tts`: заглушка (требует TTS-провайдера)
- `process_video`: заглушка (требует ffmpeg)

Параметры: `pollInterval=2s`, `maxAttempts=3`.

---

## Медиа и шифрование

`GET /api/media/{path...}` стримит объект из S3 напрямую клиенту.

Файлы с префиксом `audio/` и `video/` (слои наград родителя) шифруются
AES-256-GCM перед загрузкой в S3 и дешифруются на лету при отдаче.
Ключ задаётся через `MEDIA_ENCRYPTION_KEY` (32 байта в hex).
Если переменная не задана — шифрование отключено.

---

## Пакет slicer

Каждый режим строит `Path` → `clipPiece()` → flatten → bounds → rasterize.
Результат: `[]Piece` где каждый кусок имеет `*image.RGBA`, `Path` (SVG),
`Bounds`, `GridPos`.

Режимы:
- `Grid` — прямоугольная сетка
- `Puzzle` — классические пазловые выступы (кривые Безье)
- `Geometry` — геометрические формы (треугольники, ромбы и др.)
- `Merge` — слияние соседних ячеек

---

## Игровой экран (фронтенд)

Canvas 2D + Pointer Events API:
1. Лоток снизу (кусочки в случайном порядке, горизонтальный скролл)
2. Целевая зона сверху (SVG-контуры на месте кусков)
3. Drag-drop: `setPointerCapture` → `pointermove` → `pointerup`
4. Snap-to-place: расстояние < порог → анимация примагничивания
5. Hit-testing по alpha-каналу PNG или `ctx.isPointInPath(Path2D, x, y)`

Экран наград — конфетти (Canvas) + послойная последовательность:
`word` (текст) → `audio` (воспроизведение) → `video` (fullscreen).

---

## Фронтенд: структура

```
web/
  src/
    api.ts              — HTTP-клиент для всех эндпоинтов
    types.ts            — TypeScript типы
    i18n.ts             — хелпер перевода
    app.tsx             — роутинг (wouter)
    pages/
      catalog/          — публичный каталог
      game/             — игровой экран, экран наград
      parent/           — кабинет родителя (пазлы, слои, дети, уведомления)
      admin/            — панель администратора (каталог, модерация)
      ChildSelect.tsx   — PIN-экран выбора ребёнка
    components/         — Spinner, TopActions
  public/
    locales/{ru,en,es,zh,th}.json   — i18n строки
    manifest.json       — PWA манифест
    sw.js               — Service Worker
```

---

## UX-особенности для детей с РАС

- Крупные карточки и минимум текста в интерфейсе ребёнка
- Touch-first: `touch-action: none` на canvas, `setPointerCapture`
- Мягкая анимация snap (ease-out, 200мс), конфетти при завершении
- Последовательные награды без неожиданных автозапусков
- Галочки на уже собранных пазлах в каталоге
- Без мигающих эффектов и навязчивых звуков (аудио только по нажатию)
