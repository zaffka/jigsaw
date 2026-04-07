import { useEffect, useRef, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import type { PuzzleLayer } from '../../types';

interface Props {
  layers: PuzzleLayer[];
  onReplay: () => void;
}

const CONFETTI_COLORS = ['#f43f5e', '#f97316', '#eab308', '#22c55e', '#3b82f6', '#a855f7', '#ec4899'];
const PARTICLE_COUNT = 60;

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  color: string;
  size: number;
  rotation: number;
  rotSpeed: number;
  shape: 'rect' | 'circle';
}

function makeParticles(w: number): Particle[] {
  return Array.from({ length: PARTICLE_COUNT }, () => ({
    x: Math.random() * w,
    y: -20 - Math.random() * 80,
    vx: (Math.random() - 0.5) * 3,
    vy: 2 + Math.random() * 4,
    color: CONFETTI_COLORS[Math.floor(Math.random() * CONFETTI_COLORS.length)],
    size: 6 + Math.random() * 8,
    rotation: Math.random() * 360,
    rotSpeed: (Math.random() - 0.5) * 8,
    shape: Math.random() > 0.5 ? 'rect' : 'circle',
  }));
}

export function RewardScreen({ layers, onReplay }: Props) {
  const [, navigate] = useLocation();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animRef = useRef<number>(0);
  const particles = useRef<Particle[]>([]);
  const [phase, setPhase] = useState<'confetti' | 'layers' | 'done'>('confetti');
  const [layerIndex, setLayerIndex] = useState(0);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const resize = () => {
      canvas.width = canvas.offsetWidth;
      canvas.height = canvas.offsetHeight;
    };
    resize();

    particles.current = makeParticles(canvas.width);
    let frame = 0;

    const tick = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      particles.current.forEach((p) => {
        p.x += p.vx;
        p.y += p.vy;
        p.vy += 0.05; // gravity
        p.rotation += p.rotSpeed;

        if (p.y > canvas.height + 20) {
          p.y = -20;
          p.x = Math.random() * canvas.width;
          p.vy = 2 + Math.random() * 4;
        }

        ctx.save();
        ctx.translate(p.x, p.y);
        ctx.rotate((p.rotation * Math.PI) / 180);
        ctx.fillStyle = p.color;
        ctx.globalAlpha = 0.85;
        if (p.shape === 'circle') {
          ctx.beginPath();
          ctx.arc(0, 0, p.size / 2, 0, Math.PI * 2);
          ctx.fill();
        } else {
          ctx.fillRect(-p.size / 2, -p.size / 4, p.size, p.size / 2);
        }
        ctx.restore();
      });

      frame++;
      // Run every frame for 3s, then every other frame to save resources
      if (frame < 180 || frame % 2 === 0) {
        animRef.current = requestAnimationFrame(tick);
      }
    };

    animRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(animRef.current);
  }, []);

  const advance = () => {
    if (phase === 'confetti') {
      if (layers.length === 0) {
        setPhase('done');
      } else {
        setPhase('layers');
        setLayerIndex(0);
      }
    } else if (phase === 'layers') {
      if (layerIndex + 1 < layers.length) {
        setLayerIndex((i) => i + 1);
      } else {
        setPhase('done');
      }
    }
  };

  const currentLayer = phase === 'layers' ? layers[layerIndex] : null;

  return (
    <div class="relative flex h-dvh flex-col items-center justify-center overflow-hidden bg-gradient-to-b from-yellow-50 to-orange-50">
      {/* Confetti canvas — always rendered */}
      <canvas ref={canvasRef} class="pointer-events-none absolute inset-0 h-full w-full" />

      {/* Confetti phase */}
      {phase === 'confetti' && (
        <div
          class="relative z-10 flex h-full w-full cursor-pointer select-none flex-col items-center justify-center gap-6 px-8 text-center"
          onClick={advance}
        >
          <div class="animate-bounce text-6xl sm:text-8xl">🎉</div>
          <p class="text-3xl font-bold text-gray-700 sm:text-4xl">Молодец! 🌟</p>
          <p class="mt-4 text-sm text-gray-400">Нажми, чтобы продолжить</p>
        </div>
      )}

      {/* Word layer */}
      {phase === 'layers' && currentLayer?.type === 'word' && (
        <div
          class="relative z-10 flex h-full w-full cursor-pointer select-none flex-col items-center justify-center gap-6 px-8 text-center"
          onClick={advance}
        >
          <div
            class="rounded-2xl bg-white/90 px-8 py-6 shadow-xl backdrop-blur-sm"
            style={{ animation: 'wordPop 0.5s cubic-bezier(0.34,1.56,0.64,1) both' }}
          >
            <p class="text-6xl font-bold text-gray-800 sm:text-8xl">{currentLayer.text}</p>
          </div>
          <p class="text-sm text-gray-400">Нажми, чтобы продолжить</p>
        </div>
      )}

      {/* Audio layer */}
      {phase === 'layers' && currentLayer?.type === 'audio' && (
        <div class="relative z-10 flex h-full w-full flex-col items-center justify-center gap-6 px-8 text-center">
          <div class="cursor-pointer select-none text-8xl" onClick={advance}>🔊</div>
          {currentLayer.audio_key && (
            <audio
              key={currentLayer.id}
              src={`/api/media/${currentLayer.tts_key ?? currentLayer.audio_key}`}
              autoPlay
              onEnded={advance}
            />
          )}
          <p class="text-sm text-gray-400">Нажми, чтобы пропустить</p>
        </div>
      )}

      {/* Video layer */}
      {phase === 'layers' && currentLayer?.type === 'video' && (
        <div class="relative z-10 flex h-full w-full flex-col items-center justify-center gap-4 px-4">
          {currentLayer.video_key && (
            <video
              key={currentLayer.id}
              src={`/api/media/${currentLayer.video_key}`}
              autoPlay
              playsInline
              controls
              onEnded={advance}
              class="max-h-[70vh] max-w-full rounded-2xl shadow-xl"
            />
          )}
          <button onClick={advance} class="text-sm text-gray-400 underline">
            Пропустить
          </button>
        </div>
      )}

      {/* Done phase */}
      {phase === 'done' && (
        <div class="relative z-10 flex flex-col items-center gap-4">
          <button
            onClick={onReplay}
            class="rounded-2xl bg-blue-500 px-8 py-3 text-lg font-semibold text-white shadow-lg transition-all hover:bg-blue-600 active:scale-95"
          >
            Ещё раз
          </button>
          <button
            onClick={() => navigate('/catalog')}
            class="rounded-2xl border-2 border-gray-300 bg-white px-8 py-3 text-lg font-semibold text-gray-700 shadow transition-all hover:bg-gray-50 active:scale-95"
          >
            В каталог
          </button>
        </div>
      )}

      <style>{`
        @keyframes wordPop {
          from { opacity: 0; transform: scale(0.3); }
          to   { opacity: 1; transform: scale(1); }
        }
      `}</style>
    </div>
  );
}
