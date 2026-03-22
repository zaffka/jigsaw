import { useEffect, useRef, useState, useCallback } from 'preact/hooks';
import { useParams, useLocation } from 'wouter';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import { RewardScreen } from './RewardScreen';
import type { GamePuzzle, PuzzlePiece } from '../../types';

// ── helpers ────────────────────────────────────────────────────────────────

function imgSize(pieces: PuzzlePiece[]) {
  return {
    w: Math.max(...pieces.map((p) => p.bounds.x + p.bounds.w)),
    h: Math.max(...pieces.map((p) => p.bounds.y + p.bounds.h)),
  };
}

function fitSize(iw: number, ih: number, cw: number, ch: number) {
  if (cw <= 0 || ch <= 0) return { w: 0, h: 0 };
  const scale = Math.min(cw / iw, ch / ih);
  return { w: Math.round(iw * scale), h: Math.round(ih * scale) };
}

interface Layout {
  boardW: number;
  boardH: number;
  boardX: number; // left offset of board within game area
  boardY: number; // top offset of board within game area
  trayY: number;  // top of piece tray
  trayH: number;  // height of piece tray
  areaW: number;  // game area width
}

const HEADER_H = 56;
const PAD = 12;
const GAP = 10;

function buildLayout(iw: number, ih: number): Layout | null {
  const areaW = window.innerWidth;
  const areaH = window.innerHeight - HEADER_H;

  const maxBW = Math.min(areaW - PAD * 2, 720);
  const maxBH = Math.floor((areaH - GAP - PAD * 2) * 0.58);
  const { w: boardW, h: boardH } = fitSize(iw, ih, maxBW, maxBH);
  if (boardW <= 0) return null;

  const boardX = Math.floor((areaW - boardW) / 2);
  const boardY = PAD;
  const trayY = boardY + boardH + GAP;
  const trayH = areaH - trayY - PAD;

  return { boardW, boardH, boardX, boardY, trayY, trayH, areaW };
}

/** Scatter pieces randomly inside the tray area. Positions are in game-area coords. */
function scatterInTray(
  pieces: PuzzlePiece[],
  layout: Layout,
  scale: number,
): Record<string, { x: number; y: number }> {
  const result: Record<string, { x: number; y: number }> = {};
  const { areaW, trayY, trayH } = layout;
  const innerPad = 8;

  pieces.forEach((p) => {
    const pw = p.bounds.w * scale;
    const ph = p.bounds.h * scale;
    const maxX = areaW - pw - innerPad;
    const maxY = trayY + trayH - ph - innerPad;
    result[p.id] = {
      x: innerPad + Math.random() * Math.max(0, maxX - innerPad),
      y: trayY + innerPad + Math.random() * Math.max(0, maxY - trayY - innerPad),
    };
  });
  return result;
}

// ── component ──────────────────────────────────────────────────────────────

