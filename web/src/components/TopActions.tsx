import { useState } from 'preact/hooks';
import { useLocation } from 'wouter';
import { useLocale, useSetLocale, SUPPORTED_LOCALES, LOCALE_LABELS, type Locale } from '../i18n';

export function TopActions() {
  const [, navigate] = useLocation();
  const locale = useLocale();
  const setLocale = useSetLocale();
  const [langOpen, setLangOpen] = useState(false);

  return (
    <div class="fixed top-4 right-4 z-10 flex gap-2 items-center">
      {/* Language switcher */}
      <div class="relative">
        <button
          onClick={() => setLangOpen((v) => !v)}
          title="Язык / Language"
          class="flex h-10 w-10 items-center justify-center rounded-full bg-white shadow-md border border-gray-200 text-gray-600 hover:bg-gray-50 active:scale-95 transition-all text-xs font-bold tracking-wide"
        >
          {LOCALE_LABELS[locale]}
        </button>
        {langOpen && (
          <div class="absolute right-0 top-12 flex flex-col rounded-xl bg-white shadow-lg border border-gray-200 overflow-hidden min-w-[56px]">
            {SUPPORTED_LOCALES.map((l) => (
              <button
                key={l}
                onClick={() => { setLocale(l as Locale); setLangOpen(false); }}
                class={`px-3 py-2 text-sm font-medium text-left hover:bg-gray-50 transition-colors ${l === locale ? 'text-blue-600 bg-blue-50' : 'text-gray-700'}`}
              >
                {LOCALE_LABELS[l as Locale]}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* About */}
      <button
        onClick={() => navigate('/about')}
        title="О проекте"
        class="flex h-10 w-10 items-center justify-center rounded-full bg-white shadow-md border border-gray-200 text-gray-600 hover:bg-gray-50 active:scale-95 transition-all text-lg font-semibold"
      >
        ?
      </button>

      {/* Login / Settings */}
      <button
        onClick={() => navigate('/login')}
        title="Войти"
        class="flex h-10 w-10 items-center justify-center rounded-full bg-white shadow-md border border-gray-200 text-gray-600 hover:bg-gray-50 active:scale-95 transition-all"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
      </button>
    </div>
  );
}
