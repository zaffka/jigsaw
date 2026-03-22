import { useEffect, useState } from 'preact/hooks';
import { api } from '../../api';
import { useT } from '../../i18n';
import { Spinner } from '../../components/Spinner';
import type { User } from '../../types';

export function UserList() {
  const t = useT();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api.admin.users
      .list()
      .then(setUsers)
      .catch(() => setError(t('common.error')))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner />;

  return (
    <div>
      <h1 class="mb-6 text-xl font-semibold text-gray-900">{t('admin.users.title')}</h1>

      {error && <p class="mb-4 text-sm text-red-600">{error}</p>}

      {users.length === 0 ? (
        <p class="text-gray-500">Пользователей пока нет.</p>
      ) : (
        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white">
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  {t('admin.users.email')}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  {t('admin.users.role')}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                  {t('admin.users.locale')}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              {users.map((user) => (
                <tr key={user.id} class="hover:bg-gray-50">
                  <td class="px-4 py-3 text-sm text-gray-900">{user.email}</td>
                  <td class="px-4 py-3 text-sm text-gray-900">{user.role}</td>
                  <td class="px-4 py-3 text-sm text-gray-900">{user.locale}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