export function GameScreen() {
  const { id } = useParams<{ id: string }>();
  const [, navigate] = useLocation();

  const [puzzle, setPuzzle] = useState<GamePuzzle | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [placedIds, setPlacedIds] = useState<Set<string>>(new Set());
  const [solved, setSolved] = useState(false);

  const [layout, setLayout] = useState<Layout | null>(null);
  const positions = useRef<Record<string, { x: number; y: number }>>({});
  const [scattered, setScattered] = useState(false);

  const drag = useRef<{
    pieceId: string;
    el: HTMLElement;
    startPX: number;
    startPY: number;
    startEX: number;
    startEY: number;
    targetX: number;
    targetY: number;
    threshold: number;
  } | null>(null);

  // ── fetch ─────────────────────────────────────────────────────────────────

  useEffect(() => {
    if (!id) return;
    api.catalog
      .get(id)
      .then((p) => setPuzzle(p))
      .catch(() => setError('Не удалось загрузить пазл'))
      .finally(() => setLoading(false));
  }, [id]);

  // ── compute layout from window dimensions ─────────────────────────────────

  useEffect(() => {
    if (!puzzle?.pieces?.length) return;
    const { w: iw, h: ih } = imgSize(puzzle.pieces);

    const compute = () => {
      const l = buildLayout(iw, ih);
      if (l) setLayout(l);
    };

    compute();
    window.addEventListener('resize', compute);
    return () => window.removeEventListener('resize', compute);
  }, [puzzle]);

  // ── scatter pieces into tray once layout is ready ─────────────────────────

  useEffect(() => {
    if (!puzzle?.pieces?.length || !layout || scattered) return;
    const { w: iw } = imgSize(puzzle.pieces);
    const scale = layout.boardW / iw;
    positions.current = scatterInTray(puzzle.pieces, layout, scale);
    setScattered(true);
  }, [puzzle, layout, scattered]);

  // ── global pointer handlers ───────────────────────────────────────────────

  useEffect(() => {
    const onMove = (e: PointerEvent) => {
      if (!drag.current) return;
      const { el, pieceId, startPX, startPY, startEX, startEY } = drag.current;
      const x = startEX + e.clientX - startPX;
      const y = startEY + e.clientY - startPY;
      el.style.left = x + 'px';
      el.style.top = y + 'px';
      positions.current[pieceId] = { x, y };
    };

    const onUp = () => {
      if (!drag.current) return;
      const { el, pieceId, targetX, targetY, threshold } = drag.current;
      drag.current = null;

      const x = parseFloat(el.style.left) || 0;
      const y = parseFloat(el.style.top) || 0;
      const dist = Math.hypot(x - targetX, y - targetY);

      if (dist < threshold) {
        el.style.transition = 'left 0.15s, top 0.15s';
        el.style.left = targetX + 'px';
        el.style.top = targetY + 'px';
        el.style.zIndex = '2';
        el.dataset.placed = 'true';
        positions.current[pieceId] = { x: targetX, y: targetY };
        setPlacedIds((prev) => new Set([...prev, pieceId]));
      } else {
        el.style.zIndex = '10';
        el.style.transition = '';
      }
    };

    window.addEventListener('pointermove', onMove, { passive: true });
    window.addEventListener('pointerup', onUp);
    return () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
  }, []);

  // ── check solved ──────────────────────────────────────────────────────────

  useEffect(() => {
    if (!puzzle?.pieces?.length) return;
    if (placedIds.size >= puzzle.pieces.length) {
      setTimeout(() => setSolved(true), 700);
    }
  }, [placedIds, puzzle]);

  // ── drag start ────────────────────────────────────────────────────────────

  const onPointerDown = useCallback(
    (e: PointerEvent, p: PuzzlePiece, targetX: number, targetY: number, pw: number, ph: number) => {
      const el = e.currentTarget as HTMLElement;
      if (el.dataset.placed === 'true') return;
      e.preventDefault();
      el.setPointerCapture(e.pointerId);
      el.style.zIndex = '100';
      el.style.transition = '';
      drag.current = {
        pieceId: p.id,
        el,
        startPX: e.clientX,
        startPY: e.clientY,
        startEX: parseFloat(el.style.left) || 0,
        startEY: parseFloat(el.style.top) || 0,
        targetX,
        targetY,
        threshold: Math.min(pw, ph) * 0.45,
      };
    },
    [],
  );

  // ── replay ────────────────────────────────────────────────────────────────

  const handleReplay = useCallback(() => {
    positions.current = {};
    setScattered(false);
    setPlacedIds(new Set());
    setSolved(false);
  }, []);

  // ── render ────────────────────────────────────────────────────────────────

  if (loading)
    return (
      <div class="flex h-dvh items-center justify-center">
        <Spinner />
      </div>
    );

  if (error || !puzzle)
    return <p class="p-6 text-red-600">{error || 'Пазл не найден'}</p>;

  if (solved)
    return <RewardScreen reward={puzzle.reward ?? null} onReplay={handleReplay} />;

  const pieces = puzzle.pieces ?? [];
  const { w: iw, h: ih } = pieces.length ? imgSize(pieces) : { w: 1, h: 1 };
  const scale = layout ? layout.boardW / iw : 0;
  const areaH = window.innerHeight - HEADER_H;

  return (
    <div class="flex h-dvh flex-col">
      {/* Header */}
      <header
        class="flex shrink-0 items-center gap-3 bg-white px-4 shadow-sm"
        style={{ height: HEADER_H + 'px' }}
      >
        <button
          onClick={() => navigate('/catalog')}
          class="flex h-8 w-8 items-center justify-center rounded-full text-gray-500 hover:bg-gray-100 active:bg-gray-200"
          aria-label="Назад"
        >
          ←
        </button>
        <span class="flex-1" />
<span class="shrink-0 rounded-full bg-blue-100 px-3 py-0.5 text-sm font-medium text-blue-700">
          {placedIds.size}&nbsp;/&nbsp;{pieces.length}
        </span>
      </header>

      {/* Game area — single relative container; all pieces absolutely positioned inside */}
      <div
        class="relative overflow-hidden touch-none bg-slate-100"
        style={{ width: '100%', height: areaH + 'px' }}
      >
        {layout && (
          <>
            {/* Board: SVG silhouette */}
            <svg
              class="pointer-events-none absolute rounded-xl shadow-md"
              width={layout.boardW}
              height={layout.boardH}
              viewBox={`0 0 ${iw} ${ih}`}
              style={{
                left: layout.boardX + 'px',
                top: layout.boardY + 'px',
                background: 'white',
              }}
            >
              {pieces.map((p) => (
                <path
                  key={p.id}
                  d={p.svg_path}
                  fill={placedIds.has(p.id) ? 'rgba(74,222,128,0.2)' : 'rgba(0,0,0,0)'}
                  stroke={placedIds.has(p.id) ? '#4ade80' : '#cbd5e1'}
                  stroke-width={Math.max(0.8, 2 / scale)}
                />
              ))}
            </svg>

            {/* Tray: background hint for piece storage area */}
            {layout.trayH > 20 && (
              <div
                class="absolute rounded-xl border border-amber-200/60 bg-amber-50/70"
                style={{
                  left: PAD + 'px',
                  top: layout.trayY + 'px',
                  width: layout.areaW - PAD * 2 + 'px',
                  height: layout.trayH + 'px',
                }}
              />
            )}

            {/* Draggable pieces — positioned in game-area coords */}
            {scattered &&
              pieces.map((p) => {
                const pw = p.bounds.w * scale;
                const ph = p.bounds.h * scale;
                // Target position = board offset + piece position on board
                const tx = layout.boardX + p.bounds.x * scale;
                const ty = layout.boardY + p.bounds.y * scale;
                const pos = positions.current[p.id] ?? { x: tx, y: ty };
                const placed = placedIds.has(p.id);

                return (
                  <img
                    key={p.id}
                    src={`/api/media/${p.image_key}`}
                    draggable={false}
                    data-placed={String(placed)}
                    class={`absolute select-none ${
                      placed ? 'cursor-default' : 'cursor-grab active:cursor-grabbing'
                    }`}
                    style={{
                      width: pw + 'px',
                      height: ph + 'px',
                      left: pos.x + 'px',
                      top: pos.y + 'px',
                      zIndex: placed ? 2 : 10,
                      touchAction: 'none',
                    }}
                    onPointerDown={(e) => onPointerDown(e, p, tx, ty, pw, ph)}
                  />
                );
              })}
          </>
        )}
      </div>
    </div>
  );
}
