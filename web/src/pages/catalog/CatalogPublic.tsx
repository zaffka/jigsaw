import { useEffect, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { useT, useLocale } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import { TopActions } from '../../components/TopActions';
import type { CatalogPuzzle, Category } from '../../types';

type Difficulty = 'easy' | 'medium' | 'hard';

const DIFFICULTIES: Difficulty[] = ['easy', 'medium', 'hard'];

export function CatalogPublic() {
  const t = useT();
  const locale = useLocale();
  const [, navigate] = useLocation();

  const [puzzles, setPuzzles] = useState<CatalogPuzzle[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [category, setCategory] = useState('');
  const [difficulty, setDifficulty] = useState('');
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());

  // Load completed puzzle IDs for the current child (best-effort)
  useEffect(() => {
    if (!sessionStorage.getItem('child_token')) return;
    api.play.completed()
      .then((ids) => setCompletedIds(new Set(ids)))
      .catch(() => {}); // best-effort
  }, []);

  // Load categories once
  useEffect(() => {
    api.categories.list().catch(() => []).then((cats) => {
      if (Array.isArray(cats)) setCategories(cats);
    });
  }, []);

  // Reload puzzles whenever filters change
  useEffect(() => {
    setLoading(true);
    setError('');
    api.catalog
      .list({ category: category || undefined, difficulty: difficulty || undefined })
      .then((all) => setPuzzles(all))
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false));
  }, [category, difficulty]);

  const chipBase =
    'rounded-full border px-3 py-1 text-sm font-medium transition-colors cursor-pointer select-none';
  const chipActive = 'border-blue-500 bg-blue-500 text-white';
  const chipIdle = 'border-gray-300 bg-white text-gray-700 hover:border-blue-400';

  return (
    <div class="mx-auto max-w-4xl px-4 py-8">
      <TopActions />
      <h1 class="mb-6 text-2xl font-semibold text-gray-900">{t('catalog.title')}</h1>

      {/* Difficulty filter */}
      <div class="mb-3 flex flex-wrap gap-2">
        <button
          class={`${chipBase} ${difficulty === '' ? chipActive : chipIdle}`}
          onClick={() => setDifficulty('')}
        >
          {t('catalog.filter.all')}
        </button>
        {DIFFICULTIES.map((d) => (
          <button
            key={d}
            class={`${chipBase} ${difficulty === d ? chipActive : chipIdle}`}
            onClick={() => setDifficulty(d === difficulty ? '' : d)}
          >
            {t(`catalog.filter.difficulty.${d}`)}
          </button>
        ))}
      </div>

      {/* Category filter */}
      {categories.length > 0 && (
        <div class="mb-6 flex flex-wrap gap-2">
          <button
            class={`${chipBase} ${category === '' ? chipActive : chipIdle}`}
            onClick={() => setCategory('')}
          >
            {t('catalog.filter.allCategories')}
          </button>
          {categories.map((c) => (
            <button
              key={c.slug}
              class={`${chipBase} ${category === c.slug ? chipActive : chipIdle}`}
              onClick={() => setCategory(c.slug === category ? '' : c.slug)}
            >
              {c.icon} {c.name[locale] ?? c.name['ru'] ?? c.slug}
            </button>
          ))}
        </div>
      )}

      {error && <p class="mb-4 text-red-600">{error}</p>}

      {loading ? (
        <div class="flex justify-center py-12">
          <Spinner />
        </div>
      ) : puzzles.length === 0 ? (
        <p class="text-gray-500">{t('catalog.empty')}</p>
      ) : (
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          {puzzles.map((puzzle) => (
            <button
              key={puzzle.id}
              onClick={() => navigate(`/play/${puzzle.id}`)}
              class="relative rounded-xl overflow-hidden shadow border border-gray-200 bg-white hover:shadow-lg active:scale-95 transition-all text-left w-full"
            >
              <img
                src={`/api/media/${puzzle.image_key}`}
                class="w-full h-44 object-cover sm:h-48"
                alt={puzzle.title}
              />

              {/* Completed checkmark */}
              {completedIds.has(puzzle.id) && (
                <div class="absolute inset-0 bg-green-500/20 flex items-start justify-end p-2">
                  <span class="bg-green-500 text-white rounded-full w-8 h-8 flex items-center justify-center text-lg font-bold shadow">✓</span>
                </div>
              )}

              <div class="px-3 pb-3 pt-2 flex items-center justify-between gap-2">
                {puzzle.featured && (
                  <span class="text-xs text-yellow-600">★ {t('catalog.featured')}</span>
                )}
                {puzzle.difficulty && (
                  <span class={`ml-auto text-xs font-medium rounded-full px-2 py-0.5 ${
                    puzzle.difficulty === 'easy'
                      ? 'bg-green-100 text-green-700'
                      : puzzle.difficulty === 'medium'
                        ? 'bg-yellow-100 text-yellow-700'
                        : 'bg-red-100 text-red-700'
                  }`}>
                    {t(`catalog.filter.difficulty.${puzzle.difficulty}`)}
                  </span>
                )}
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
