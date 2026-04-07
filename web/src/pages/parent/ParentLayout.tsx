import type { ComponentChildren } from 'preact';
import { useEffect, useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import type { User } from '../../types';

interface ParentLayoutProps {
  children: ComponentChildren;
}

export function ParentLayout({ children }: ParentLayoutProps) {
  const [, setLocation] = useLocation();
  const [user, setUser] = useState<User | null>(null);
  const [checking, setChecking] = useState(true);
  const [currentPath] = useLocation();

  useEffect(() => {
    api.auth
      .me()
      .then((u) => {
        if (u.role !== 'parent' && u.role !== 'admin') {
          setLocation('/login');
        } else {
          setUser(u);
        }
      })
      .catch(() => setLocation('/login'))
      .finally(() => setChecking(false));
  }, []);

  const handleLogout = async () => {
    try {
      await api.auth.logout();
    } catch (_) {
      // ignore
    }
    setLocation('/login');
  };

  if (checking) return <Spinner />;
  if (!user) return null;

  const navItems = [
    { path: '/parent/puzzles', label: 'Пазлы' },
    { path: '/parent/children', label: 'Дети' },
  ];

  return (
    <div class="flex min-h-screen bg-gray-50">
      {/* Sidebar */}
      <aside class="w-56 border-r border-gray-200 bg-white">
        <div class="border-b border-gray-200 px-4 py-4">
          <a href="/catalog" class="text-lg font-semibold text-blue-600">
            Мой кабинет
          </a>
          <p class="mt-1 text-xs text-gray-500">{user.email}</p>
        </div>
        <nav class="p-2">
          {navItems.map((item) => (
            <a
              key={item.path}
              href={item.path}
              class={`block rounded-md px-3 py-2 text-sm font-medium ${
                currentPath === item.path || currentPath.startsWith(item.path + '/')
                  ? 'bg-blue-50 text-blue-700'
                  : 'text-gray-700 hover:bg-gray-100'
              }`}
            >
              {item.label}
            </a>
          ))}
        </nav>
        <div class="absolute bottom-0 w-56 border-t border-gray-200 p-2">
          <button
            onClick={handleLogout}
            class="w-full rounded-md px-3 py-2 text-left text-sm text-gray-600 hover:bg-gray-100"
          >
            Выйти
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main class="flex-1 p-6">{children}</main>
    </div>
  );
}
