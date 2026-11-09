import { execFileSync } from 'node:child_process';
import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const fragments = resolve(root, 'frontend/messages/fragments');
const reference = process.argv[2] ?? '217a7166';
const revision = execFileSync('git', ['rev-parse', reference], {
  cwd: root,
  encoding: 'utf8',
}).trim();
const gaps = [];
let translated = 0;
const argumentsOf = (text) =>
  [
    ...new Set(
      [...text.matchAll(/\{(\w+)(?:\s*[,}])/g)].map((match) => match[1]),
    ),
  ]
    .sort()
    .join(',');

for (const name of readdirSync(fragments)
  .filter((name) => name.endsWith('.en.json'))
  .sort()) {
  const target = name.replace('.en.json', '.ru.json');
  const source = JSON.parse(readFileSync(resolve(fragments, name), 'utf8'));
  const upstream = JSON.parse(
    execFileSync(
      'git',
      ['show', `${revision}:frontend/messages/fragments/${target}`],
      { cwd: root, encoding: 'utf8' },
    ),
  );
  function align(english, russian, prefix = '') {
    return Object.fromEntries(
      Object.entries(english).map(([key, value]) => {
        const path = prefix ? `${prefix}.${key}` : key;
        if (value && typeof value === 'object' && !Array.isArray(value))
          return [key, align(value, russian?.[key], path)];
        const candidate = russian?.[key];
        if (
          typeof candidate === 'string' &&
          argumentsOf(candidate) === argumentsOf(value)
        ) {
          translated++;
          return [key, candidate];
        }
        gaps.push({
          file: target,
          key: path,
          reason:
            candidate === undefined
              ? 'missing-upstream-key'
              : 'argument-mismatch',
          fallback: value,
        });
        return [key, value];
      }),
    );
  }
  writeFileSync(
    resolve(fragments, target),
    `${JSON.stringify(align(source, upstream), null, 2)}\n`,
  );
}
writeFileSync(
  resolve(root, 'docs/design/frontend-russian-fallbacks.json'),
  `${JSON.stringify({ upstream: revision, translated, fallbackCount: gaps.length, gaps }, null, 2)}\n`,
);
console.log(
  `Russian import: ${translated} upstream translations, ${gaps.length} explicit English fallbacks`,
);
