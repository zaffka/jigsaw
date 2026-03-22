import { useEffect, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { useT } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import type { CatalogPuzzle } from '../../types';

export function CatalogPublic() {
  const t = useT();
  const [, navigate] = useLocation();
  const [puzzles, setPuzzles] = useState<CatalogPuzzle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api.catalog
      .list()
      .then((all) => setPuzzles(all.filter((p) => p.status === 'ready')))
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner />;

  return (
    <div class="mx-auto max-w-4xl px-4 py-8">
      <h1 class="mb-8 text-2xl font-semibold text-gray-900">{t('catalog.title')}</h1>

      {error && <p class="mb-4 text-red-600">{error}</p>}

      {puzzles.length === 0 ? (
        <p class="text-gray-500">Пока нет пазлов</p>
      ) : (
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          {puzzles.map((puzzle) => (
            <button
              key={puzzle.id}
              onClick={() => navigate(`/play/${puzzle.id}`)}
              class="rounded-xl overflow-hidden shadow border border-gray-200 bg-white hover:shadow-lg active:scale-95 transition-all text-left w-full"
            >
              <img
                src={`/api/media/${puzzle.image_key}`}
                class="w-full h-44 object-cover sm:h-48"
                alt={puzzle.title}
              />
              {puzzle.featured && (
                <div class="px-3 pb-2 pt-1">
                  <span class="text-xs text-yellow-600">★ Рекомендуется</span>
                </div>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
