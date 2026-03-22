import { useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { useT } from '../../i18n';

export function CatalogCreate() {
  const t = useT();
  const [, setLocation] = useLocation();
  const [titleRu, setTitleRu] = useState('');
  const [titleEn, setTitleEn] = useState('');
  const [config, setConfig] = useState('');
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
      formData.append('title_ru', titleRu);
      formData.append('title_en', titleEn);
      if (config.trim()) {
        formData.append('config', config.trim());
      }

      await api.admin.catalog.create(formData);
      setLocation('/admin/catalog');
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
              if (f && f.size <= 10 * 1024 * 1024) {
                setFile(f);
                setError('');
              } else if (f) {
                setError('Файл слишком большой (максимум 10MB)');
              }
            }}
            class="w-full text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-blue-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
          />
        </div>

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

        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            Config (JSON, необязательно)
          </label>
          <textarea
            value={config}
            onInput={(e) => setConfig((e.target as HTMLTextAreaElement).value)}
            rows={4}
            placeholder="{}"
            class="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
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
