import { computed, ref } from 'vue'
import {
  quickDownload,
  quickUpload,
  type ApplyConflict,
  type QuickOperationResult,
} from '../lib/backend'
import { describeSyncActivity, type SyncActivity } from '../lib/syncActivity'
import { useSettingsStore } from './useSettingsStore'

const settingsStore = useSettingsStore()
const activity = ref<SyncActivity>('')
const status = ref('')
const lastResult = ref<QuickOperationResult | null>(null)
const showResultDetails = ref(false)
const conflicts = ref<ApplyConflict[]>([])
const conflictVisible = ref(false)

const selectedProfileId = computed(() => settingsStore.state.value?.activeProfileId ?? '')
const profiles = computed(() => settingsStore.state.value?.profiles ?? [])

async function initialize(): Promise<void> {
  await settingsStore.ensureLoaded()
  await settingsStore.initializeStartupSync()
}

async function switchProfile(profileId: string): Promise<void> {
  activity.value = 'switching_profile'
  status.value = describeSyncActivity(activity.value)
  try {
    await settingsStore.switchActiveProfile(profileId)
    status.value = ''
  } finally {
    activity.value = ''
  }
}

async function upload(): Promise<void> {
  if (!selectedProfileId.value) {
    status.value = '请先选择配置集'
    return
  }
  activity.value = 'uploading'
  status.value = describeSyncActivity(activity.value)
  try {
    const result = await quickUpload({ profileId: selectedProfileId.value })
    lastResult.value = result
    showResultDetails.value = false
    status.value = `上传完成：快照 ${result.snapshotId || '-'}，文件 ${result.summary.uploaded} 个`
  } catch (error) {
    status.value = `上传失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

async function download(): Promise<void> {
  if (!selectedProfileId.value) {
    status.value = '请先选择配置集'
    return
  }
  activity.value = 'downloading'
  status.value = describeSyncActivity(activity.value)
  try {
    const result = await quickDownload({
      profileId: selectedProfileId.value,
      conflictPolicy: 'manual',
      overwriteItemIds: [],
    })
    if (result.requiresConflictResolution) {
      conflicts.value = result.conflicts.map((item) => ({
        itemId: item.itemId,
        targetPath: item.targetPath,
        diffPreview: item.diffPreview || '',
        diffStatus: item.diffStatus || '',
        diffLines: item.diffLines ?? [],
        addedLines: item.addedLines ?? 0,
        removedLines: item.removedLines ?? 0,
      }))
      conflictVisible.value = true
      status.value = `检测到 ${result.summary.conflicts} 个冲突，默认将全部覆盖，可按需修改`
      return
    }
    lastResult.value = result
    showResultDetails.value = false
    status.value = `下载完成：应用 ${result.summary.applied}，跳过 ${result.summary.skipped}`
  } catch (error) {
    status.value = `下载失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

async function submitConflictDecision(overwriteItemIds: string[]): Promise<void> {
  if (!selectedProfileId.value) {
    return
  }
  activity.value = 'applying_snapshot'
  status.value = describeSyncActivity(activity.value)
  try {
    const result = await quickDownload({
      profileId: selectedProfileId.value,
      conflictPolicy: 'manual',
      overwriteItemIds,
    })
    conflictVisible.value = false
    conflicts.value = []
    lastResult.value = result
    showResultDetails.value = false
    status.value = `下载完成：应用 ${result.summary.applied}，跳过 ${result.summary.skipped}`
  } catch (error) {
    status.value = `下载失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

function closeConflictDialog(): void {
  conflictVisible.value = false
  conflicts.value = []
}

function toggleDetails(): void {
  showResultDetails.value = !showResultDetails.value
}

export function useQuickSyncStore() {
  return {
    selectedProfileId,
    profiles,
    activity,
    status,
    lastResult,
    showResultDetails,
    conflicts,
    conflictVisible,
    initialize,
    switchProfile,
    upload,
    download,
    submitConflictDecision,
    closeConflictDialog,
    toggleDetails,
  }
}
