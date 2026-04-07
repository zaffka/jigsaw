# Jigsaw — Дизайн и план реализации

## Контекст

Jigsaw — онлайн-игра с пазлами для детей с РАС. Текущий прototip имеет готовую библиотеку нарезки (`pkg/slicer`), инфраструктуру (PostgreSQL + SeaweedFS + Docker) и каркас фронтенда (Preact + Tailwind). Концепция существенно расширяется: добавляются приватный каталог родителей, послойные награды, профили детей с PIN, модерация, категории.

**Ключевые решения:**
- Хостинг: self-hosted, текущий стек (без Supabase)
- Приватность: изоляция через права доступа (JWT/сессия), шифрование файлов — в перспективе
- Сложность: автоматически по числу кусков — Лёгкий (4–6), Средний (9–16), Сложный (25+)
- Прогресс: минимальный — только факт завершения (галочка на карточке)
- Аудио: TTS как дефолт + загрузка живого голоса
- Категории: управляются администратором, стартовый набор задан
- Фронтенд: два отдельных Preact-приложения (`web/game/` и `web/dashboard/`)
- Родительский UX: desktop-приоритет для создания пазлов
- Профили детей: PIN-вход (4 цифры или картинка-пароль)

---

## Архитектура

### Два фронтенд-приложения

**`web/game/`** — touch-first, для ребёнка:
- `/` → PIN-экран выбора профиля ребёнка
- `/catalog` → каталог пазлов (доступных этому ребёнку)
- `/play/:id` → игровой экран (Canvas + drag-drop, лоток снизу)
- `/reward/:id` → последовательность слоёв наград

**`web/dashboard/`** — desktop-приоритет, для родителя и администратора:
- `/login` → вход
- `/parent/*` → кабинет родителя
- `/admin/*` → панель администратора

### Traefik-роутинг
- `/play/*`, `/catalog/*`, `/` → `web/game/` (статика)
- `/parent/*`, `/admin/*`, `/login` → `web/dashboard/` (статика)
- `/api/*` → Go API :8080

### Go API + Worker
- REST API (net/http), текущая структура пакетов сохраняется
- Worker: задачи `process_image`, `generate_tts`, `process_video`

---

## Изменения в схеме БД

### Новые таблицы

```sql
-- Категории (управляются администратором)
categories (
  id UUID PK,
  slug TEXT UNIQUE,
  name JSONB,        -- {ru: "Животные", en: "Animals", ...}
  icon TEXT,         -- эмодзи или S3-ключ SVG
  sort_order INT,
  created_at TIMESTAMPTZ
)

-- Слои наград (заменяет таблицу rewards)
puzzle_layers (
  id UUID PK,
  puzzle_id UUID FK puzzles,
  sort_order INT,
  type TEXT,         -- 'word' | 'audio' | 'video'
  text TEXT,         -- для type='word' и как источник TTS
  audio_key TEXT,    -- S3-ключ загруженного аудио или TTS
  tts_key TEXT,      -- S3-ключ сгенерированного TTS (заполняет воркер)
  video_key TEXT,    -- S3-ключ видео
  created_at TIMESTAMPTZ
)

-- Сессии детей (PIN-вход)
child_sessions (
  id UUID PK,
  child_id UUID FK children,
  token TEXT UNIQUE,
  created_at TIMESTAMPTZ
)
```

### Изменения в существующих таблицах

```sql
-- puzzles: добавить
ALTER TABLE puzzles ADD COLUMN category_id UUID REFERENCES categories(id);
ALTER TABLE puzzles ADD COLUMN difficulty TEXT;
-- 'easy' | 'medium' | 'hard', вычисляется и выставляется воркером process_image
-- на основе cols*rows: ≤6 → easy, ≤16 → medium, >16 → hard
ALTER TABLE puzzles ADD COLUMN visibility TEXT DEFAULT 'private'; -- 'private' | 'public'
ALTER TABLE puzzles ADD COLUMN owner_type TEXT DEFAULT 'parent';  -- 'parent' | 'admin'

-- children: добавить
ALTER TABLE children ADD COLUMN pin_hash TEXT;    -- bcrypt(PIN)
ALTER TABLE children ADD COLUMN avatar_emoji TEXT DEFAULT '🧒';

-- catalog_submissions: добавить
ALTER TABLE catalog_submissions ADD COLUMN admin_comment TEXT;
ALTER TABLE catalog_submissions ADD COLUMN notified_at TIMESTAMPTZ;

-- play_results: completed BOOLEAN уже есть; duration_ms и attempts оставляем в схеме,
-- но пока не заполняем (добавим позже)
```

### Удалить
- Таблицу `rewards` (заменяется на `puzzle_layers`)

---

## Новые API-эндпоинты

