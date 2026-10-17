import {describe, expect, it} from 'vitest';
import {chunks, compose} from './diff';
describe('Markdown 冲突选择', () => {
 it('默认保持原文件，按处选择保留版本', () => {
  const parts = chunks('标题\n本机\n结尾\n', '标题\n对端\n结尾\n');
  expect(compose(parts, {})).toBe('标题\n本机\n结尾\n');
  const i = parts.findIndex(p => p.changed);
  expect(compose(parts, {[i]: 'right'})).toBe('标题\n对端\n结尾\n');
  expect(compose(parts, {[i]: 'both'})).toBe('标题\n本机\n对端\n结尾\n');
 });
 it('处理空文件、无尾换行及中文', () => {
  const parts = chunks('', '你好');
  expect(compose(parts, {0: 'right'})).toBe('你好');
  expect(compose(chunks('相同', '相同'), {})).toBe('相同');
 });
});
