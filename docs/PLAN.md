# План реализации

## Фаза 0 — Фундамент (есть)

- [x] Библиотека нарезки изображений (`pkg/slicer`) — 4 режима
- [x] Инфраструктурные пакеты: `pkg/s3`, `pkg/pgx`, `pkg/logger`, `pkg/di`
- [x] Docker-манифесты: PostgreSQL 18, SeaweedFS, Traefik
- [x] Инструмент миграций (`internal/migrate/`) + начальная схема БД

## Фаза 1 — Минимальный бэкенд

### 1.1 Точка входа и конфигурация
- [ ] `cmd/jigsaw/main.go` — API-сервер
- [ ] Конфигурация через env-переменные (viper)
- [ ] Graceful shutdown
- [ ] Health check endpoint (`GET /healthz`)

### 1.2 Схема БД
- [x] Начальная миграция с таблицами:
  - `users` — id, email, password_hash, role (parent/admin), locale (text, default 'ru'), created_at
  - `sessions` — id, user_id, token, expires_at
  - `children` — id, user_id, name, avatar, created_at
  - `puzzles` — id, user_id, child_id, titles (jsonb), image_key, status, config (jsonb), created_at
  - `puzzle_pieces` — id, puzzle_id, image_key, path_svg, grid_x, grid_y, bounds (jsonb)
  - `rewards` — id, puzzle_id, type, content_key, words (jsonb), tts_keys (jsonb)
  - `catalog_submissions` — id, puzzle_id, status, reviewer_id, reason, created_at
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

## Фаза 2 — REST API

### 2.1 Профили детей
- [ ] `POST /api/children` — создать профиль ребёнка
- [ ] `GET /api/children` — список профилей
- [ ] `PUT /api/children/:id` — обновить (имя, аватарка)
- [ ] `DELETE /api/children/:id`

### 2.2 Пазлы
- [ ] `POST /api/puzzles` — загрузить изображение, создать задачу обработки
- [ ] `GET /api/puzzles` — список пазлов текущего пользователя
- [ ] `GET /api/puzzles/:id` — метаданные пазла (куски, пути, bounds)
- [ ] `GET /api/puzzles/:id/pieces/:piece_id/image` — изображение куска (presigned URL)
- [ ] `DELETE /api/puzzles/:id`

### 2.3 Награды
- [ ] `POST /api/puzzles/:id/reward` — загрузить/назначить награду
- [ ] `GET /api/puzzles/:id/reward` — получить награду
- [ ] Поддержка типов: видео (mp4, до 30 сек), слово (текст + опц. TTS)

### 2.4 Общий каталог
- [ ] `POST /api/puzzles/:id/submit` — предложить пазл в каталог
- [ ] `GET /api/catalog` — список одобренных пазлов (пагинация, фильтры)
- [ ] `GET /api/catalog/:id` — детали каталожного пазла

### 2.5 Админ-эндпоинты
- [ ] `GET /api/admin/submissions` — очередь на модерацию
- [ ] `PUT /api/admin/submissions/:id` — одобрить/отклонить (с причиной)
- [ ] `POST /api/admin/catalog` — добавить пазл напрямую в каталог
- [ ] `GET /api/admin/users` — список пользователей
- [ ] `PUT /api/admin/users/:id` — блокировка/разблокировка

### 2.6 Игровой процесс
- [ ] `POST /api/play/:puzzle_id/start` — начать игру (для ребёнка)
- [ ] `POST /api/play/:puzzle_id/complete` — завершить (время, попытки)
- [ ] `GET /api/children/:id/stats` — статистика прогресса

## Фаза 3 — Фронтенд

Фронтенд — отдельный проект в `web/`. Собирается Vite, раздаётся как статика через Traefik. API проксируется Traefik на Go-сервис. При разработке — `vite dev` с proxy на `localhost:8082`.

### 3.1 Каркас
- [ ] Preact + Tailwind + Vite, проект в `web/`
- [ ] Роутинг (preact-router или wouter)
- [ ] Адаптивный layout (mobile-first)
- [ ] i18n: файлы `web/public/locales/{ru,en}.json`, определение языка по Accept-Language, переключатель в UI
- [ ] HTTP-клиент для API (`/api/*`)
- [ ] `vite.config.ts` с proxy на Go-сервер для dev-режима

### 3.2 Экраны родителя
- [ ] Регистрация / Вход
- [ ] Выбор профиля ребёнка
- [ ] Коллекция пазлов (свои + каталог)
- [ ] Загрузка изображения + настройка пазла
- [ ] Назначение награды
- [ ] Статистика ребёнка

### 3.3 Экран ребёнка
- [ ] Выбор пазла (крупные карточки, без текста, аватарки)
- [ ] Игровой экран:
  - Canvas 2D рендер
  - Drag-and-drop с touch-событиями
  - Snap-to-place при правильном размещении
  - Визуальная обратная связь (подсветка, анимация)
- [ ] Экран награды (воспроизведение видео/слова)

### 3.4 Админ-панель
- [ ] Очередь модерации с превью
- [ ] Одобрение/отклонение с комментарием
- [ ] Управление каталогом (featured, порядок)
- [ ] Список пользователей

## Фаза 4 — Полировка и мобилка

### 4.1 Улучшения
- [ ] OAuth2 (Google, Apple) для регистрации
- [ ] TTS-озвучка слов-наград (внешний API или встроенный)
- [ ] WebSocket для live-прогресса обработки изображения
- [ ] PWA-манифест для установки на домашний экран
- [ ] Оффлайн-режим (Service Worker + кешированные пазлы)
- [ ] CDN для статики (в проде Traefik проксирует на S3/CDN)

### 4.2 Android-приложение
- [ ] REST API уже готов к интеграции (Bearer token auth)
- [ ] Нативный Android-клиент или WebView-обёртка
- [ ] Push-уведомления (пазл обработан, новый контент в каталоге)

### 4.3 Мониторинг и эксплуатация
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
Фаза 1.1 → 1.2 → 1.3 → 1.4    (бэкенд-фундамент)
           ↓
Фаза 2.1 → 2.2 → 2.3           (core API)
           ↓
Фаза 3.1 → 3.3                  (игровой экран — главная ценность)
           ↓
Фаза 2.4 → 2.5 → 2.6           (каталог, админка, статистика)
           ↓
Фаза 3.2 → 3.4                  (остальные экраны)
           ↓
Фаза 4                          (улучшения, мобилка)
```

Приоритет — как можно скорее получить работающую игру (ребёнок собирает пазл и получает награду). Всё остальное наращивается итеративно.