### Публичные (без авторизации)
```
GET  /api/categories                    — список категорий
GET  /api/catalog?category=&difficulty= — публичные пазлы (с фильтрами)
GET  /api/catalog/:id                   — детали пазла + слои
```

### Для ребёнка (child_token из PIN-входа)
```
POST /api/children/auth                 — PIN-вход {child_id, pin} → child_token
POST /api/play/:id/complete             — отметить как собранный
```
`child_token` — opaque UUID, хранится в `child_sessions`. Сессия активна пока открыта вкладка, токен хранится в sessionStorage.

### Для родителя (auth middleware)
```
GET  /api/parent/children               — профили детей
POST /api/parent/children               — создать профиль
GET  /api/parent/children/:id
PUT  /api/parent/children/:id           — обновить (имя, PIN, аватар)
DEL  /api/parent/children/:id

GET  /api/parent/puzzles                — мой каталог пазлов
POST /api/parent/puzzles                — создать пазл (FormData: image + config)
GET  /api/parent/puzzles/:id
PUT  /api/parent/puzzles/:id
DEL  /api/parent/puzzles/:id

GET  /api/parent/puzzles/:id/layers
POST /api/parent/puzzles/:id/layers     — добавить слой
PUT  /api/parent/puzzles/:id/layers/:lid
DEL  /api/parent/puzzles/:id/layers/:lid
POST /api/parent/puzzles/:id/layers/reorder — [{id, sort_order}]

POST /api/parent/puzzles/:id/submit     — подать заявку на модерацию
GET  /api/parent/notifications          — уведомления (ответы по заявкам)
```

### Для администратора (admin middleware)
```
GET  /api/admin/categories
POST /api/admin/categories
PUT  /api/admin/categories/:id
DEL  /api/admin/categories/:id

GET  /api/admin/moderation              — очередь заявок
POST /api/admin/moderation/:id/approve
POST /api/admin/moderation/:id/reject   — body: {comment}

GET  /api/admin/catalog/puzzles         — все пазлы (существующее)
POST /api/admin/catalog/puzzles
...
```

---

## Игровой экран (`web/game/`)

### Компоновка (утверждено: вариант A)
- Вверху: кнопка «назад» (слева) + кнопка «домой» (справа) — круглые иконки
- Центр: Canvas с целевой зоной (SVG-контуры на месте кусков)
- Снизу: лоток с кусочками, горизонтальный скролл

### Экран наград (последовательность слоёв)
1. **Конфетти-анимация** (всегда) — ребёнок нажимает на собранный пазл
2. **Слои по порядку** — каждый слой занимает весь экран:
   - `word` → большое слово, анимация появления букв, нажать → следующий
   - `audio` → большая иконка 🔊, нажать → играет аудио → следующий
   - `video` → кнопка ▶️, нажать → видео на весь экран → после окончания → следующий
3. **Кнопка «Ещё пазл»** → возврат к каталогу
4. Собранная картинка пазла остаётся видна на фоне всех слоёв

### PIN-экран
- Большие карточки с именем и аватаром ребёнка
- При выборе — NUM-пад для ввода 4-значного PIN
- `POST /api/children/auth` → `child_token` в sessionStorage

---

## Интерфейс родителя (`web/dashboard/parent/`)

### Создание пазла — трёхшаговый мастер
**Шаг 1 — Картинка:**
- Drag-and-drop зона загрузки (JPG/PNG, до 10 МБ)
- Выбор колонок/строк (пресеты: 2×2, 3×3, 4×4, 3×4, 4×6, 6×8)
- Выбор режима нарезки (Сетка / Пазл / Геометрия / Слияние)
- Живой предпросмотр сетки + автоматический уровень сложности
- Кнопка «Далее»

**Шаг 2 — Слои:**
- Список добавленных слоёв с drag-and-drop сортировкой
- Кнопки «+ Добавить слово / аудио / видео»
- Для `word`: поле ввода текста + выбор языка
- Для `audio`: загрузить файл (MP3/WAV) или «Сгенерировать из слова» (TTS)
- Для `video`: загрузить файл (MP4, до 100 МБ)

**Шаг 3 — Детали:**
- Название пазла
- Категория (выпадающий список)
- Язык
- Кнопка «Сохранить в мой каталог»
- Опция «Подать на публикацию» (откроет отдельный флоу)

---

## Стартовые категории (seed-данные)

```
Базовые: Животные 🐾, Еда 🍎, Транспорт 🚗, Люди 👨‍👩‍👧, Природа 🌿,
         Игрушки 🧸, Одежда 👕, Дом 🏠
Учебные: Буквы 🔤, Цифры 🔢, Цвета 🎨, Формы 🔷, Эмоции 😊, Действия 🏃
```

---

## S3-структура (новые префиксы)

