import { useEffect, useState } from 'preact/hooks';
import { useLocation, useParams } from 'wouter';
import { api } from '../../api';
import { useT } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import type { CatalogPuzzle } from '../../types';

export function CatalogEdit() {
  const t = useT();
  const [, setLocation] = useLocation();
  const params = useParams<{ id: string }>();
  const id = params.id;

  const [puzzle, setPuzzle] = useState<CatalogPuzzle | null>(null);
  const [titleRu, setTitleRu] = useState('');
  const [titleEn, setTitleEn] = useState('');
  const [featured, setFeatured] = useState(false);
  const [sortOrder, setSortOrder] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!id) return;
    api.admin.catalog
      .list()
      .then((list) => {
        const found = list.find((p) => p.id === id);
        if (found) {
          setPuzzle(found);
          setTitleRu(found.titles['ru'] ?? '');
          setTitleEn(found.titles['en'] ?? '');
          setFeatured(found.featured);
          setSortOrder(found.sort_order);
        }
      })
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false));
  }, [id]);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    if (!id) return;
    setError('');
    setSaving(true);
    try {
      await api.admin.catalog.update(id, {
        titles: { ru: titleRu, en: titleEn },
        featured,
        sort_order: sortOrder,
      });
      setLocation('/admin/catalog');
    } catch (_) {
      setError(t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Spinner />;
  if (!puzzle) return <p class="text-red-600">{t('common.error')}</p>;

  return (
    <div class="max-w-lg">
      <h1 class="mb-6 text-xl font-semibold text-gray-900">
        {t('common.edit')}
      </h1>

      <form onSubmit={handleSubmit} class="space-y-4 rounded-lg border border-gray-200 bg-white p-6">
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            {t('admin.catalog.form.titleRu')}
          </label>
          <input
            type="text"
            required
            value={titleRu}
            onInput={(e) => setTitleRu((e.target as HTMLInputElement).value)}
            class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            {t('admin.catalog.form.titleEn')}
          </label>
          <input
            type="text"
            value={titleEn}
            onInput={(e) => setTitleEn((e.target as HTMLInputElement).value)}
            class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div class="flex items-center gap-2">
          <input
            type="checkbox"
            id="featured"
            checked={featured}
            onChange={(e) => setFeatured((e.target as HTMLInputElement).checked)}
            class="h-4 w-4 rounded border-gray-300 text-blue-600"
          />
          <label for="featured" class="text-sm font-medium text-gray-700">
            {t('admin.catalog.form.featured')}
          </label>
        </div>

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            {t('admin.catalog.form.sortOrder')}
          </label>
          <input
            type="number"
            value={sortOrder}
            onInput={(e) => setSortOrder(Number((e.target as HTMLInputElement).value))}
            class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        {error && <p class="text-sm text-red-600">{error}</p>}

        <div class="flex gap-3">
          <button
            type="submit"
            disabled={saving}
            class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? t('common.loading') : t('admin.catalog.form.save')}
          </button>
          <button
            type="button"
            onClick={() => setLocation('/admin/catalog')}
            class="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {t('admin.catalog.form.cancel')}
          </button>
        </div>
      </form>
    </div>
  );
}
