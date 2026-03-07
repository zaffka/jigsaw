# Архитектура сервиса разрезных картинок и пазлов

Техническая статья описывает подход к реализации веб-сервиса,
в котором ребёнок собирает разрезную картинку на планшете или телефоне,
перетаскивая кусочки пальцем на контуры.

---

## 1. Общая схема

```
┌──────────────┐         JSON/PNG          ┌──────────────────┐
│   Go Backend │  ───────────────────────>  │  Browser Client   │
│              │                            │  (Canvas + SVG)   │
│  slicer pkg  │  <── drag result, events   │                   │
│  HTTP API    │                            │  Touch / Mouse    │
└──────────────┘                            └──────────────────┘
```

**Backend (Go)** — нарезает картинку пакетом `pkg/slicer`, отдаёт клиенту
метаданные и изображения кусочков.

**Frontend (JS/TS)** — рендерит игровое поле, обрабатывает перетаскивание,
проверяет примагничивание.

Основной принцип: **вся геометрия вычисляется на сервере**, клиент получает
готовые данные и занимается только отрисовкой и интерактивностью.

---

## 2. Backend API

### 2.1. Создание игры

```
POST /api/games
{
  "image_id": "ducks",
  "mode": "puzzle",        // grid | merge | geometry | puzzle
  "cols": 3,
  "rows": 3,
  "shape": "mixed",        // для geometry: triangles | diamonds | trapezoids | parallelograms | mixed
  "seed": 42,
  "tab_size": 0.2          // для puzzle
}
```

Сервер вызывает `slicer.Puzzle(img, opts)` (или другой метод в зависимости от `mode`)
и возвращает:

```json
{
  "game_id": "abc123",
  "image_width": 1024,
  "image_height": 1024,
  "pieces": [
    {
      "id": 0,
      "image_url": "/api/games/abc123/pieces/0.png",
      "svg_path": "M 0.00 0.00 L 256.00 0.00 C 256.00 ...",
      "bounds": { "x": 0, "y": 0, "w": 306, "h": 295 },
      "target": { "x": 0, "y": 0 },
      "grid_pos": { "col": 0, "row": 0 }
    }
  ]
}
```

Ключевые поля каждого кусочка:

| Поле | Назначение |
|------|-----------|
| `image_url` | PNG-изображение кусочка с прозрачностью |
| `svg_path` | SVG path data — контур для отрисовки рамки, тени, подсветки |
| `bounds` | Bounding box в координатах исходной картинки |
| `target` | Точка, куда кусочек должен встать (левый верхний угол bounds) |
| `grid_pos` | Позиция в сетке (для игровой логики, подсказок) |

### 2.2. Изображения кусочков

```
GET /api/games/{game_id}/pieces/{piece_id}.png
```

Отдаёт `Piece.Image` — PNG с альфа-каналом. Кусочек уже обрезан
по контуру, фон прозрачный.

### 2.3. Исходная картинка (для фона)

```
GET /api/games/{game_id}/source.jpg       — полная картинка
GET /api/games/{game_id}/silhouette.svg   — контуры всех кусочков
```

Силуэт нужен для отображения целевых контуров на игровом поле.
Генерируется на сервере из `Piece.Outline.SVG()`:

```go
func silhouetteSVG(pieces []slicer.Piece, w, h int) string {
    var b strings.Builder
    fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`, w, h)
    for _, p := range pieces {
        fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#ccc" stroke-width="2"/>`, p.Outline.SVG())
    }
    b.WriteString(`</svg>`)
    return b.String()
}
```

---

## 3. Игровое поле (Frontend)

### 3.1. Структура экрана

```
┌─────────────────────────────────────────┐
│             Целевая зона                │
│  ┌───┬───┬───┐                          │
│  │   │   │   │  Контуры кусочков        │
│  ├───┼───┼───┤  (SVG silhouette или     │
│  │   │   │   │   полупрозрачная          │
│  ├───┼───┼───┤   исходная картинка)     │
│  │   │   │   │                          │
│  └───┴───┴───┘                          │
│                                         │
│─────────────────────────────────────────│
│           Лоток с кусочками             │
│  ┌──┐  ┌──┐  ┌──┐  ┌──┐  ┌──┐          │
│  │1 │  │2 │  │3 │  │4 │  │5 │  ← скролл│
│  └──┘  └──┘  └──┘  └──┘  └──┘          │
└─────────────────────────────────────────┘
```

- **Целевая зона** — верхняя часть экрана. Показывает контуры или
  полупрозрачную оригинальную картинку (настраиваемая подсказка).
- **Лоток** — нижняя полоса с горизонтальным скроллом. Кусочки
  расположены в случайном порядке, уменьшены до миниатюр.

### 3.2. Выбор технологии отрисовки

**Рекомендуемый стек: HTML Canvas для кусочков + SVG для контуров.**

Почему именно Canvas, а не чистый SVG или DOM:

| Критерий | DOM/CSS | SVG | Canvas |
|----------|---------|-----|--------|
| Произвольные формы | Нет | Да | Да |
| Перетаскивание | Просто | Средне | Средне |
| Производительность (50+ элементов) | Хорошо | Средне | Отлично |
| Hit-testing сложных форм | Нет | Встроен | Нужен вручную |
| Масштабирование | CSS | Встроено | Нужен вручную |

Практический компромисс:

- **Контуры целевой зоны** — `<svg>` с `<path>` из `svg_path`. SVG удобен
  для стилизации (штрихи, подсветка, анимация).
- **Кусочки в лотке и при перетаскивании** — `<canvas>`, на который
  рисуются PNG-изображения кусочков. Canvas обеспечивает плавность
  при 60fps на мобильных устройствах.
- **Альтернатива** — использовать только SVG, размещая `<image>` внутри
  `<clipPath>`. Проще в реализации, но медленнее при большом числе кусочков.

### 3.3. Инициализация

```typescript
interface PieceState {
  id: number;
  img: HTMLImageElement;       // загруженный PNG
  svgPath: string;             // контур для рамки
  bounds: Rect;                // bounding box в координатах оригинала
  target: Point;               // куда должен встать
  current: Point;              // текущая позиция на экране
  placed: boolean;             // уже на месте?
  scale: number;               // масштаб отображения
}

async function initGame(gameId: string) {
  const resp = await fetch(`/api/games/${gameId}`);
  const data = await resp.json();

  // Рассчитываем масштаб целевой зоны под размер экрана.
  const targetScale = Math.min(
    targetZoneWidth / data.image_width,
    targetZoneHeight / data.image_height
  );

  // Загружаем все PNG параллельно.
  const pieces: PieceState[] = await Promise.all(
    data.pieces.map(async (p) => {
      const img = new Image();
      img.src = p.image_url;
      await img.decode();
      return {
        id: p.id,
        img,
        svgPath: p.svg_path,
        bounds: p.bounds,
        target: { x: p.target.x * targetScale, y: p.target.y * targetScale },
        current: randomTrayPosition(p.id),
        placed: false,
        scale: targetScale,
      };
    })
  );

  // Перемешиваем порядок в лотке.
  shuffle(pieces);
  layoutTray(pieces);
}
```

---

## 4. Перетаскивание (Drag & Drop)

### 4.1. Pointer Events API

Для единообразной работы с пальцем и мышью используется
[Pointer Events API](https://developer.mozilla.org/en-US/docs/Web/API/Pointer_events).
Он объединяет `mouse`, `touch` и `pen` в один интерфейс.

```typescript
let dragging: PieceState | null = null;
let dragOffset = { x: 0, y: 0 };

canvas.addEventListener('pointerdown', (e) => {
  const pos = canvasPoint(e);
  const piece = hitTest(pos, pieces);
  if (!piece || piece.placed) return;

  dragging = piece;
  dragOffset = { x: pos.x - piece.current.x, y: pos.y - piece.current.y };

  // Поднимаем кусочек наверх (z-order).
  bringToFront(piece, pieces);

  // Захватываем указатель, чтобы не терять события при выходе за canvas.
  canvas.setPointerCapture(e.pointerId);
});

canvas.addEventListener('pointermove', (e) => {
  if (!dragging) return;
  const pos = canvasPoint(e);
  dragging.current = {
    x: pos.x - dragOffset.x,
    y: pos.y - dragOffset.y,
  };
  requestRedraw();
});

