import { useEffect, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { useT } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import type { CatalogPuzzle } from '../../types';

const statusClasses: Record<CatalogPuzzle['status'], string> = {
  processing: 'bg-yellow-100 text-yellow-800',
  ready: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
};

export function CatalogList() {
  const t = useT();
  const [, navigate] = useLocation();
  const [puzzles, setPuzzles] = useState<CatalogPuzzle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    api.admin.catalog
      .list()
      .then(setPuzzles)
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const handleDelete = async (id: string) => {
    if (!confirm('Удалить пазл?')) return;
    try {
      await api.admin.catalog.delete(id);
      load();
    } catch (_) {
      setError(t('common.error'));
    }
  };

  if (loading) return <Spinner />;

  return (
    <div>
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-xl font-semibold text-gray-900">{t('admin.catalog.title')}</h1>
        <button
          onClick={() => navigate('/admin/catalog/new')}
          class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {t('admin.catalog.add')}
        </button>
      </div>

      {error && <p class="mb-4 text-sm text-red-600">{error}</p>}

      {puzzles.length === 0 ? (
        <p class="text-gray-500">{t('catalog.empty')}</p>
      ) : (
        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white">
          <table class="min-w-full border-collapse">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  Превью
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  Название
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  Статус
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  Featured
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500">
                  Действия
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              {puzzles.map((puzzle) => (
                <tr key={puzzle.id} class="even:bg-gray-50 hover:bg-gray-100">
                  <td class="px-4 py-3">
                    <img
                      src={`/api/media/${puzzle.image_key}`}
                      class="w-16 h-16 object-cover rounded"
                      alt=""
                    />
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-900">
                    {puzzle.title || '—'}
                  </td>
                  <td class="px-4 py-3">
                    <span
                      class={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${statusClasses[puzzle.status]}`}
                    >
                      {t(`admin.catalog.status.${puzzle.status}`)}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-900">
                    {puzzle.featured ? '✓' : '—'}
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      onClick={() => navigate(`/admin/catalog/${puzzle.id}/edit`)}
                      class="mr-2 rounded-md bg-blue-600 px-3 py-1 text-sm text-white hover:bg-blue-700"
                    >
                      {t('common.edit')}
                    </button>
                    <button
                      onClick={() => handleDelete(puzzle.id)}
                      class="rounded-md bg-red-600 px-3 py-1 text-sm text-white hover:bg-red-700"
                    >
                      {t('common.delete')}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
