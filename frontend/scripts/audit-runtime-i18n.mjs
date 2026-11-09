import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import ts from 'typescript';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const findings = [];
function inspect(file) {
  const source = readFileSync(file, 'utf8');
  const ast = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true);
  function visit(node) {
    const literal =
      ts.isStringLiteral(node) ||
      ts.isNoSubstitutionTemplateLiteral(node) ||
      ts.isJsxText(node) ||
      ts.isTemplateHead(node) ||
      ts.isTemplateMiddle(node) ||
      ts.isTemplateTail(node);
    if (literal && /\p{Script=Han}/u.test(node.text)) {
      findings.push({
        file: relative(root, file).replaceAll('\\', '/'),
        line: ast.getLineAndCharacterOfPosition(node.getStart(ast)).line + 1,
        kind: 'literal-requires-review',
      });
    }
    if (
      (ts.isCallExpression(node) || ts.isNewExpression(node)) &&
      /(?:Intl\.(?:NumberFormat|DateTimeFormat|DisplayNames)|\.toLocale(?:String|DateString|TimeString)|\.localeCompare)$/.test(
        node.expression.getText(ast),
      ) &&
      node.arguments?.some(
        (arg) => ts.isStringLiteral(arg) && /^zh-(CN|TW)$/.test(arg.text),
      )
    ) {
      findings.push({
        file: relative(root, file).replaceAll('\\', '/'),
        line: ast.getLineAndCharacterOfPosition(node.getStart(ast)).line + 1,
        kind: 'fixed-formatting-locale',
      });
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
}
function walk(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.name.startsWith('.') || entry.name.startsWith('node_modules'))
      continue;
    const file = resolve(directory, entry.name);
    if (entry.isDirectory()) walk(file);
    else if (/\.tsx?$/.test(file) && !/\.(test|spec)\.tsx?$/.test(file))
      inspect(file);
  }
}
for (const name of ['app', 'components', 'contexts', 'lib', 'hooks', 'i18n'])
  walk(resolve(root, 'frontend', name));
const files = new Set(findings.map((item) => item.file));
const result = {
  scope:
    'frontend TypeScript literals and explicit Chinese formatters; comments, tests and catalogs excluded; findings require manual classification',
  count: findings.length,
  files: files.size,
  findings,
};
if (process.argv.includes('--write'))
  writeFileSync(
    resolve(root, 'docs/design/frontend-runtime-i18n-gaps.json'),
    `${JSON.stringify(result, null, 2)}\n`,
  );
console.log(
  `Runtime i18n audit: ${findings.length} candidates in ${files.size} files`,
);