canvas.addEventListener('pointerup', (e) => {
  if (!dragging) return;
  trySnap(dragging);
  dragging = null;
  requestRedraw();
});
```

### 4.2. Hit-testing (определение, на какой кусочек нажали)

Для прямоугольных кусочков (Grid) достаточно проверки `point in rect`.
Для произвольных форм — два подхода:

**Подход 1: проверка альфа-канала (простой)**

```typescript
function hitTest(pos: Point, pieces: PieceState[]): PieceState | null {
  // Перебираем от верхнего к нижнему (обратный z-order).
  for (let i = pieces.length - 1; i >= 0; i--) {
    const p = pieces[i];
    if (p.placed) continue;

    // Позиция клика относительно кусочка.
    const lx = (pos.x - p.current.x) / p.scale;
    const ly = (pos.y - p.current.y) / p.scale;

    if (lx < 0 || ly < 0 || lx >= p.img.width || ly >= p.img.height) continue;

    // Читаем альфа пиксела из offscreen canvas.
    offCtx.clearRect(0, 0, 1, 1);
    offCtx.drawImage(p.img, -lx, -ly);
    const alpha = offCtx.getImageData(0, 0, 1, 1).data[3];
    if (alpha > 0) return p;
  }
  return null;
}
```

**Подход 2: point-in-polygon по SVG path (точный)**

Используем `Path2D` и `isPointInPath`:

```typescript
function hitTest(pos: Point, pieces: PieceState[]): PieceState | null {
  for (let i = pieces.length - 1; i >= 0; i--) {
    const p = pieces[i];
    if (p.placed) continue;

    const path = new Path2D(p.svgPath);
    ctx.save();
    ctx.translate(p.current.x, p.current.y);
    ctx.scale(p.scale, p.scale);
    ctx.translate(-p.bounds.x, -p.bounds.y);
    if (ctx.isPointInPath(path, pos.x, pos.y)) {
      ctx.restore();
      return p;
    }
    ctx.restore();
  }
  return null;
}
```

`isPointInPath` с `Path2D` работает в том числе с кривыми Безье
из пазловых контуров — браузер обрабатывает `C` команды нативно.

### 4.3. Визуальная обратная связь при перетаскивании

Для детей-аутистов важна предсказуемая, спокойная обратная связь:

- При захвате — кусочек слегка увеличивается (scale 1.05) и получает тень.
- При движении — плавное следование за пальцем (никаких задержек).
- При поднесении к правильному месту — контур подсвечивается (зелёная обводка).
- При отпускании не на место — кусочек плавно возвращается в лоток.

```typescript
function drawPiece(ctx: CanvasRenderingContext2D, p: PieceState, isDragging: boolean) {
  ctx.save();
  ctx.translate(p.current.x, p.current.y);

  if (isDragging) {
    ctx.scale(1.05, 1.05);
    ctx.shadowColor = 'rgba(0,0,0,0.3)';
    ctx.shadowBlur = 12;
    ctx.shadowOffsetY = 4;
  }

  ctx.drawImage(p.img, 0, 0, p.img.width * p.scale, p.img.height * p.scale);
  ctx.restore();
}
```

---

## 5. Примагничивание (Snap)

### 5.1. Алгоритм

Когда пользователь отпускает кусочек, проверяем расстояние
от текущей позиции до целевой. Если расстояние меньше порога —
кусочек «примагничивается» на место.

```typescript
const SNAP_THRESHOLD = 40; // пикселей на экране

function trySnap(piece: PieceState): boolean {
  const dx = piece.current.x - piece.target.x;
  const dy = piece.current.y - piece.target.y;
  const dist = Math.sqrt(dx * dx + dy * dy);

  if (dist < SNAP_THRESHOLD) {
    // Примагничиваем.
    animateSnap(piece, piece.target);
    piece.placed = true;
    onPiecePlaced(piece);
    return true;
  }

  // Не попал — возвращаем в лоток.
  animateReturn(piece);
  return false;
}
```

### 5.2. Анимация примагничивания

Плавная анимация через `requestAnimationFrame`:

```typescript
function animateSnap(piece: PieceState, target: Point) {
  const start = { ...piece.current };
  const startTime = performance.now();
  const duration = 200; // мс

  function tick(now: number) {
    const t = Math.min((now - startTime) / duration, 1);
    const ease = t * (2 - t); // ease-out

    piece.current = {
      x: start.x + (target.x - start.x) * ease,
      y: start.y + (target.y - start.y) * ease,
    };
    requestRedraw();

    if (t < 1) {
      requestAnimationFrame(tick);
    } else {
      piece.current = target;
      piece.placed = true;
    }
  }

  requestAnimationFrame(tick);
}
```

### 5.3. Адаптивный порог

Для младших детей или начальных уровней порог стоит увеличить:

```typescript
// Сложность влияет на «прощение» при размещении.
const thresholds = {
  easy:   60,  // большая зона захвата
  medium: 40,
  hard:   20,  // нужно положить точно
};
```

### 5.4. Подсветка целевого контура

Когда перетаскиваемый кусочек приближается к своему месту,
подсвечиваем целевой контур:

```typescript
function drawTargetZone(ctx, pieces, dragging) {
  for (const p of pieces) {
    const path = new Path2D(p.svgPath);

    if (dragging && dragging.id === p.id) {
      const dx = dragging.current.x - p.target.x;
      const dy = dragging.current.y - p.target.y;
      const dist = Math.sqrt(dx * dx + dy * dy);

      if (dist < SNAP_THRESHOLD * 1.5) {
        // Близко — зелёная подсветка.
        ctx.strokeStyle = '#4CAF50';
        ctx.lineWidth = 3;
      } else {
        ctx.strokeStyle = '#ccc';
        ctx.lineWidth = 1;
      }
    } else if (p.placed) {
      continue; // Уже на месте — не рисуем контур.
    } else {
      ctx.strokeStyle = '#ddd';
      ctx.lineWidth = 1;
    }

    ctx.stroke(path);
  }
}
```

---

## 6. Игровой цикл отрисовки

Единый render-loop через `requestAnimationFrame`:

```typescript
let needsRedraw = true;

