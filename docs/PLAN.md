# План реализации

## Фаза 0 — Фундамент (есть)

- [x] Библиотека нарезки изображений (`pkg/slicer`) — 4 режима
- [x] Инфраструктурные пакеты: `pkg/s3`, `pkg/pgx`, `pkg/logger`, `pkg/di`
- [x] Docker-манифесты: PostgreSQL 18, SeaweedFS, Traefik
- [x] Инструмент миграций (`internal/migrate/`) + начальная схема БД

## Фаза 1 — Минимальный бэкенд

### 1.1 Точка входа и конфигурация
- [x] `main.go` — API-сервер
- [x] Конфигурация через env-переменные (viper)
- [x] Graceful shutdown
- [ ] Health check endpoint (`GET /healthz`)

### 1.2 Схема БД
- [x] Начальная миграция с таблицами:
  - `users` — id, email, password_hash, role (parent/admin), locale (text, default 'ru'), created_at
  - `sessions` — id, user_id, token, expires_at
  - `children` — id, user_id, name, avatar, created_at
  - `puzzles` — id, user_id, child_id, titles (jsonb), image_key, status, config (jsonb), created_at
  - `puzzle_pieces` — id, puzzle_id, image_key, path_svg, grid_x, grid_y, bounds (jsonb)
  - `rewards` — id, puzzle_id, type, content_key, words (jsonb), tts_keys (jsonb)
  - `catalog_submissions` — id, puzzle_id, status, reviewer_id, reason, created_at _(отложено)_
  - `catalog_puzzles` — id, puzzle_id, featured, sort_order
  - `play_results` — id, child_id, puzzle_id, completed, duration_ms, attempts, created_at
  - `tasks` — id, type, status, payload (jsonb), error, attempts, created_at, updated_at
- [ ] Генерация запросов через sqlc

### 1.3 Аутентификация
- [ ] `POST /api/auth/register` — регистрация (email + пароль + locale)
- [ ] `POST /api/auth/login` — вход, создание сессии
- [ ] `POST /api/auth/logout` — удаление сессии
- [ ] Middleware проверки сессии (cookie)
- [ ] Middleware проверки Bearer token (для мобильного приложения)
- [ ] Middleware определения locale (Accept-Language → настройка пользователя → fallback "ru")
- [ ] Хеширование паролей (bcrypt)

### 1.4 Очередь фоновых задач
- [ ] Воркер на горутинах (SELECT ... FOR UPDATE SKIP LOCKED)
- [ ] Типы задач: `process_image`, `generate_tts`
- [ ] Ретрай с экспоненциальной задержкой (max 3 попытки)
- [ ] Пайплайн обработки изображения:
  1. Валидация
  2. Ресайз/нормализация
  3. Нарезка (pkg/slicer)
  4. Сохранение кусков в S3
  5. Генерация метаданных
  6. Обновление статуса пазла → `ready`

## Фаза 2 — Публичный каталог и админка ← **ТЕКУЩИЙ ПРИОРИТЕТ**

> Каталог формируется администрацией. Пользовательские заявки (`catalog_submissions`) — отложено до появления необходимости.

### 2.1 Админ-бэкенд
- [ ] Middleware проверки роли admin
- [ ] `POST /api/admin/catalog/puzzles` — создать пазл в каталог (загрузить изображение)
- [ ] `GET /api/admin/catalog/puzzles` — список пазлов каталога (статус, превью)
- [ ] `PUT /api/admin/catalog/puzzles/:id` — редактировать метаданные (titles, featured, sort_order)
- [ ] `DELETE /api/admin/catalog/puzzles/:id` — удалить пазл из каталога
- [ ] `POST /api/admin/catalog/puzzles/:id/reward` — назначить награду
- [ ] `GET /api/admin/users` — список пользователей
- [ ] `PUT /api/admin/users/:id` — блокировка/разблокировка

### 2.2 Публичный каталог (API для игроков)
- [ ] `GET /api/catalog` — список пазлов (пагинация, locale-aware titles)
- [ ] `GET /api/catalog/:id` — детали пазла (куски, bounds, reward)

### 2.3 Фронтенд: каркас + админка
- [ ] Preact + Tailwind + Vite, проект в `web/`
- [ ] Роутинг (wouter)
- [ ] Адаптивный layout (mobile-first), базовые компоненты
- [ ] i18n: `web/public/locales/{ru,en}.json`, определение языка по Accept-Language
- [ ] HTTP-клиент для API (`/api/*`)
- [ ] `vite.config.ts` с proxy на Go-сервер для dev-режима
- [ ] Экран входа (admin login)
- [ ] Админ-панель:
  - Список пазлов каталога с превью
  - Загрузка изображения + настройка нарезки
  - Назначение награды (видео / слово)
  - Редактирование titles (ru/en), featured, sort_order
  - Список пользователей

