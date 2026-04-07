import { useEffect, useState } from 'preact/hooks';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import type { ModerationItem, ModerationLayer } from '../../types';

function layerIcon(type: ModerationLayer['type']): string {
  if (type === 'word') return '📝';
  if (type === 'audio') return '🔊';
  return '🎬';
}

function layerLabel(layer: ModerationLayer): string {
  if (layer.type === 'word') return layer.text ?? '—';
  if (layer.type === 'audio') return layer.audio_key ?? 'audio file';
  return layer.video_key ?? 'video file';
}

interface RejectPanelProps {
  onConfirm: (comment: string) => void;
  onCancel: () => void;
}

function RejectPanel({ onConfirm, onCancel }: RejectPanelProps) {
  const [comment, setComment] = useState('');

  return (
    <div class="mt-2 space-y-2">
      <textarea
        class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-red-400 focus:outline-none focus:ring-1 focus:ring-red-400"
        rows={3}
        placeholder="Причина отклонения (необязательно)"
        value={comment}
        onInput={(e) => setComment((e.target as HTMLTextAreaElement).value)}
      />
      <div class="flex gap-2">
        <button
          onClick={() => onConfirm(comment)}
          class="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700"
        >
          Подтвердить отклонение
        </button>
        <button
          onClick={onCancel}
          class="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          Отмена
        </button>
      </div>
    </div>
  );
}

interface ModerationCardProps {
  item: ModerationItem;
  isRejecting: boolean;
  onApprove: () => void;
  onRejectStart: () => void;
  onRejectConfirm: (comment: string) => void;
  onRejectCancel: () => void;
}

function ModerationCard({
  item,
  isRejecting,
  onApprove,
  onRejectStart,
  onRejectConfirm,
  onRejectCancel,
}: ModerationCardProps) {
  const date = new Date(item.created_at).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });

  return (
    <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
      <div class="grid grid-cols-1 gap-4 p-4 md:grid-cols-3">
        {/* Left: puzzle preview */}
        <div class="flex items-start gap-3">
          <img
            src={`/api/media/${item.image_key}`}
            alt={item.puzzle_title}
            class="h-20 w-20 flex-shrink-0 rounded-md object-cover"
          />
          <div class="min-w-0">
            <p class="truncate text-sm font-medium text-gray-900">{item.puzzle_title || '—'}</p>
            <p class="mt-1 text-xs text-gray-500">Подано: {date}</p>
            <span class="mt-1 inline-block rounded bg-yellow-100 px-1.5 py-0.5 text-xs font-medium text-yellow-800">
              На проверке
            </span>
          </div>
        </div>

        {/* Middle: layers */}
        <div>
          <p class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-500">Слои</p>
          {item.layers.length === 0 ? (
            <p class="text-sm text-gray-400">Нет слоёв</p>
          ) : (
            <ul class="space-y-1">
              {item.layers.map((layer) => (
                <li key={layer.id} class="flex items-center gap-1.5 text-sm text-gray-700">
                  <span>{layerIcon(layer.type)}</span>
                  <span class="truncate">{layerLabel(layer)}</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Right: actions */}
        <div class="flex flex-col justify-start">
          {!isRejecting ? (
            <div class="flex flex-wrap gap-2">
              <button
                onClick={onApprove}
                class="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"
              >
                Одобрить
              </button>
              <button
                onClick={onRejectStart}
                class="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
              >
                Отклонить
              </button>
            </div>
          ) : (
            <RejectPanel onConfirm={onRejectConfirm} onCancel={onRejectCancel} />
          )}
        </div>
      </div>
    </div>
  );
}

export function ModerationList() {
  const [items, setItems] = useState<ModerationItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [rejectingId, setRejectingId] = useState<string | null>(null);

  useEffect(() => {
    api.admin.moderation
      .list()
      .then(setItems)
      .catch(() => setError('Не удалось загрузить очередь модерации'))
      .finally(() => setLoading(false));
  }, []);

  const handleApprove = async (id: string) => {
    try {
      await api.admin.moderation.approve(id);
      setItems((prev) => prev.filter((item) => item.id !== id));
    } catch (_) {
      setError('Не удалось одобрить заявку');
    }
  };

  const handleReject = async (id: string, comment: string) => {
    try {
      await api.admin.moderation.reject(id, comment);
      setItems((prev) => prev.filter((item) => item.id !== id));
      setRejectingId(null);
    } catch (_) {
      setError('Не удалось отклонить заявку');
    }
  };

  if (loading) return <Spinner />;

  return (
    <div>
      <div class="mb-6">
        <h1 class="text-xl font-semibold text-gray-900">Модерация</h1>
        <p class="mt-1 text-sm text-gray-500">Проверка пазлов, отправленных родителями</p>
      </div>

      {error && <p class="mb-4 text-sm text-red-600">{error}</p>}

      {items.length === 0 ? (
        <div class="rounded-lg border border-gray-200 bg-white p-10 text-center">
          <p class="text-gray-500">Очередь на модерацию пуста ✅</p>
        </div>
      ) : (
        <div class="space-y-4">
          {items.map((item) => (
            <ModerationCard
              key={item.id}
              item={item}
              isRejecting={rejectingId === item.id}
              onApprove={() => handleApprove(item.id)}
              onRejectStart={() => setRejectingId(item.id)}
              onRejectConfirm={(comment) => handleReject(item.id, comment)}
              onRejectCancel={() => setRejectingId(null)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