function requestRedraw() {
  needsRedraw = true;
}

function renderLoop() {
  if (needsRedraw) {
    needsRedraw = false;
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // 1. Фон — полупрозрачная исходная картинка (подсказка).
    ctx.globalAlpha = hintOpacity; // 0.1 для сложного, 0.4 для лёгкого
    ctx.drawImage(sourceImage, 0, 0, scaledW, scaledH);
    ctx.globalAlpha = 1.0;

    // 2. Контуры целевых мест.
    drawTargetZone(ctx, pieces, dragging);

    // 3. Уже установленные кусочки.
    for (const p of pieces) {
      if (p.placed) drawPiece(ctx, p, false);
    }

    // 4. Кусочки в лотке.
    drawTray(ctx, pieces);

    // 5. Перетаскиваемый кусочек (поверх всего).
    if (dragging) drawPiece(ctx, dragging, true);
  }
  requestAnimationFrame(renderLoop);
}
```

---

## 7. Лоток с кусочками

### 7.1. Горизонтальный скролл

Лоток — отдельный `<div>` с `overflow-x: scroll` под основным canvas,
или отдельный маленький canvas. Кусочки в нём уменьшены (масштаб 0.3-0.5
от целевого размера) и расположены в ряд.

```typescript
function layoutTray(pieces: PieceState[]) {
  const trayY = targetZoneHeight + 20;
  const trayScale = 0.4;
  let x = 10;

  for (const p of pieces) {
    if (p.placed) continue;
    p.current = { x, y: trayY };
    p.scale = targetScale * trayScale;
    x += p.bounds.w * p.scale + 10;
  }
}
```

### 7.2. Подхват из лотка

При `pointerdown` в зоне лотка кусочек увеличивается до целевого масштаба
и начинает перетаскиваться:

```typescript
canvas.addEventListener('pointerdown', (e) => {
  const pos = canvasPoint(e);
  const piece = hitTest(pos, pieces);
  if (!piece || piece.placed) return;

  // Переключаем масштаб с лоточного на целевой.
  piece.scale = targetScale;

  // Центрируем под пальцем.
  piece.current = {
    x: pos.x - (piece.bounds.w * piece.scale) / 2,
    y: pos.y - (piece.bounds.h * piece.scale) / 2,
  };

  dragging = piece;
  dragOffset = { x: 0, y: 0 };
  // ...
});
```

---

## 8. Завершение игры

Когда все `piece.placed === true`:

```typescript
function onPiecePlaced(piece: PieceState) {
  if (pieces.every(p => p.placed)) {
    // Победа!
    showCompletionAnimation();
    reportCompletion(gameId);
  }
}
```

Анимация завершения: убираем контуры, показываем полную картинку
с плавным fade-in, мягкая вибрация (Vibration API) или звуковой сигнал.

---

## 9. Серверная часть на Go

### 9.1. Кэширование нарезки

Нарезка 1024x1024 картинки пакетом `slicer` занимает ~200-300мс.
Результат стоит кэшировать:

```go
type Game struct {
    ID       string
    Pieces   []slicer.Piece
    Source   image.Image
    Settings GameSettings
}

// Хранилище игр: Redis/Memcached для продакшена, sync.Map для прототипа.
var games sync.Map
```

### 9.2. Минимальный HTTP-обработчик

```go
mux.HandleFunc("POST /api/games", createGame)
mux.HandleFunc("GET /api/games/{id}", getGame)
mux.HandleFunc("GET /api/games/{id}/pieces/{pid}.png", getPiecePNG)
mux.HandleFunc("GET /api/games/{id}/silhouette.svg", getSilhouette)
```

### 9.3. JSON-формат кусочка

```go
type PieceJSON struct {
    ID       int    `json:"id"`
    ImageURL string `json:"image_url"`
    SVGPath  string `json:"svg_path"`
    Bounds   Rect   `json:"bounds"`
    Target   Point  `json:"target"`
    GridPos  Cell   `json:"grid_pos"`
}

