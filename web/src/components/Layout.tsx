import type { ComponentChildren } from 'preact';
import { useT } from '../i18n';
import { api } from '../api';
import { useLocation } from 'wouter';

interface LayoutProps {
  children: ComponentChildren;
}

export function Layout({ children }: LayoutProps) {
  const t = useT();
  const [, setLocation] = useLocation();

  const handleLogout = async () => {
    try {
      await api.auth.logout();
    } catch (_) {
      // ignore
    }
    setLocation('/login');
  };

  return (
    <div class="min-h-screen bg-gray-50">
      <nav class="border-b border-gray-200 bg-white px-4 py-3">
        <div class="mx-auto flex max-w-6xl items-center justify-between">
          <a href="/catalog" class="text-lg font-semibold text-blue-600">
            Jigsaw
          </a>
          <button
            onClick={handleLogout}
            class="text-sm text-gray-600 hover:text-gray-900"
          >
            {t('common.logout')}
          </button>
        </div>
      </nav>
      <main class="mx-auto max-w-6xl px-4 py-6">{children}</main>
    </div>
  );
}
