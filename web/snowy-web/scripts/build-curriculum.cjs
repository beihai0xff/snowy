/* eslint-disable @typescript-eslint/no-require-imports */
/**
 * Snowy v8 · 课纲 YAML → 前端可静态 import 的 JSON
 *
 * 用法：node scripts/build-curriculum.cjs
 * CI 在 build 前跑一次；本地开发也可用 `pnpm curriculum:build`。
 * 不引入 yaml 运行时依赖，使用极小内联解析（仅支持本项目用到的 YAML 子集）。
 */

const fs = require('node:fs');
const path = require('node:path');

const ROOT = path.resolve(__dirname, '..', '..', '..', 'docs', 'curriculum', 'pep');
const OUT = path.resolve(__dirname, '..', 'data', 'curriculum');

// 极简 YAML 解析：支持
//   key: value
//   key: [a, b, c]
//   key: ["a", b, "c"]
//   key:    (开始多行字符串/对象/数组)
//     - item
//     - key2: value
//   嵌套对象用缩进 2 空格
function parseYAML(text) {
  const lines = text.split(/\r?\n/);
  const root = {};
  const stack = [{ obj: root, indent: -1 }];

  function setOnTop(key, value) {
    const top = stack[stack.length - 1].obj;
    if (Array.isArray(top)) {
      const last = top[top.length - 1];
      if (last && typeof last === 'object' && !Array.isArray(last)) {
        last[key] = value;
      } else {
        top.push({ [key]: value });
      }
    } else {
      top[key] = value;
    }
  }

  function unquote(s) {
    s = s.trim();
    if ((s.startsWith('"') && s.endsWith('"')) || (s.startsWith("'") && s.endsWith("'"))) {
      return s.slice(1, -1);
    }
    return s;
  }

  function parseScalar(raw) {
    const v = raw.trim();
    if (v === '') return '';
    if (v === 'true') return true;
    if (v === 'false') return false;
    if (v === 'null' || v === '~') return null;
    if (/^-?\d+$/.test(v)) return parseInt(v, 10);
    if (/^-?\d+\.\d+$/.test(v)) return parseFloat(v);
    // inline array
    if (v.startsWith('[') && v.endsWith(']')) {
      const inner = v.slice(1, -1).trim();
      if (!inner) return [];
      // split by comma not inside quotes
      const out = [];
      let depth = 0, q = null, buf = '';
      for (let i = 0; i < inner.length; i += 1) {
        const c = inner[i];
        if (q) { if (c === q) q = null; buf += c; continue; }
        if (c === '"' || c === "'") { q = c; buf += c; continue; }
        if (c === '[' || c === '{') { depth += 1; buf += c; continue; }
        if (c === ']' || c === '}') { depth -= 1; buf += c; continue; }
        if (c === ',' && depth === 0) { out.push(parseScalar(buf)); buf = ''; continue; }
        buf += c;
      }
      if (buf.trim()) out.push(parseScalar(buf));
      return out;
    }
    return unquote(v);
  }

  // pre-process: drop empty / comment lines, but keep indentation tracking
  for (let i = 0; i < lines.length; i += 1) {
    const raw = lines[i];
    const stripped = raw.replace(/\s+#.*$/, ''); // trailing inline comments
    if (!stripped.trim()) continue;
    if (/^\s*#/.test(stripped)) continue;

    const indentMatch = stripped.match(/^(\s*)/);
    const indent = indentMatch ? indentMatch[1].length : 0;
    let content = stripped.slice(indent);

    // pop stack to current indent
    while (stack.length > 1 && stack[stack.length - 1].indent >= indent) {
      stack.pop();
    }

    // list item
    if (content.startsWith('- ')) {
      content = content.slice(2);
      const top = stack[stack.length - 1].obj;
      if (!Array.isArray(top)) {
        // shouldn't happen if 上一行是 "key:"
        continue;
      }
      // 子项可能是 scalar 或 key: value
      const kv = content.match(/^([A-Za-z_][\w-]*)\s*:\s*(.*)$/);
      if (kv) {
        const obj = {};
        obj[kv[1]] = kv[2] === '' ? null : parseScalar(kv[2]);
        top.push(obj);
        if (kv[2] === '') {
          // 下面是子对象/数组，先用占位
          stack.push({ obj, indent });
          // 嵌套继续等下一行的 key
          stack.push({ obj: null, indent, pendingKey: kv[1] });
        } else {
          stack.push({ obj, indent });
        }
      } else {
        top.push(parseScalar(content));
      }
      continue;
    }

    // key: value
    const m = content.match(/^([A-Za-z_$][\w$-]*)\s*:\s*(.*)$/);
    if (!m) continue;
    const key = m[1];
    const val = m[2];

    // handle pendingKey from prior list item
    let topFrame = stack[stack.length - 1];
    if (topFrame.pendingKey && topFrame.obj === null) {
      stack.pop();
      const carrier = stack[stack.length - 1].obj;
      const last = Array.isArray(carrier) ? carrier[carrier.length - 1] : null;
      if (last && typeof last === 'object') {
        if (val === '') {
          last[topFrame.pendingKey] = [];
          stack.push({ obj: last, indent });
          stack.push({ obj: last[topFrame.pendingKey], indent });
        } else {
          last[topFrame.pendingKey] = parseScalar(val);
        }
        // continue parsing current line as normal
      }
      topFrame = stack[stack.length - 1];
    }

    if (val === '') {
      // opens a child (could be obj or array — decide by next non-empty line)
      let nextNonEmpty = '';
      for (let j = i + 1; j < lines.length; j += 1) {
        const t = lines[j].replace(/\s+#.*$/, '');
        if (!t.trim() || /^\s*#/.test(t)) continue;
        nextNonEmpty = t;
        break;
      }
      const child = nextNonEmpty.trim().startsWith('- ') ? [] : {};
      setOnTop(key, child);
      stack.push({ obj: child, indent });
    } else {
      setOnTop(key, parseScalar(val));
    }
  }

  return root;
}

function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full));
    else if (entry.isFile() && /\.ya?ml$/.test(entry.name)) out.push(full);
  }
  return out;
}