func pieceToJSON(gameID string, p slicer.Piece) PieceJSON {
    return PieceJSON{
        ID:       p.ID,
        ImageURL: fmt.Sprintf("/api/games/%s/pieces/%d.png", gameID, p.ID),
        SVGPath:  p.Outline.SVG(),
        Bounds:   Rect{p.Bounds.Min.X, p.Bounds.Min.Y, p.Bounds.Dx(), p.Bounds.Dy()},
        Target:   Point{float64(p.Bounds.Min.X), float64(p.Bounds.Min.Y)},
        GridPos:  Cell{p.GridPos.X, p.GridPos.Y},
    }
}
```

---

## 10. Особенности UX для детей-аутистов

### 10.1. Предсказуемость

- Никаких неожиданных звуков или анимаций.
- Каждое действие имеет чёткий визуальный ответ.
- Кнопка «отмена» всегда доступна и заметна.

### 10.2. Сенсорная чувствительность

- Минимум ярких мигающих эффектов.
- Мягкая цветовая палитра интерфейса (не картинки, а UI-обёртки).
- Опциональные звуки (по умолчанию выключены).
- Настраиваемый фон рабочей области (однотонный, без текстур).

### 10.3. Настраиваемая сложность

- Регулируемый размер сетки (от 2x2 до 8x8).
- Подсказка: прозрачность фоновой картинки (от 0% до 50%).
- Размер зоны примагничивания (щедрый для начинающих).
- Возможность показать номера или цветовые метки на кусочках.

### 10.4. Моторика

- Большие кусочки для тех, у кого сложности с мелкой моторикой.
- Увеличенная зона захвата (touch target минимум 48x48 CSS px).
- `touch-action: none` на canvas, чтобы браузер не перехватывал жесты.

```css
canvas {
  touch-action: none;      /* отключаем зум/скролл на canvas */
  -webkit-user-select: none;
  user-select: none;
}
```

### 10.5. Прогресс и мотивация

- Визуальный индикатор прогресса (сколько кусочков осталось).
- Мягкая похвала при завершении (без перегрузки стимулами).
- Сохранение прогресса — ребёнок может вернуться и продолжить.

---

## 11. Поток данных: от картинки до собранного пазла

```
1. Педагог загружает картинку, выбирает уровень сложности.
          │
          v
2. Backend: slicer.Puzzle(img, opts) -> []Piece
          │
          v
3. Сохраняем Game{pieces, source} в хранилище.
          │
          v
4. Ребёнок открывает ссылку на игру.
          │
          v
5. Frontend: GET /api/games/{id} -> JSON с метаданными.
          │
          v
6. Frontend: загружает PNG кусочков параллельно.
          │
          v
7. Рендерит игровое поле: контуры + лоток с кусочками.
          │
          v
8. Ребёнок перетаскивает кусочки → pointerdown/move/up.
          │
          v
9. trySnap() → примагничивание или возврат.
          │
          v
10. Все placed? → анимация завершения → сохранение результата.
```

---

## 12. Технологический стек (рекомендация)

| Компонент | Технология |
|-----------|-----------|
| Backend | Go, net/http, пакет slicer |
| Хранение картинок | S3-совместимое хранилище или файловая система |
| Кэш игр | Redis или in-memory |
| Frontend | TypeScript, Canvas API, Pointer Events |
| Сборка фронтенда | Vite |
| Деплой | Docker, за Nginx (отдаёт статику + проксирует API) |

---

## 13. Чеклист реализации

Фаза 1 — MVP:
- [ ] Go HTTP сервер с эндпоинтами создания/получения игры
- [ ] Отдача PNG-кусочков и SVG-силуэта
- [ ] Фронтенд: отрисовка целевой зоны с контурами
- [ ] Фронтенд: лоток с кусочками
- [ ] Drag & drop через Pointer Events
- [ ] Snap-логика с фиксированным порогом
- [ ] Экран завершения

Фаза 2 — UX:
- [ ] Анимации (snap, return, completion)
- [ ] Подсветка при приближении к цели
- [ ] Адаптивный layout (телефон/планшет)
- [ ] Настройки сложности в UI педагога

Фаза 3 — Продакшен:
- [ ] Аутентификация (педагог / ребёнок)
- [ ] Сохранение прогресса
- [ ] Аналитика (время сборки, количество попыток)
- [ ] Галерея картинок
- [ ] PWA (offline-режим)
