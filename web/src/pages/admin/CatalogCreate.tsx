import { useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { useT } from '../../i18n';
import type { Locale } from '../../types';

type PuzzleMode = 'grid' | 'merge' | 'geometry' | 'puzzle';

const LOCALES: { value: Locale; label: string }[] = [
  { value: 'ru', label: 'Русский' },
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Español' },
  { value: 'zh', label: '中文' },
  { value: 'th', label: 'ภาษาไทย' },
];

export function CatalogCreate() {
  const t = useT();
  const [, navigate] = useLocation();
  const [title, setTitle] = useState('');
  const [locale, setLocale] = useState<Locale>('ru');
  const [cols, setCols] = useState('4');
  const [rows, setRows] = useState('3');
  const [mode, setMode] = useState<PuzzleMode>('grid');
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    if (!file) return;

    setError('');
    setLoading(true);

    try {
      const formData = new FormData();
      formData.append('image', file);
      formData.append('title', title);
      formData.append('locale', locale);
      formData.append('config', JSON.stringify({ mode, cols: +cols, rows: +rows }));

      await api.admin.catalog.create(formData);
      navigate('/admin/catalog');
    } catch (_) {
      setError(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="max-w-lg">
      <h1 class="mb-6 text-xl font-semibold text-gray-900">
        {t('admin.catalog.form.upload')}
      </h1>

      <form onSubmit={handleSubmit} class="space-y-4 rounded-lg border border-gray-200 bg-white p-6">
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            {t('admin.catalog.form.image')}
          </label>
          <input
            type="file"
            accept="image/jpeg,image/png"
            required
            onChange={(e) => {
              const f = (e.target as HTMLInputElement).files?.[0];
              if (f) {
                setFile(f);
                setError('');
              }
            }}
            class="w-full border rounded px-3 py-2 text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-blue-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
          />
        </div>

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            {t('admin.catalog.form.locale')}
          </label>
          <select
            value={locale}
            onChange={(e) => setLocale((e.target as HTMLSelectElement).value as Locale)}
            class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            {LOCALES.map((l) => (
              <option key={l.value} value={l.value}>{l.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            {t('admin.catalog.form.title')}
          </label>
          <input
            type="text"
            required
            value={title}
            onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
            class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div class="flex gap-4">
          <div class="flex-1">
            <label class="mb-1 block text-sm font-medium text-gray-700">Колонки</label>
            <input
              type="number"
              min="1"
              max="20"
              value={cols}
              onInput={(e) => setCols((e.target as HTMLInputElement).value)}
              class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div class="flex-1">
            <label class="mb-1 block text-sm font-medium text-gray-700">Строки</label>
            <input
              type="number"
              min="1"
              max="20"
              value={rows}
              onInput={(e) => setRows((e.target as HTMLInputElement).value)}
              class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        </div>

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">Режим нарезки</label>
          <select
            value={mode}
            onChange={(e) => setMode((e.target as HTMLSelectElement).value as PuzzleMode)}
            class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="grid">Grid</option>
            <option value="merge">Merge</option>
            <option value="geometry">Geometry</option>
            <option value="puzzle">Puzzle</option>
          </select>
        </div>

        {error && <p class="text-sm text-red-600">{error}</p>}

        <div class="flex gap-3">
          <button
            type="submit"
            disabled={loading || !file}
            class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? t('common.loading') : t('admin.catalog.form.save')}
          </button>
          <button
            type="button"
            onClick={() => navigate('/admin/catalog')}
            class="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {t('admin.catalog.form.cancel')}
          </button>
        </div>
      </form>
    </div>
  );
}
