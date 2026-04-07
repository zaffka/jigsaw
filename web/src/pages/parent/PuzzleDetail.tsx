import { useEffect, useState, useRef } from 'preact/hooks';
import { useLocation, useRoute } from 'wouter';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import type { ParentPuzzle, PuzzleLayer } from '../../types';

const TYPE_ICONS: Record<PuzzleLayer['type'], string> = {
  word: '📝',
  audio: '🎵',
  video: '🎬',
};

const TYPE_LABELS: Record<PuzzleLayer['type'], string> = {
  word: 'Слово',
  audio: 'Аудио',
  video: 'Видео',
};

export function PuzzleDetail() {
  const [, setLocation] = useLocation();
  const [, params] = useRoute('/parent/puzzles/:id');
  const id = params?.id ?? '';

  const [puzzle, setPuzzle] = useState<ParentPuzzle | null>(null);
  const [layers, setLayers] = useState<PuzzleLayer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Submit state
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'loading' | 'success' | 'conflict' | 'error'>('idle');

  // Edit title state
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState('');
  const [savingTitle, setSavingTitle] = useState(false);

  // Add layer state
  const [addLayerType, setAddLayerType] = useState<PuzzleLayer['type'] | null>(null);
  const [wordText, setWordText] = useState('');
  const [addingLayer, setAddingLayer] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!id) return;
    Promise.all([api.parent.getPuzzle(id), api.parent.listLayers(id)])
      .then(([p, l]) => {
        setPuzzle(p);
        setLayers(l);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [id]);

  const handleSaveTitle = async () => {
    if (!puzzle || !titleDraft.trim()) return;
    setSavingTitle(true);
    try {
      const updated = await api.parent.updatePuzzle(id, { title: titleDraft.trim() });
      setPuzzle(updated);
      setEditingTitle(false);
    } catch (e: unknown) {
      alert((e as Error).message);
    } finally {
      setSavingTitle(false);
    }
  };

  const handleDeletePuzzle = async () => {
    if (!window.confirm('Удалить пазл? Это действие необратимо.')) return;
    try {
      await api.parent.deletePuzzle(id);
      setLocation('/parent/puzzles');
    } catch (e: unknown) {
      alert((e as Error).message);
    }
  };

  const handleSubmit = async () => {
    if (!puzzle) return;
    setSubmitStatus('loading');
    try {
      await api.parent.submit(puzzle.id);
      setSubmitStatus('success');
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '';
      if (msg.includes('409') || msg.toLowerCase().includes('already')) {
        setSubmitStatus('conflict');
      } else {
        setSubmitStatus('error');
      }
    }
  };

  const handleDeleteLayer = async (layerId: string) => {
    if (!window.confirm('Удалить этот слой?')) return;
    try {
      await api.parent.deleteLayer(id, layerId);
      setLayers((prev) => prev.filter((l) => l.id !== layerId));
    } catch (e: unknown) {
      alert((e as Error).message);
    }
  };

  const handleAddWordLayer = async () => {
    if (!wordText.trim()) return;
    setAddingLayer(true);
    try {
      const form = new FormData();
      form.append('type', 'word');
      form.append('text', wordText.trim());
      const layer = await api.parent.createLayer(id, form);
      setLayers((prev) => [...prev, layer]);
      setWordText('');
      setAddLayerType(null);
    } catch (e: unknown) {
      alert((e as Error).message);
    } finally {
      setAddingLayer(false);
    }
  };

  const handleAddFileLayer = async (type: 'audio' | 'video', file: File) => {
    setAddingLayer(true);
    try {
      const form = new FormData();
      form.append('type', type);
      form.append('file', file);
      const layer = await api.parent.createLayer(id, form);
      setLayers((prev) => [...prev, layer]);
      setAddLayerType(null);
    } catch (e: unknown) {
      alert((e as Error).message);
    } finally {
      setAddingLayer(false);
    }
  };

  if (loading) return <Spinner />;
  if (error) return <div class="rounded-md bg-red-50 p-4 text-red-700">{error}</div>;
  if (!puzzle) return null;

  const sortedLayers = [...layers].sort((a, b) => a.sort_order - b.sort_order);

  return (
    <div class="max-w-2xl">
      {/* Back */}
      <button
        onClick={() => setLocation('/parent/puzzles')}
        class="mb-4 text-sm text-blue-600 hover:underline"
      >
        ← Назад к списку
      </button>

      {/* Title */}
      <div class="mb-6">
        {editingTitle ? (
          <div class="flex items-center gap-2">
            <input
              class="rounded-md border border-gray-300 px-3 py-1.5 text-xl font-semibold focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={titleDraft}
              onInput={(e) => setTitleDraft((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSaveTitle();
                if (e.key === 'Escape') setEditingTitle(false);
              }}
              autoFocus
            />
            <button
              onClick={handleSaveTitle}
              disabled={savingTitle}
              class="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              Сохранить
            </button>
            <button
              onClick={() => setEditingTitle(false)}
              class="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50"
            >
              Отмена
            </button>
          </div>
        ) : (
          <div class="flex items-center gap-2">
            <h1 class="text-2xl font-semibold text-gray-900">{puzzle.title}</h1>
            <button
              onClick={() => {
                setTitleDraft(puzzle.title);
                setEditingTitle(true);
              }}
              class="text-sm text-gray-400 hover:text-blue-600"
              title="Редактировать название"
            >
              ✏️
            </button>
          </div>
        )}

        {/* Meta */}
        <div class="mt-2 flex flex-wrap gap-2">
          <span
            class={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
              puzzle.status === 'ready'
                ? 'bg-green-100 text-green-800'
                : puzzle.status === 'processing'
                ? 'bg-yellow-100 text-yellow-800'
                : 'bg-red-100 text-red-800'
            }`}
          >
            {puzzle.status === 'ready'
              ? 'Готов'
              : puzzle.status === 'processing'
              ? 'Обработка'
              : 'Ошибка'}
          </span>
          {puzzle.difficulty && (
            <span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
              {puzzle.difficulty === 'easy'
                ? 'Лёгкий'
                : puzzle.difficulty === 'medium'
                ? 'Средний'
                : 'Сложный'}
            </span>
          )}
        </div>
      </div>

      {/* Layers */}
      <div class="mb-8">
        <h2 class="mb-3 text-lg font-semibold text-gray-800">Слои</h2>

        {sortedLayers.length === 0 ? (
          <p class="mb-4 text-sm text-gray-500">Слоёв пока нет</p>
        ) : (
          <div class="mb-4 space-y-2">
            {sortedLayers.map((layer) => (
              <div
                key={layer.id}
                class="flex items-center justify-between rounded-md border border-gray-200 bg-white px-4 py-3"
              >
                <div class="flex items-center gap-3">
                  <span class="text-lg" title={TYPE_LABELS[layer.type]}>
                    {TYPE_ICONS[layer.type]}
                  </span>
                  <div>
                    <span class="text-xs font-medium text-gray-500">{TYPE_LABELS[layer.type]}</span>
                    {layer.text && (
                      <p class="text-sm text-gray-800">{layer.text}</p>
                    )}
                    {layer.audio_key && (
                      <p class="text-xs text-gray-500 truncate max-w-xs">{layer.audio_key}</p>
                    )}
                    {layer.video_key && (
                      <p class="text-xs text-gray-500 truncate max-w-xs">{layer.video_key}</p>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => handleDeleteLayer(layer.id)}
                  class="text-gray-400 hover:text-red-600"
                  title="Удалить слой"
                >
                  🗑
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Add layer buttons */}
        {addLayerType === null && (
          <div class="flex flex-wrap gap-2">
            <button
              onClick={() => setAddLayerType('word')}
              class="rounded-md border border-dashed border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:border-blue-400 hover:text-blue-600"
            >
              📝 + Слово
            </button>
            <button
              onClick={() => setAddLayerType('audio')}
              class="rounded-md border border-dashed border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:border-blue-400 hover:text-blue-600"
            >
              🎵 + Аудио
            </button>
            <button
              onClick={() => setAddLayerType('video')}
              class="rounded-md border border-dashed border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:border-blue-400 hover:text-blue-600"
            >
              🎬 + Видео
            </button>
          </div>
        )}

        {/* Add word form */}
        {addLayerType === 'word' && (
          <div class="mt-3 rounded-md border border-gray-200 bg-gray-50 p-4">
            <label class="mb-1 block text-sm font-medium text-gray-700">Текст слова</label>
            <input
              class="mb-3 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Введите слово или фразу"
              value={wordText}
              onInput={(e) => setWordText((e.target as HTMLInputElement).value)}
              autoFocus
            />
            <div class="flex gap-2">
              <button
                onClick={handleAddWordLayer}
                disabled={addingLayer || !wordText.trim()}
                class="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                Добавить
              </button>
              <button
                onClick={() => { setAddLayerType(null); setWordText(''); }}
                class="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
              >
                Отмена
              </button>
            </div>
          </div>
        )}

        {/* Add audio form */}
        {addLayerType === 'audio' && (
          <div class="mt-3 rounded-md border border-gray-200 bg-gray-50 p-4">
            <label class="mb-1 block text-sm font-medium text-gray-700">Аудиофайл (.mp3, .wav)</label>
            <input
              ref={fileInputRef}
              type="file"
              accept=".mp3,.wav,audio/*"
              class="mb-3 block w-full text-sm text-gray-600"
              onChange={(e) => {
                const file = (e.target as HTMLInputElement).files?.[0];
                if (file) handleAddFileLayer('audio', file);
              }}
            />
            {addingLayer && <p class="text-sm text-gray-500">Загрузка...</p>}
            <button
              onClick={() => setAddLayerType(null)}
              class="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
            >
              Отмена
            </button>
          </div>
        )}

        {/* Add video form */}
        {addLayerType === 'video' && (
          <div class="mt-3 rounded-md border border-gray-200 bg-gray-50 p-4">
            <label class="mb-1 block text-sm font-medium text-gray-700">Видеофайл (.mp4)</label>
            <input
              ref={fileInputRef}
              type="file"
              accept=".mp4,video/*"
              class="mb-3 block w-full text-sm text-gray-600"
              onChange={(e) => {
                const file = (e.target as HTMLInputElement).files?.[0];
                if (file) handleAddFileLayer('video', file);
              }}
            />
            {addingLayer && <p class="text-sm text-gray-500">Загрузка...</p>}
            <button
              onClick={() => setAddLayerType(null)}
              class="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
            >
              Отмена
            </button>
          </div>
        )}
      </div>

      {/* Publication section */}
      <div class="mt-8 border-t pt-6">
        <h2 class="text-lg font-semibold text-gray-800 mb-3">Публикация</h2>
        <p class="text-sm text-gray-500 mb-4">
          Подайте пазл на проверку администратором для добавления в публичный каталог.
        </p>
        {submitStatus === 'idle' && (
          <button
            onClick={handleSubmit}
            disabled={puzzle.status !== 'ready'}
            class="rounded-xl bg-indigo-500 px-5 py-2.5 text-sm font-semibold text-white shadow hover:bg-indigo-600 active:scale-95 transition-all disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Подать на публикацию
          </button>
        )}
        {submitStatus === 'loading' && <p class="text-sm text-gray-500">Отправка...</p>}
        {submitStatus === 'success' && (
          <p class="text-sm text-green-600 font-medium">✓ Заявка отправлена! Ожидайте проверки.</p>
        )}
        {submitStatus === 'conflict' && (
          <p class="text-sm text-yellow-600 font-medium">Заявка уже отправлена или пазл уже опубликован.</p>
        )}
        {submitStatus === 'error' && (
          <p class="text-sm text-red-600">Не удалось отправить заявку. Попробуйте позже.</p>
        )}
      </div>

      {/* Delete puzzle */}
      <div class="border-t border-gray-200 pt-6">
        <button
          onClick={handleDeletePuzzle}
          class="rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50"
        >
          Удалить пазл
        </button>
      </div>
    </div>
  );
}
