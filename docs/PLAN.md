# Jigsaw — Статус реализации

Все пять фаз реализованы и закоммичены в ветку `feat/phase-1-foundation`.

---

## Что реализовано

### Инфраструктура и база
- PostgreSQL 18 + SeaweedFS + Traefik (Docker Compose)
- Миграции через golang-migrate + embed.FS
- Фоновый воркер: очередь задач на PostgreSQL (SKIP LOCKED), max 3 попытки
- Seed: первый администратор создаётся из `ADMIN_PASSWORD` env

### Аутентификация
- `POST /api/auth/register` — регистрация (email + пароль + locale)
- `POST /api/auth/login` / `POST /api/auth/logout`
- `GET /api/auth/me`
- Сессии в cookie (bcrypt для паролей)
- Middleware: Auth, RequireAuth, RequireAdmin, ChildAuth, RequireChild

### Публичный каталог
- `GET /api/categories` — список категорий
- `GET /api/catalog?category=&difficulty=` — пазлы с фильтрами
- `GET /api/catalog/:id` — детали пазла + слои (puzzle_layers)
- Сложность: easy (≤6 кусков) / medium (≤16) / hard (>16), вычисляется воркером

### Игра и прогресс
- `POST /api/play/:id/complete` — отметить пазл собранным (X-Child-Token)
- `GET /api/play/completed` — список собранных пазлов текущего ребёнка
- `POST /api/children/auth` — PIN-вход ребёнка → child_token в sessionStorage

### Родительский кабинет
- CRUD `/api/parent/children` — профили детей (PIN хэшируется bcrypt, аватар-эмодзи)
- CRUD `/api/parent/puzzles` — пазлы родителя (загрузка изображения, config)
- CRUD `/api/parent/puzzles/:id/layers` — слои наград (word / audio / video)
- `POST /api/parent/puzzles/:id/layers/reorder` — порядок слоёв
- `POST /api/parent/puzzles/:id/submit` — заявка на публикацию
- `GET /api/parent/notifications` — ответы по заявкам

### Администрирование
- CRUD `/api/admin/categories`
- CRUD `/api/admin/catalog/puzzles`
- `GET /api/admin/users`
- `GET /api/admin/moderation` + approve/reject

### Медиа и безопасность
- `GET /api/media/{path...}` — стриминг файлов из S3
- AES-256-GCM шифрование аудио/видео слоёв (ключ: `MEDIA_ENCRYPTION_KEY`)
- Дешифровка на лету при отдаче файлов с префиксами `audio/` и `video/`

### Фоновый воркер
- `process_image`: скачать из S3 → resize → slice (pkg/slicer) → upload pieces → DB → status=ready
- `generate_tts`: stub (заглушка, требует интеграции TTS-провайдера)
- `process_video`: stub (заглушка, требует ffmpeg)

### Фронтенд (Preact + Tailwind + Vite)
- Публичный каталог с фильтрами по категории и сложности, галочки на собранных пазлах
- Игровой экран: Canvas + Pointer Events drag-drop, snap-to-place, лоток
- Экран наград: конфетти → послойная последовательность (word / audio / video)
- Родительский кабинет: трёхшаговый мастер создания пазла, управление слоями, уведомления
- PIN-экран выбора профиля ребёнка (numpad, state machine)
- Страница модерации для администратора
- i18n: ru, en, es, zh, th (JSON-файлы в `web/public/locales/`)
- PWA: manifest.json, service worker (cache-first media, network-first API), SW registration

---

## Схема БД (актуальная)

```
users           — id, email, password_hash, role, locale, created_at
sessions        — id, user_id, token, expires_at
child_sessions  — id, child_id, token, created_at
children        — id, user_id, name, pin_hash, avatar_emoji, created_at
puzzles         — id, user_id, image_key, status, config (jsonb),
                  category_id, difficulty, visibility, owner_type, created_at
puzzle_pieces   — id, puzzle_id, image_key, path_svg, grid_x, grid_y, bounds (jsonb)
puzzle_layers   — id, puzzle_id, sort_order, type, text, audio_key, tts_key, video_key, created_at
categories      — id, slug, name (jsonb), icon, sort_order, created_at
catalog_puzzles — id, puzzle_id, featured, sort_order, created_at
catalog_submissions — id, puzzle_id, status, reviewer_id, admin_comment, reason,
                      notified_at, created_at
play_results    — id, child_id, puzzle_id, completed, duration_ms, attempts, created_at
tasks           — id, type, status, payload (jsonb), error, attempts, created_at, updated_at
```

---

## S3-структура

```
originals/{key}.jpg|png          — оригинальные изображения пазлов
puzzle-pieces/{puzzle_id}/{n}.png — нарезанные куски
audio/{puzzle_id}/{layer_id}.mp3|wav — аудио (зашифровано AES-256-GCM)
video/{puzzle_id}/{layer_id}.mp4     — видео (зашифровано AES-256-GCM)
```

---

## Env-переменные

| Переменная | Назначение |
|------------|-----------|
| `DATABASE_URL` | Строка подключения к PostgreSQL |
| `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` | SeaweedFS / S3 |
| `COOKIE_SECURE` | `true` в продакшене (HTTPS) |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | Seed первого администратора |
| `MEDIA_ENCRYPTION_KEY` | 32 байта в hex (64 символа) для AES-256 |

---

## Что ещё не реализовано

- TTS-провайдер (воркер `generate_tts` — заглушка)
- Транскодирование видео через ffmpeg (воркер `process_video` — заглушка)
- OAuth2 (Google, Apple) — не планировалось в текущей фазе
- Нативное Android-приложение
- Prometheus-метрики, бекапы, CI/CD
