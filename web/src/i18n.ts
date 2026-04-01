import { createContext } from 'preact';
import { useContext, useState, useEffect } from 'preact/hooks';
import type { ComponentChildren } from 'preact';
import { h } from 'preact';

type Translations = Record<string, unknown>;

export const SUPPORTED_LOCALES = ['ru', 'en', 'es', 'zh', 'th'] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];

export const LOCALE_LABELS: Record<Locale, string> = {
  ru: 'RU',
  en: 'EN',
  es: 'ES',
  zh: '中',
  th: 'TH',
};

const STORAGE_KEY = 'jigsaw_locale';

function getNestedValue(obj: Translations, key: string): string {
  const parts = key.split('.');
  let current: unknown = obj;
  for (const part of parts) {
    if (current === null || typeof current !== 'object') return key;
    current = (current as Record<string, unknown>)[part];
  }
  if (typeof current === 'string') return current;
  return key;
}

export function detectLocale(): Locale {
  const saved = localStorage.getItem(STORAGE_KEY) as Locale | null;
  if (saved && SUPPORTED_LOCALES.includes(saved)) return saved;

  const lang = navigator.language || 'ru';
  if (lang.startsWith('ru')) return 'ru';
  if (lang.startsWith('en')) return 'en';
  if (lang.startsWith('es')) return 'es';
  if (lang.startsWith('zh')) return 'zh';
  if (lang.startsWith('th')) return 'th';
  return 'ru';
}

export async function loadLocale(locale: string): Promise<Translations> {
  const res = await fetch(`/locales/${locale}.json`);
  if (!res.ok) throw new Error(`Failed to load locale: ${locale}`);
  return res.json() as Promise<Translations>;
}

interface I18nContextValue {
  t: (key: string) => string;
  locale: Locale;
  setLocale: (locale: Locale) => void;
}

const I18nContext = createContext<I18nContextValue>({
  t: (key: string) => key,
  locale: 'ru',
  setLocale: () => {},
});

interface I18nProviderProps {
  children: ComponentChildren;
}

export function I18nProvider({ children }: I18nProviderProps) {
  const [translations, setTranslations] = useState<Translations>({});
  const [locale, setLocaleState] = useState<Locale>(() => detectLocale());

  useEffect(() => {
    loadLocale(locale)
      .then(setTranslations)
      .catch(() => {
        if (locale !== 'ru') {
          return loadLocale('ru').then(setTranslations);
        }
      });
  }, [locale]);

  const setLocale = (next: Locale) => {
    localStorage.setItem(STORAGE_KEY, next);
    setLocaleState(next);
  };

  const t = (key: string): string => getNestedValue(translations, key);

  return h(I18nContext.Provider, { value: { t, locale, setLocale } }, children);
}

export function useT() {
  return useContext(I18nContext).t;
}

export function useLocale() {
  return useContext(I18nContext).locale;
}

export function useSetLocale() {
  return useContext(I18nContext).setLocale;
}
