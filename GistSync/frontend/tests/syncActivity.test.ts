import assert from 'node:assert/strict'
import test from 'node:test'

import {
  advancedDownloadButtonLabel,
  advancedUploadButtonLabel,
  describeSyncActivity,
  isBusyActivity,
  quickDownloadButtonLabel,
  quickUploadButtonLabel,
  type SyncActivity,
} from '../src/lib/syncActivity.ts'

test('describeSyncActivity returns human-readable loading text', () => {
  const cases: Array<[SyncActivity, string]> = [
    ['switching_profile', '正在切换配置集...'],
    ['loading_snapshots', '正在加载最新快照...'],
    ['uploading', '正在上传文件到云端...'],
    ['downloading', '正在拉取云端数据并同步到本地...'],
    ['checking_conflicts', '正在检查本地冲突...'],
    ['applying_snapshot', '正在写入文件到本地...'],
  ]

  for (const [activity, expected] of cases) {
    assert.equal(describeSyncActivity(activity), expected)
  }
})

test('button labels change only for their own active operation', () => {
  assert.equal(quickUploadButtonLabel('uploading'), '正在上传配置...')
  assert.equal(quickUploadButtonLabel('downloading'), '一键上传配置')
  assert.equal(quickDownloadButtonLabel('downloading'), '正在下载更新...')
  assert.equal(quickDownloadButtonLabel('applying_snapshot'), '正在执行覆盖...')
  assert.equal(advancedUploadButtonLabel('uploading'), '正在上传选中条目...')
  assert.equal(advancedDownloadButtonLabel('checking_conflicts'), '正在预检冲突...')
  assert.equal(advancedDownloadButtonLabel('applying_snapshot'), '正在应用快照...')
})

test('isBusyActivity detects any non-idle state', () => {
  assert.equal(isBusyActivity(''), false)
  assert.equal(isBusyActivity('uploading'), true)
  assert.equal(isBusyActivity('loading_snapshots'), true)
})
