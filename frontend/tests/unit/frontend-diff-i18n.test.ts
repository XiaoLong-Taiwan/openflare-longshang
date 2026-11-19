import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createTranslator } from 'next-intl';
import { describe, expect, it } from 'vitest';

describe.each(['zh-CN', 'zh-TW', 'en', 'ru'])(
  'frontend diff messages (%s)',
  (locale) => {
    it('resolves database and deployment labels without translation errors', () => {
      const messages = JSON.parse(
        readFileSync(resolve(process.cwd(), `messages/${locale}.json`), 'utf8'),
      );
      const errors: string[] = [];
      const t = createTranslator({
        locale,
        messages,
        onError: (error) => errors.push(error.message),
      });
      for (const key of [
        'admin.database.sections.tables',
        'admin.database.sections.cache',
        'admin.database.sections.maintenance',
        'admin.database.maintenance.optimizeDatabase',
        'nodes.install.hostNetworkTitle',
        'nodes.install.hostNetworkDescription',
      ]) {
        expect(t(key), key).not.toBe(key);
      }
      expect(errors).toEqual([]);
    });

    it('keeps the request structure tab distinct from the request count', () => {
      const messages = JSON.parse(
        readFileSync(
          resolve(process.cwd(), `messages/fragments/ops.${locale}.json`),
          'utf8',
        ),
      );
      const sections = messages.nodes.detailSections;
      expect(sections.requests).toBeTruthy();
      expect(sections.requestCount).toBeTruthy();
      expect(sections.requests).not.toBe(sections.requestCount);
    });
  },
);

it('renders every Russian maintenance message with preserved parameters', () => {
  const fragment = JSON.parse(
    readFileSync(
      resolve(process.cwd(), 'messages/fragments/admin.ru.json'),
      'utf8',
    ),
  );
  const generated = JSON.parse(
    readFileSync(resolve(process.cwd(), 'messages/ru.json'), 'utf8'),
  );
  const messages = fragment.admin.database.maintenance;
  expect(generated.admin.database.maintenance).toEqual(messages);
  const errors: string[] = [];
  const t = createTranslator({
    locale: 'ru',
    messages: generated,
    namespace: 'admin.database.maintenance',
    onError: (error) => errors.push(error.message),
  });
  for (const [key, value] of Object.entries(messages)) {
    expect(value, key).toMatch(/[А-Яа-яЁё]/);
    expect(
      t(key, { count: 12, type: 'SQLite', taskID: 'task-42' }),
      key,
    ).toMatch(/[А-Яа-яЁё]/);
  }
  expect(t('expiredTasks', { count: 12 })).toContain('12');
  expect(t('cacheItems', { count: 12 })).toContain('12');
  expect(t('databaseType', { type: 'SQLite' })).toContain('SQLite');
  expect(t('queuedDesc', { taskID: 'task-42' })).toContain('task-42');
  expect(errors).toEqual([]);
});
