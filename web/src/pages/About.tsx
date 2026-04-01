import { useLocation } from 'wouter';
import { useT } from '../i18n';
import { TopActions } from '../components/TopActions';

export function About() {
  const t = useT();
  const [, navigate] = useLocation();

  return (
    <div class="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-6 py-12">
      <TopActions />
      <div class="w-full max-w-md rounded-2xl bg-white p-8 shadow-sm border border-gray-100 text-center">
        <div class="mb-4 text-5xl">🧩</div>
        <h1 class="mb-3 text-2xl font-semibold text-gray-900">{t('about.title')}</h1>
        <p class="mb-6 text-gray-600 leading-relaxed">{t('about.description')}</p>
        <button
          onClick={() => navigate('/catalog')}
          class="rounded-xl bg-blue-600 px-6 py-2 text-sm font-medium text-white hover:bg-blue-700 active:scale-95 transition-all"
        >
          {t('about.toCatalog')}
        </button>
      </div>
    </div>
  );
}
