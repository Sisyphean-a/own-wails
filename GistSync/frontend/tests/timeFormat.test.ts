import assert from 'node:assert/strict'
import test from 'node:test'

import { formatSnapshotCreatedAt } from '../src/lib/timeFormat.ts'

test('formatSnapshotCreatedAt converts UTC timestamp into local wall-clock text', () => {
  const actual = formatSnapshotCreatedAt('2026-05-18T09:37:31Z', { timeZone: 'Asia/Shanghai' })
  assert.equal(actual, '2026-05-18 17:37:31')
})

test('formatSnapshotCreatedAt keeps original text for invalid timestamp', () => {
  assert.equal(formatSnapshotCreatedAt('not-a-time'), 'not-a-time')
})