```
audio/{puzzle_id}/{layer_id}.{mp3|wav}  — загруженное аудио
audio/{puzzle_id}/{layer_id}_tts.mp3    — сгенерированный TTS
video/{puzzle_id}/{layer_id}.mp4        — видео-слои
avatars/{child_id}.{jpg|png}            — аватарки детей
```

---

## Фазы реализации

### Фаза 1 — Фундамент (публичный каталог работает)
**Бэкенд:**
- [ ] Завершить `POST /api/auth/login` и `GET /api/auth/me`
- [ ] Первый admin создаётся через seed-миграцию (INSERT с email + bcrypt-хэш из env `ADMIN_PASSWORD`)
- [ ] Воркер `process_image`: нарезка → S3 → puzzle_pieces → difficulty → status='ready'
- [ ] Миграция: `categories` (со стартовым seed), поля `category_id`/`difficulty`/`visibility`/`owner_type` в `puzzles`
- [ ] `GET /api/categories`
- [ ] `CRUD /api/admin/categories`
- [ ] Обновить `GET /api/catalog` — фильтры по категории и сложности

**Фронтенд (остаётся в `web/`):**
- [ ] Завершить `GameScreen.tsx` — полный drag-drop, snap-алгоритм
- [ ] `POST /api/play/:id/complete` при завершении
- [ ] Простой `RewardScreen.tsx` (конфетти + слои)
- [ ] Фильтры в публичном каталоге (категория, сложность)

### Фаза 2 — Приватный каталог родителя
**Бэкенд:**
- [ ] Миграция: `puzzle_layers`, `child_sessions`, `pin_hash` в `children`, `admin_comment` в `catalog_submissions`
- [ ] CRUD `/api/parent/children` (с хэшированием PIN)
- [ ] CRUD `/api/parent/puzzles` и `/api/parent/puzzles/:id/layers`
- [ ] `POST /api/children/auth` — PIN-вход, child_token
- [ ] Воркер `generate_tts` — генерация аудио из text-слоя
- [ ] Middleware: child_token для `POST /api/play/:id/complete`

**Фронтенд:**
- [ ] Разделить `web/` → `web/game/` и `web/dashboard/` (два Vite-проекта)
- [ ] `web/game/`: PIN-экран выбора профиля ребёнка
- [ ] `web/dashboard/parent/`: трёхшаговый мастер создания пазла
- [ ] `web/dashboard/parent/`: список пазлов родителя с управлением слоями

### Фаза 3 — Игровая петля
- [ ] Полный экран наград (`/reward/:id`) с последовательностью слоёв
- [ ] Аудио-слой: воспроизведение через Web Audio API
- [ ] Видео-слой: `<video>` fullscreen
- [ ] Галочка «собрал» на карточке каталога (из play_results)
- [ ] Обновить Traefik-конфиг для двух статических сервисов

### Фаза 4 — Модерация
- [ ] `POST /api/parent/puzzles/:id/submit`
- [ ] `GET /api/parent/notifications`
- [ ] `GET /api/admin/moderation` + approve/reject endpoints
- [ ] `web/dashboard/admin/`: страница модерации (превью + слои + кнопки)
- [ ] Внутренние уведомления в UI родителя

### Фаза 5 — Полировка и PWA
- [ ] Воркер `process_video` (нормализация)
- [ ] `web/game/` как PWA (manifest.json, service worker, offline-каталог)
- [ ] Шифрование файлов в S3 (AES-256) для приватного контента

---

## Критические файлы

| Файл | Что меняется |
|------|-------------|
| `internal/migrate/sql/migrations/` | Новая миграция: categories, puzzle_layers, child_sessions, изменения в puzzles/children |
| `internal/store/store.go` | Новые методы для categories, puzzle_layers, children (PIN), child_sessions |
| `internal/handler/` | Новые файлы: parent.go, moderation.go, children_auth.go |
| `internal/worker/generate_tts.go` | Новый воркер |
| `main.go` | Новые маршруты /api/parent/*, /api/children/auth |
| `web/` | Разделение на web/game/ и web/dashboard/ |
| `.docker/base.yml` | Traefik-роутинг для двух фронтендов |
| `web/src/pages/game/GameScreen.tsx` | Завершение drag-drop + snap |
| `web/src/pages/game/RewardScreen.tsx` | Последовательность слоёв |

---

## Верификация

**Фаза 1:**
```bash
go test ./...
# docker compose up, создать пазл через админку, воркер обрабатывает
# /catalog → пазл появился → сыграть до конца
```

**Фаза 2:**
```bash
# Зарегистрировать родителя, создать профиль ребёнка с PIN
# Войти через PIN, открыть каталог, сыграть
# Создать пазл с тремя слоями, TTS-воркер сработал
```

**Фаза 4:**
```bash
# Подать пазл на модерацию, войти как admin, одобрить
# Уведомление у родителя → пазл в публичном каталоге
```
