import { useEffect, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../api';
import { Spinner } from '../components/Spinner';
import type { Child } from '../types';

type Stage =
  | { kind: 'selecting' }
  | { kind: 'entering_pin'; child: Child; pin: string; error: string; shake: boolean }
  | { kind: 'submitting'; child: Child; pin: string };

const DIGIT_BUTTONS = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0'];

export function ChildSelect() {
  const [, navigate] = useLocation();

  const [children, setChildren] = useState<Child[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [stage, setStage] = useState<Stage>({ kind: 'selecting' });

  const loadChildren = () => {
    setLoading(true);
    setLoadError(null);
    api.parent
      .listChildren()
      .then(setChildren)
      .catch((e: Error) => setLoadError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadChildren();
  }, []);

  const handleSelectChild = (child: Child) => {
    setStage({ kind: 'entering_pin', child, pin: '', error: '', shake: false });
  };

  const handleBack = () => {
    setStage({ kind: 'selecting' });
  };

  const handleDigit = (digit: string) => {
    if (stage.kind !== 'entering_pin') return;
    const newPin = stage.pin + digit;
    if (newPin.length > 4) return;

    if (newPin.length === 4) {
      // Auto-submit
      const child = stage.child;
      setStage({ kind: 'submitting', child, pin: newPin });
      api.children
        .auth(child.id, newPin)
        .then(({ token }) => {
          sessionStorage.setItem('child_token', token);
          navigate('/catalog');
        })
        .catch((e: Error) => {
          setStage({
            kind: 'entering_pin',
            child,
            pin: '',
            error: 'Неверный PIN',
            shake: true,
          });
          // Clear shake after animation
          setTimeout(() => {
            setStage((prev) =>
              prev.kind === 'entering_pin' ? { ...prev, shake: false } : prev,
            );
          }, 600);
          console.warn('child auth failed:', e.message);
        });
    } else {
      setStage({ ...stage, pin: newPin, error: '', shake: false });
    }
  };

  const handleBackspace = () => {
    if (stage.kind !== 'entering_pin') return;
    setStage({ ...stage, pin: stage.pin.slice(0, -1), error: '', shake: false });
  };

  if (loading) {
    return (
      <div class="flex min-h-screen items-center justify-center bg-sky-50">
        <Spinner />
      </div>
    );
  }

  if (loadError) {
    return (
      <div class="flex min-h-screen flex-col items-center justify-center gap-4 bg-sky-50 p-6">
        <p class="text-center text-lg text-red-600">{loadError}</p>
        <button
          onClick={loadChildren}
          class="rounded-2xl bg-blue-500 px-8 py-4 text-lg font-semibold text-white active:scale-95"
        >
          Повторить
        </button>
      </div>
    );
  }

  if (children.length === 0) {
    return (
      <div class="flex min-h-screen flex-col items-center justify-center gap-4 bg-sky-50 p-6 text-center">
        <span class="text-6xl">👨‍👩‍👧</span>
        <p class="text-xl font-semibold text-gray-700">
          Нет профилей детей.
        </p>
        <p class="text-gray-500">
          Добавьте ребёнка в родительском кабинете.
        </p>
        <a
          href="/parent/children"
          class="mt-2 rounded-2xl bg-blue-500 px-8 py-4 text-lg font-semibold text-white active:scale-95"
        >
          Перейти в кабинет
        </a>
      </div>
    );
  }

  // PIN entry overlay
  if (stage.kind === 'entering_pin' || stage.kind === 'submitting') {
    const child = stage.child;
    const pin = stage.kind === 'entering_pin' ? stage.pin : stage.pin;
    const error = stage.kind === 'entering_pin' ? stage.error : '';
    const shake = stage.kind === 'entering_pin' ? stage.shake : false;
    const isSubmitting = stage.kind === 'submitting';

    return (
      <div class="flex min-h-screen flex-col items-center justify-center bg-sky-50 p-6">
        {/* Child info */}
        <div class="mb-8 flex flex-col items-center gap-2">
          <span class="text-7xl">{child.avatar_emoji}</span>
          <p class="text-2xl font-bold text-gray-800">{child.name}</p>
        </div>

        {/* PIN dots */}
        <div
          class={`mb-4 flex gap-5 ${shake ? 'animate-[shake_0.5s_ease-in-out]' : ''}`}
        >
          {[0, 1, 2, 3].map((i) => (
            <div
              key={i}
              class={`h-5 w-5 rounded-full border-2 transition-all duration-150 ${
                i < pin.length
                  ? 'border-blue-500 bg-blue-500'
                  : 'border-gray-400 bg-transparent'
              }`}
            />
          ))}
        </div>

        {/* Error message */}
        {error && (
          <p class="mb-4 text-center text-base font-medium text-red-500">{error}</p>
        )}

        {/* Numpad */}
        <div class="grid grid-cols-3 gap-3 mb-6">
          {DIGIT_BUTTONS.map((d) => (
            <button
              key={d}
              onClick={() => handleDigit(d)}
              disabled={isSubmitting}
              class={`flex h-16 w-16 items-center justify-center rounded-full bg-white text-2xl font-bold text-gray-800 shadow-md transition active:scale-90 disabled:opacity-50 ${
                d === '0' ? 'col-start-2' : ''
              }`}
            >
              {d}
            </button>
          ))}
        </div>

        {/* Backspace */}
        <button
          onClick={handleBackspace}
          disabled={isSubmitting || pin.length === 0}
          class="mb-8 flex h-14 w-14 items-center justify-center rounded-full bg-gray-200 text-xl text-gray-700 shadow active:scale-90 disabled:opacity-30"
          aria-label="Стереть"
        >
          ⌫
        </button>

        {/* Back button */}
        <button
          onClick={handleBack}
          disabled={isSubmitting}
          class="text-base font-medium text-gray-400 active:text-gray-600 disabled:opacity-50"
        >
          ← Назад
        </button>

        {isSubmitting && (
          <div class="mt-6">
            <Spinner />
          </div>
        )}
      </div>
    );
  }

  // Child selection screen
  return (
    <div class="flex min-h-screen flex-col items-center justify-center bg-sky-50 p-6">
      <h1 class="mb-8 text-3xl font-bold text-gray-800">Кто играет?</h1>
      <div class="flex w-full max-w-3xl gap-5 overflow-x-auto pb-4 justify-center flex-wrap">
        {children.map((child) => (
          <button
            key={child.id}
            onClick={() => handleSelectChild(child)}
            class="flex min-w-[120px] flex-col items-center gap-3 rounded-3xl bg-white p-6 shadow-lg transition active:scale-95 hover:shadow-xl"
          >
            <span class="text-6xl">{child.avatar_emoji}</span>
            <span class="text-center text-lg font-semibold text-gray-800">{child.name}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
