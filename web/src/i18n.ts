import { createContext } from 'preact';
import { useContext, useState, useEffect } from 'preact/hooks';
import type { ComponentChildren } from 'preact';
import { h } from 'preact';

type Translations = Record<string, unknown>;

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

export function detectLocale(): string {
  const lang = navigator.language || 'ru';
  if (lang.startsWith('ru')) return 'ru';
  if (lang.startsWith('en')) return 'en';
  return 'ru';
}

export async function loadLocale(locale: string): Promise<Translations> {
  const res = await fetch(`/locales/${locale}.json`);
  if (!res.ok) throw new Error(`Failed to load locale: ${locale}`);
  return res.json() as Promise<Translations>;
}

interface I18nContextValue {
  t: (key: string) => string;
  locale: string;
}

const I18nContext = createContext<I18nContextValue>({
  t: (key: string) => key,
  locale: 'ru',
});

interface I18nProviderProps {
  children: ComponentChildren;
}

export function I18nProvider({ children }: I18nProviderProps) {
  const [translations, setTranslations] = useState<Translations>({});
  const [locale, setLocale] = useState<string>('ru');

  useEffect(() => {
    const detected = detectLocale();
    setLocale(detected);
    loadLocale(detected)
      .then(setTranslations)
      .catch(() => {
        if (detected !== 'ru') {
          return loadLocale('ru').then(setTranslations);
        }
      });
  }, []);

  const t = (key: string): string => getNestedValue(translations, key);

  return h(I18nContext.Provider, { value: { t, locale } }, children);
}

export function useT() {
  return useContext(I18nContext).t;
}

export function useLocale() {
  return useContext(I18nContext).locale;
}
