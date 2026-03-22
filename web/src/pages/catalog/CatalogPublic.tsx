import { useEffect, useState } from 'preact/hooks';
import { api } from '../../api';
import { useT, useLocale } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import type { CatalogPuzzle } from '../../types';

export function CatalogPublic() {
  const t = useT();
  const locale = useLocale();
  const [puzzles, setPuzzles] = useState<CatalogPuzzle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api.catalog
      .list()
      .then(setPuzzles)
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner />;

  return (
    <div class="mx-auto max-w-4xl px-4 py-8">
      <h1 class="mb-8 text-2xl font-semibold text-gray-900">{t('catalog.title')}</h1>

      {error && <p class="mb-4 text-red-600">{error}</p>}

      {puzzles.length === 0 ? (
        <p class="text-gray-500">{t('catalog.empty')}</p>
      ) : (
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          {puzzles.map((puzzle) => (
            <div
              key={puzzle.id}
              class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm hover:shadow-md transition-shadow"
            >
              <div class="mb-3 flex h-32 items-center justify-center rounded-md bg-gray-100 text-xs text-gray-400">
                {puzzle.image_key}
              </div>
              <h2 class="font-medium text-gray-900">
                {puzzle.titles[locale] ?? puzzle.titles['ru'] ?? puzzle.titles['en'] ?? '—'}
              </h2>
              {puzzle.status !== 'ready' && (
                <span class="mt-1 inline-block text-xs text-gray-500">
                  {t(`admin.catalog.status.${puzzle.status}`)}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
