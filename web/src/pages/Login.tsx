import { useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../api';
import { useT } from '../i18n';

export function Login() {
  const t = useT();
  const [, setLocation] = useLocation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const user = await api.auth.login(email, password);
      if (user.role === 'admin') {
        setLocation('/admin/catalog');
      } else {
        setLocation('/catalog');
      }
    } catch (_) {
      setError(t('login.error'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div class="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 class="mb-6 text-center text-2xl font-semibold text-gray-900">
          {t('login.title')}
        </h1>
        <form onSubmit={handleSubmit} class="space-y-4">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              {t('login.email')}
            </label>
            <input
              type="email"
              required
              value={email}
              onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
              class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">
              {t('login.password')}
            </label>
            <input
              type="password"
              required
              value={password}
              onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
              class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          {error && (
            <p class="text-sm text-red-600">{error}</p>
          )}
          <button
            type="submit"
            disabled={loading}
            class="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? t('common.loading') : t('login.submit')}
          </button>
        </form>
      </div>
    </div>
  );
}
