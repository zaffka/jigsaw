import { useEffect, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import type { ParentPuzzle } from '../../types';

const STATUS_LABELS: Record<ParentPuzzle['status'], string> = {
  processing: 'Обработка',
  ready: 'Готов',
  failed: 'Ошибка',
};

const STATUS_COLORS: Record<ParentPuzzle['status'], string> = {
  processing: 'bg-yellow-100 text-yellow-800',
  ready: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
};

const DIFFICULTY_LABELS: Record<string, string> = {
  easy: 'Лёгкий',
  medium: 'Средний',
  hard: 'Сложный',
  '': '',
};

export function PuzzleList() {
  const [, setLocation] = useLocation();
  const [puzzles, setPuzzles] = useState<ParentPuzzle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.parent
      .listPuzzles()
      .then(setPuzzles)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner />;

  return (
    <div>
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-2xl font-semibold text-gray-900">Мои пазлы</h1>
        <button
          onClick={() => setLocation('/parent/puzzles/new')}
          class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          + Создать пазл
        </button>
      </div>

      {error && (
        <div class="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {puzzles.length === 0 ? (
        <div class="py-12 text-center text-gray-500">
          <p class="text-lg">Пазлов пока нет</p>
          <p class="mt-1 text-sm">Создайте первый пазл, нажав кнопку выше</p>
        </div>
      ) : (
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {puzzles.map((puzzle) => (
            <div
              key={puzzle.id}
              onClick={() => setLocation(`/parent/puzzles/${puzzle.id}`)}
              class="cursor-pointer rounded-lg border border-gray-200 bg-white p-4 shadow-sm hover:shadow-md transition-shadow"
            >
              {puzzle.image_key && (
                <div class="mb-3 h-40 overflow-hidden rounded-md bg-gray-100">
                  <img
                    src={`/api/files/${puzzle.image_key}`}
                    alt={puzzle.title}
                    class="h-full w-full object-cover"
                    onError={(e) => {
                      (e.currentTarget as HTMLImageElement).style.display = 'none';
                    }}
                  />
                </div>
              )}
              <h2 class="text-sm font-semibold text-gray-900 truncate">{puzzle.title}</h2>
              <div class="mt-2 flex flex-wrap gap-2">
                <span
                  class={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[puzzle.status]}`}
                >
                  {STATUS_LABELS[puzzle.status]}
                </span>
                {puzzle.difficulty && (
                  <span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
                    {DIFFICULTY_LABELS[puzzle.difficulty]}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