## Фаза 3 — Игровой экран

### 3.1 Профили детей (бэкенд)
- [ ] `POST /api/children` — создать профиль ребёнка
- [ ] `GET /api/children` — список профилей
- [ ] `PUT /api/children/:id` — обновить (имя, аватарка)
- [ ] `DELETE /api/children/:id`

### 3.2 Пазлы пользователя (бэкенд) _(опционально, если решим разрешить загрузку)_
- [ ] `POST /api/puzzles` — загрузить изображение, создать задачу обработки
- [ ] `GET /api/puzzles` — список пазлов текущего пользователя
- [ ] `GET /api/puzzles/:id` — метаданные пазла
- [ ] `DELETE /api/puzzles/:id`

### 3.3 Награды (бэкенд)
- [ ] `POST /api/puzzles/:id/reward` — назначить награду
- [ ] `GET /api/puzzles/:id/reward` — получить награду
- [ ] Поддержка типов: видео (mp4, до 30 сек), слово (текст + опц. TTS)

### 3.4 Игровой экран (фронтенд)
- [ ] Выбор пазла (крупные карточки, без текста, аватарки)
- [ ] Игровой экран:
  - Canvas 2D рендер
  - Drag-and-drop с touch-событиями
  - Snap-to-place при правильном размещении
  - Визуальная обратная связь (подсветка, анимация)
- [ ] Экран награды (воспроизведение видео/слова)

### 3.5 Игровой процесс (бэкенд)
- [ ] `POST /api/play/:puzzle_id/start` — начать игру
- [ ] `POST /api/play/:puzzle_id/complete` — завершить (время, попытки)
- [ ] `GET /api/children/:id/stats` — статистика прогресса

## Фаза 4 — Экраны родителя

- [ ] Регистрация / Вход родителя
- [ ] Выбор профиля ребёнка
- [ ] Коллекция пазлов (каталог + возможно свои)
- [ ] Статистика ребёнка
- [ ] Загрузка собственного изображения + настройка пазла _(если решим включить)_
- [ ] Назначение награды

## Фаза 5 — Полировка и мобилка

### 5.1 Улучшения
- [ ] OAuth2 (Google, Apple) для регистрации
- [ ] TTS-озвучка слов-наград (внешний API или встроенный)
- [ ] WebSocket для live-прогресса обработки изображения
- [ ] PWA-манифест для установки на домашний экран
- [ ] Оффлайн-режим (Service Worker + кешированные пазлы)
- [ ] CDN для статики (в проде Traefik проксирует на S3/CDN)
- [ ] Пользовательские заявки в каталог (`catalog_submissions`) — если будет спрос

### 5.2 Android-приложение
- [ ] REST API уже готов к интеграции (Bearer token auth)
- [ ] Нативный Android-клиент или WebView-обёртка
- [ ] Push-уведомления (пазл обработан, новый контент в каталоге)

### 5.3 Мониторинг и эксплуатация
- [ ] Prometheus-метрики (API latency, очередь задач, ошибки)
- [ ] Structured logging (уже есть через zap)
- [ ] Бекапы PostgreSQL + S3
- [ ] CI/CD: раздельная сборка бэкенда (Go Docker-образ) и фронтенда (статика → S3 или volume)

## Деплой-схема

```
                    Traefik (:80)
                    ┌──────────────────────────┐
                    │                          │
    /api/*  ───────►│  jigsaw-api:8082         │  Go-сервис
                    │                          │
    /*      ───────►│  статика (volume или S3) │  Preact-бандл
                    │                          │
    s3.domain.tld ─►│  seaweedfs:8333          │  Объекты
                    └──────────────────────────┘
```

- Фронт и API на одном домене → нет CORS-проблем, cookie работают без SameSite=None
- Статику можно хранить в Docker volume, в SeaweedFS filer, или на CDN
- Фронт деплоится независимо от бэкенда (копирование файлов, без пересборки Go)

## Порядок работы

```
Фаза 1.1 → 1.3             (health check + аутентификация)
    ↓
Фаза 2.1 → 2.2             (admin API + публичный каталог)
    ↓
Фаза 2.3                   (фронтенд: каркас + админ-панель)
    ↓
Фаза 1.4                   (очередь задач — обработка пазлов)
    ↓
Фаза 3.4 → 3.5             (игровой экран + игровой API)
    ↓
Фаза 3.1 → 3.2 → 3.3      (профили детей, пазлы, награды)
    ↓
Фаза 4                     (экраны родителя)
    ↓
Фаза 5                     (полировка, мобилка)
```

Приоритет первого релиза: администратор может наполнить каталог → ребёнок видит пазлы и собирает их → получает награду.
