import { useEffect, useState } from 'preact/hooks';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import type { Submission } from '../../types';

export function NotificationList() {
  const [items, setItems] = useState<Submission[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.parent.listNotifications()
      .then(setItems)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div class="flex justify-center py-12"><Spinner /></div>;

  return (
    <div class="max-w-2xl mx-auto px-4 py-8">
      <h1 class="text-2xl font-semibold text-gray-900 mb-6">Уведомления</h1>
      {items.length === 0 ? (
        <p class="text-gray-500">Новых уведомлений нет</p>
      ) : (
        <div class="space-y-4">
          {items.map((item) => (
            <div key={item.id} class={`rounded-xl border p-4 ${
              item.status === 'approved' ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'
            }`}>
              <div class="flex items-start justify-between gap-4">
                <div>
                  <p class="font-medium text-gray-800">{item.puzzle_title}</p>
                  {item.status === 'approved' ? (
                    <p class="text-sm text-green-700 mt-1">✓ Пазл одобрен и добавлен в публичный каталог</p>
                  ) : (
                    <>
                      <p class="text-sm text-red-700 mt-1">✗ Пазл отклонён</p>
                      {item.admin_comment && (
                        <p class="text-sm text-gray-600 mt-1 italic">«{item.admin_comment}»</p>
                      )}
                    </>
                  )}
                </div>
                <span class={`text-xs font-medium rounded-full px-2 py-1 ${
                  item.status === 'approved'
                    ? 'bg-green-100 text-green-700'
                    : 'bg-red-100 text-red-700'
                }`}>
                  {item.status === 'approved' ? 'Одобрен' : 'Отклонён'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
