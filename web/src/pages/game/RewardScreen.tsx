import { useEffect, useRef } from 'preact/hooks';
import { useLocation } from 'wouter';
import type { Reward } from '../../types';

interface Props {
  reward: Reward | null;
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

export function RewardScreen({ reward, onReplay }: Props) {
  const [, navigate] = useLocation();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animRef = useRef<number>(0);
  const particles = useRef<Particle[]>([]);

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
      // Slow down after 3s
      if (frame < 180 || frame % 2 === 0) {
        animRef.current = requestAnimationFrame(tick);
      } else {
        animRef.current = requestAnimationFrame(tick);
      }
    };

    animRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(animRef.current);
  }, []);

  return (
    <div class="relative flex h-dvh flex-col items-center justify-center overflow-hidden bg-gradient-to-b from-yellow-50 to-orange-50">
      {/* Confetti canvas */}
      <canvas
        ref={canvasRef}
        class="pointer-events-none absolute inset-0 h-full w-full"
      />

      {/* Content */}
      <div class="relative z-10 flex flex-col items-center gap-6 px-8 text-center">
        <div class="animate-bounce text-6xl sm:text-8xl">🎉</div>

        {reward?.word && (
          <div
            class="rounded-2xl bg-white/90 px-8 py-4 shadow-xl backdrop-blur-sm"
            style={{ animation: 'wordPop 0.5s cubic-bezier(0.34,1.56,0.64,1) both' }}
          >
            <p class="text-5xl font-bold text-gray-800 sm:text-7xl">{reward.word}</p>
          </div>
        )}

        {!reward?.word && (
          <p class="text-3xl font-bold text-gray-700 sm:text-4xl">Молодец! 🌟</p>
        )}

        {reward?.video_key && (
          <video
            src={`/api/media/${reward.video_key}`}
            autoPlay
            playsInline
            loop
            class="mt-2 max-h-48 max-w-xs rounded-2xl shadow-lg sm:max-h-64 sm:max-w-sm"
          />
        )}

        <div class="mt-4 flex flex-col gap-3 sm:flex-row">
          <button
            onClick={onReplay}
            class="rounded-2xl bg-blue-500 px-8 py-3 text-lg font-semibold text-white shadow-lg active:scale-95 hover:bg-blue-600 transition-all"
          >
            Ещё раз
          </button>
          <button
            onClick={() => navigate('/catalog')}
            class="rounded-2xl border-2 border-gray-300 bg-white px-8 py-3 text-lg font-semibold text-gray-700 shadow active:scale-95 hover:bg-gray-50 transition-all"
          >
            В каталог
          </button>
        </div>
      </div>

      <style>{`
        @keyframes wordPop {
          from { opacity: 0; transform: scale(0.3); }
          to   { opacity: 1; transform: scale(1); }
        }
      `}</style>
    </div>
  );
}