function main() {
  if (!fs.existsSync(ROOT)) {
    console.error('Missing curriculum source dir:', ROOT);
    process.exit(0);
  }
  fs.mkdirSync(OUT, { recursive: true });

  const allBooks = { physics: [], biology: [], chemistry: [] };
  const allPitfalls = { physics: [], biology: [], chemistry: [] };

  for (const f of walk(ROOT)) {
    try {
      const text = fs.readFileSync(f, 'utf8');
      const parsed = parseYAML(text);
      const rel = path.relative(ROOT, f);
      if (rel.startsWith('pitfalls/')) {
        const subject = parsed.subject;
        if (subject && Array.isArray(parsed.pitfalls)) {
          allPitfalls[subject].push(...parsed.pitfalls);
        }
      } else {
        const subject = parsed.subject;
        if (subject) {
          allBooks[subject].push(parsed);
        }
      }
    } catch (err) {
      console.error('parse failed', f, err);
      process.exit(1);
    }
  }

  // 构建 knowledge_tags → 课纲条目反查索引
  const tagIndex = { physics: {}, biology: {}, chemistry: {} };
  for (const subject of Object.keys(allBooks)) {
    for (const book of allBooks[subject]) {
      for (const chapter of book.chapters || []) {
        for (const section of chapter.sections || []) {
          const ref = {
            subject,
            version: book.version,
            book: book.book,
            book_title: book.book_title,
            chapter_id: chapter.id,
            chapter_title: chapter.title,
            section_id: section.id,
            section_title: section.title,
            learning_goals: section.learning_goals || [],
            core_formulas: section.core_formulas || [],
            difficulty: section.difficulty || 'medium',
          };
          for (const tag of section.knowledge_tags || []) {
            const list = tagIndex[subject][tag] || (tagIndex[subject][tag] = []);
            list.push(ref);
          }
        }
      }
    }
  }

  // 写出 per-subject 索引
  for (const subject of Object.keys(allBooks)) {
    const payload = {
      version: 'pep-2019',
      subject,
      books: allBooks[subject],
      tag_index: tagIndex[subject],
      pitfalls: allPitfalls[subject],
    };
    const outFile = path.join(OUT, `${subject}.json`);
    fs.writeFileSync(outFile, JSON.stringify(payload, null, 2), 'utf8');
    console.log('wrote', path.relative(process.cwd(), outFile), '·',
      allBooks[subject].reduce((n, b) => n + (b.chapters || []).reduce((m, c) => m + (c.sections || []).length, 0), 0),
      'sections,', allPitfalls[subject].length, 'pitfalls,',
      Object.keys(tagIndex[subject]).length, 'tags');
  }
}

main();

