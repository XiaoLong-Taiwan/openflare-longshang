import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

function flatten(obj, prefix = '') {
  const keys = new Set();
  for (const [key, value] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      for (const child of flatten(value, path)) keys.add(child);
    } else {
      keys.add(path);
    }
  }
  return keys;
}

const locales = ['zh-CN', 'zh-TW', 'en'];
const localeKeys = new Map(
  locales.map((locale) => {
    const messages = JSON.parse(
      readFileSync(resolve(root, `messages/${locale}.json`), 'utf8'),
    );
    return [locale, flatten(messages)];
  }),
);
const referenceKeys = localeKeys.get('zh-CN');
const mismatches = [];

for (const locale of locales) {
  if (locale === 'zh-CN') continue;
  const keys = localeKeys.get(locale);
  const missing = [...referenceKeys].filter((key) => !keys.has(key)).sort();
  const extra = [...keys].filter((key) => !referenceKeys.has(key)).sort();
  if (missing.length || extra.length) {
    mismatches.push({ locale, missing, extra });
  }
}

if (mismatches.length) {
  console.error('i18n key mismatch');
  for (const mismatch of mismatches) {
    if (mismatch.missing.length) {
      console.error(`missing in ${mismatch.locale}:`, mismatch.missing);
    }
    if (mismatch.extra.length) {
      console.error(`only in ${mismatch.locale}:`, mismatch.extra);
    }
  }
  process.exit(1);
}

console.log(
  `i18n keys OK (${referenceKeys.size} keys across ${locales.length} locales)`,
);
