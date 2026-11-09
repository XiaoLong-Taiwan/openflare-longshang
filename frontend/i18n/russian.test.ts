import { afterEach, describe, expect, it } from 'vitest';
import { createTranslator } from 'next-intl';
import english from '@/messages/en.json';
import russian from '@/messages/ru.json';
import {
  defaultLocale,
  isAppLocale,
  locales,
  normalizeLocale,
  pickLocaleFromAcceptLanguage,
} from './config';
import {
  detectBrowserLocale,
  getClientLocalePreference,
  resolveClientLocale,
} from './client';
import { formatNumber } from './format';

function flatten(value: object, prefix = ''): Record<string, string> {
  return Object.fromEntries(
    Object.entries(value).flatMap(([key, item]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      return typeof item === 'string'
        ? [[path, item]]
        : Object.entries(flatten(item, path));
    }),
  );
}

afterEach(() => {
  document.cookie = 'NEXT_LOCALE=; Max-Age=0; Path=/';
  document.documentElement.lang = 'zh-TW';
});

describe('Russian integration with Traditional Chinese customization', () => {
  it('retains Traditional Chinese default and all existing locales', () => {
    expect(defaultLocale).toBe('zh-TW');
    expect(locales).toEqual(['zh-CN', 'zh-TW', 'en', 'ru']);
    expect(isAppLocale('ru')).toBe(true);
    expect(normalizeLocale('zh-Hant')).toBe('zh-TW');
    expect(normalizeLocale('ru_RU')).toBe('ru');
    expect(normalizeLocale('ru-BY')).toBe('ru');
    expect(pickLocaleFromAcceptLanguage('de,ru-RU;q=0.9,en;q=0.7')).toBe('ru');
    expect(pickLocaleFromAcceptLanguage('ru;q=0,en;q=0.5')).toBe('en');
    expect(pickLocaleFromAcceptLanguage('de,fr')).toBe('zh-TW');
  });

  it('resolves Russian browser preference and preserves cookie precedence', () => {
    const descriptor = Object.getOwnPropertyDescriptor(navigator, 'languages');
    Object.defineProperty(navigator, 'languages', {
      configurable: true,
      value: ['ru-RU', 'en'],
    });
    try {
      expect(detectBrowserLocale()).toBe('ru');
      document.cookie = 'NEXT_LOCALE=zh-TW; Path=/';
      expect(resolveClientLocale()).toBe('zh-TW');
      document.cookie = 'NEXT_LOCALE=ru; Path=/';
      expect(getClientLocalePreference()).toBe('ru');
      document.cookie = 'NEXT_LOCALE=%E0%A4%A; Path=/';
      expect(resolveClientLocale()).toBe('ru');
    } finally {
      if (descriptor) Object.defineProperty(navigator, 'languages', descriptor);
      else Reflect.deleteProperty(navigator, 'languages');
    }
  });

  it('formats numbers using the rendered Russian locale', () => {
    document.documentElement.lang = 'ru';
    expect(formatNumber(1234.5)).toBe(
      new Intl.NumberFormat('ru-RU').format(1234.5),
    );
  });

  it('preserves catalog keys and named ICU arguments', () => {
    const en = flatten(english);
    const ru = flatten(russian);
    expect(Object.keys(ru).sort()).toEqual(Object.keys(en).sort());
    const args = (text: string) =>
      [
        ...new Set(
          [...text.matchAll(/\{(\w+)(?:\s*[,}])/g)].map((match) => match[1]),
        ),
      ].sort();
    for (const key of Object.keys(en))
      expect(args(ru[key]), key).toEqual(args(en[key]));
    const errors: string[] = [];
    const t = createTranslator({
      locale: 'ru',
      messages: russian,
      onError: (error) => errors.push(error.message),
    });
    expect(t('common.save')).toBe('Сохранить');
    expect(errors).toEqual([]);
  });
});
