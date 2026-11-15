import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createTranslator } from 'next-intl';
import { describe, expect, it } from 'vitest';

describe.each(['zh-CN', 'zh-TW', 'en', 'ru'])('frontend diff messages (%s)', (locale) => {
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
      readFileSync(resolve(process.cwd(), `messages/fragments/ops.${locale}.json`), 'utf8'),
    );
    const sections = messages.nodes.detailSections;
    expect(sections.requests).toBeTruthy();
    expect(sections.requestCount).toBeTruthy();
    expect(sections.requests).not.toBe(sections.requestCount);
  });
});
