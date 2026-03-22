import { useEffect, useState } from 'preact/hooks';
import { useLocation, useParams } from 'wouter';
import { api } from '../../api';
import { useT } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import type { CatalogPuzzle, Reward } from '../../types';

type AnimationType = 'confetti' | 'fireworks' | 'stars';

export function CatalogEdit() {
  const t = useT();
  const [, navigate] = useLocation();
  const params = useParams<{ id: string }>();
  const id = params.id;

  // Puzzle metadata state
  const [puzzle, setPuzzle] = useState<CatalogPuzzle | null>(null);
  const [title, setTitle] = useState('');
  const [featured, setFeatured] = useState(false);
  const [sortOrder, setSortOrder] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [metaError, setMetaError] = useState('');

  // Reward state
  const [reward, setReward] = useState<Reward | null>(null);
  const [rewardLoading, setRewardLoading] = useState(true);
  const [word, setWord] = useState('');
  const [videoFile, setVideoFile] = useState<File | null>(null);
  const [animation, setAnimation] = useState<AnimationType>('confetti');
  const [rewardSaving, setRewardSaving] = useState(false);
  const [rewardError, setRewardError] = useState('');

  useEffect(() => {
    if (!id) return;

    api.admin.catalog
      .list()
      .then((list) => {
        const found = list.find((p) => p.id === id);
        if (found) {
          setPuzzle(found);
          setTitle(found.title);
          setFeatured(found.featured);
          setSortOrder(found.sort_order);
        }
      })
      .catch(() => setMetaError(t('common.error')))
      .finally(() => setLoading(false));

    api.admin.catalog
      .getReward(id)
      .then((r) => {
        if (r) {
          setReward(r);
          setWord(r.word ?? '');
          setAnimation((r.animation as AnimationType) || 'confetti');
        }
      })
      .catch(() => {
        // reward may not exist yet
      })
      .finally(() => setRewardLoading(false));
  }, [id]);

  const handleMetaSubmit = async (e: Event) => {
    e.preventDefault();
    if (!id) return;
    setMetaError('');
    setSaving(true);
    try {
      await api.admin.catalog.update(id, {
        title,
        featured,
        sort_order: sortOrder,
      });
      navigate('/admin/catalog');
    } catch (_) {
      setMetaError(t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  const handleRewardSubmit = async (e: Event) => {
    e.preventDefault();
    if (!id) return;
    setRewardError('');
    setRewardSaving(true);
    try {
      const formData = new FormData();
      if (word) formData.append('word', word);
      formData.append('animation', animation);
      if (videoFile) {
        formData.append('video', videoFile);
      }
      const updated = await api.admin.catalog.upsertReward(id, formData);
      setReward(updated);
      setVideoFile(null);
    } catch (_) {
      setRewardError(t('common.error'));
    } finally {
      setRewardSaving(false);
    }
  };

  if (loading) return <Spinner />;
  if (!puzzle) return <p class="text-red-600">{t('common.error')}</p>;

  return (
    <div class="max-w-lg space-y-8">
      {/* Section 1: Puzzle metadata */}
      <div>
        <h1 class="mb-4 text-xl font-semibold text-gray-900">{t('common.edit')}</h1>

        <div class="mb-4">
          <img
            src={`/api/media/${puzzle.image_key}`}
            class="w-32 h-32 object-cover rounded-lg border border-gray-200"
            alt=""
          />
        </div>

        <form
          onSubmit={handleMetaSubmit}
          class="space-y-4 rounded-lg border border-gray-200 bg-white p-6"
        >
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
              class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          {metaError && <p class="text-sm text-red-600">{metaError}</p>}

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
              onClick={() => navigate('/admin/catalog')}
              class="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              {t('admin.catalog.form.cancel')}
            </button>
          </div>
        </form>
      </div>

      {/* Section 2: Reward */}
      <div>
        <h2 class="mb-4 text-lg font-semibold text-gray-900">Награда</h2>

        {rewardLoading ? (
          <Spinner />
        ) : (
          <form
            onSubmit={handleRewardSubmit}
            class="space-y-4 rounded-lg border border-gray-200 bg-white p-6"
          >
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700">
                Слово-награда
              </label>
              <input
                type="text"
                value={word}
                placeholder={puzzle.locale === 'ru' ? 'например: море' : 'e.g. sea'}
                onInput={(e) => setWord((e.target as HTMLInputElement).value)}
                class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700">
                Видео (mp4, опционально)
              </label>
              {reward?.video_key && (
                <p class="mb-2 text-sm text-gray-500">
                  Видео загружено: {reward.video_key}
                </p>
              )}
              <input
                type="file"
                accept="video/mp4"
                onChange={(e) => {
                  const f = (e.target as HTMLInputElement).files?.[0];
                  setVideoFile(f ?? null);
                }}
                class="border rounded px-3 py-2 w-full text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-blue-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
              />
            </div>

            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700">Анимация</label>
              <select
                value={animation}
                onChange={(e) =>
                  setAnimation((e.target as HTMLSelectElement).value as AnimationType)
                }
                class="border rounded px-3 py-2 w-full text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                <option value="confetti">Confetti</option>
                <option value="fireworks">Fireworks</option>
                <option value="stars">Stars</option>
              </select>
            </div>

            {rewardError && <p class="text-sm text-red-600">{rewardError}</p>}

            <button
              type="submit"
              disabled={rewardSaving}
              class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {rewardSaving ? t('common.loading') : 'Сохранить награду'}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
