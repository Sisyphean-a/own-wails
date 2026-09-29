import { test } from 'node:test'
import { strict as assert } from 'node:assert'
import { buildHistoryTree, expandedPath, searchHistory } from './historyTree.js'

const entry = (id, day, time, text) => ({ id, day, createdAt: `${day}T${time}:00+08:00`, text })

test('one month shows days, a single entry is the day itself', () => {
  const { tree, sorted } = buildHistoryTree([
    entry('a', '2026-06-17', '12:00', '营业通知'),
    entry('b', '2026-06-18', '09:00', '今日促销'),
    entry('c', '2026-06-18', '14:32', '开店提醒'),
  ])
  assert.deepEqual(sorted.map(x => x.id), ['c', 'b', 'a'])
  assert.equal(tree[0].label, '18 日')
  assert.equal(tree[0].children[0].label, '14:32')
  assert.equal(tree[1].label, '17 日')
  assert.equal(tree[1].entry.text, '营业通知')
  assert.equal(tree[1].entry.id, 'a')
  assert.deepEqual(expandedPath(tree, 'c'), ['day:2026-06-18'])
})

test('months and years appear only when needed', () => {
  const june = entry('a', '2026-06-18', '09:00', '六月')
  const may = entry('b', '2026-05-18', '09:00', '五月')
  const old = entry('c', '2025-12-18', '09:00', '去年')
  const months = buildHistoryTree([june, may]).tree
  assert.deepEqual(months.map(x => x.label), ['6 月', '5 月'])
  assert.deepEqual(expandedPath(months, 'a'), ['month:2026-06'])
  const years = buildHistoryTree([june, may, old]).tree
  assert.deepEqual(years.map(x => x.label), ['2026 年', '2025 年'])
  assert.deepEqual(expandedPath(years, 'c'), ['year:2025', 'month:2025-12'])
  assert.deepEqual(buildHistoryTree([]).tree, [])
})

test('global search returns matching lines with date context', () => {
  const entries = [
    entry('new', '2026-06-19', '15:10', '红薯特价每斤 2.99 元\n欢迎选购'),
    entry('old', '2026-06-18', '09:00', '今日红薯每斤 3.49 元，数量有限'),
    entry('other', '2026-06-17', '09:00', '苹果每斤 5.99 元'),
  ]

  const results = searchHistory(entries, ' 红薯 ')
  assert.deepEqual(results.map(result => ({ id: result.entry.id, line: result.line, lineIndex: result.lineIndex })), [
    { id: 'new', line: '红薯特价每斤 2.99 元', lineIndex: 0 },
    { id: 'old', line: '今日红薯每斤 3.49 元，数量有限', lineIndex: 0 },
  ])
  assert.equal(searchHistory(entries, '不存在').length, 0)
  assert.deepEqual(searchHistory(entries.filter(item => item.id !== 'new'), '红薯').map(result => result.entry.id), ['old'])
})
