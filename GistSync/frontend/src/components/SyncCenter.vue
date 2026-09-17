<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import ConflictResolverDialog from './ConflictResolverDialog.vue'
import {
  applySnapshot,
  previewApplyConflicts,
  uploadProfile,
  type ApplyConflict,
  type ApplySnapshotRequest,
  type SnapshotMeta,
} from '../lib/backend'
import {
  advancedDownloadButtonLabel,
  advancedUploadButtonLabel,
  describeSyncActivity,
  isBusyActivity,
  type SyncActivity,
} from '../lib/syncActivity'
import { formatSnapshotCreatedAt } from '../lib/timeFormat'
import { useSettingsStore } from '../composables/useSettingsStore'

const store = useSettingsStore()
const snapshots = ref<SnapshotMeta[]>([])
const selectedSnapshotId = ref('')
const selectedUploadItemIds = ref<string[]>([])
const selectedRestoreItemIds = ref<string[]>([])
const status = ref('')
const activity = ref<SyncActivity>('')
const conflictVisible = ref(false)
const conflicts = ref<ApplyConflict[]>([])
const pendingApplyRequest = ref<ApplySnapshotRequest | null>(null)
const busy = computed(() => isBusyActivity(activity.value))
const busyDescription = computed(() => describeSyncActivity(activity.value))
const uploadButtonLabel = computed(() => advancedUploadButtonLabel(activity.value))
const downloadButtonLabel = computed(() => advancedDownloadButtonLabel(activity.value))
const snapshotSyncing = computed(() => store.snapshotLoading.value)
const snapshotSyncError = computed(() => store.snapshotError.value)
const localizedSnapshots = computed(() =>
  snapshots.value.map((snapshot) => ({
    ...snapshot,
    displayCreatedAt: formatSnapshotCreatedAt(snapshot.createdAt),
  })),
)

const uploadKeyword = ref('')
const restoreKeyword = ref('')

onMounted(async () => {
  await refreshState()
})

watch(
  () => store.state.value?.activeProfileId ?? '',
  (profileId) => {
    if (!profileId) {
      snapshots.value = []
      selectedSnapshotId.value = ''
      return
    }
    snapshots.value = store.getSnapshots(profileId)
    selectedSnapshotId.value = snapshots.value[0]?.id ?? ''
  },
)

async function refreshState(): Promise<void> {
  await store.refresh()
  const profile = store.activeProfile.value
  if (!profile) {
    snapshots.value = []
    selectedSnapshotId.value = ''
    selectedUploadItemIds.value = []
    selectedRestoreItemIds.value = []
    return
  }
  const enabledIds = profile.items.filter((item) => item.enabled).map((item) => item.id)
  selectedUploadItemIds.value = enabledIds
  selectedRestoreItemIds.value = enabledIds
  snapshots.value = store.getSnapshots(profile.id)
  selectedSnapshotId.value = snapshots.value[0]?.id ?? ''
}

async function syncSnapshots(profileId: string): Promise<void> {
  activity.value = 'loading_snapshots'
  const pendingStatus = describeSyncActivity(activity.value)
  status.value = pendingStatus
  try {
    snapshots.value = await store.refreshSnapshots(profileId)
    selectedSnapshotId.value = snapshots.value[0]?.id ?? ''
    if (status.value === pendingStatus) {
      status.value = ''
    }
  } catch (error) {
    status.value = `快照同步失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

async function switchProfile(profileId: string): Promise<void> {
  activity.value = 'switching_profile'
  status.value = describeSyncActivity(activity.value)
  try {
    await store.switchActiveProfile(profileId)
    await refreshState()
  } catch (error) {
    status.value = `切换失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

async function uploadSelectedItems(): Promise<void> {
  const profile = store.activeProfile.value
  if (!profile) {
    status.value = '请先选择配置集'
    return
  }
  if (selectedUploadItemIds.value.length === 0) {
    status.value = '请至少选择一个上传条目'
    return
  }
  activity.value = 'uploading'
  status.value = describeSyncActivity(activity.value)
  try {
    const result = await uploadProfile(profile.id, selectedUploadItemIds.value)
    await syncSnapshots(profile.id)
    status.value = `上传完成：快照 ${result.snapshotId}，文件 ${result.uploaded} 个`
  } catch (error) {
    status.value = `上传失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

async function startApplyFlow(): Promise<void> {
  const profile = store.activeProfile.value
  if (!profile) {
    status.value = '请先选择配置集'
    return
  }
  if (selectedRestoreItemIds.value.length === 0) {
    status.value = '请至少选择一个恢复条目'
    return
  }
  const request: ApplySnapshotRequest = {
    profileId: profile.id,
    snapshotId: selectedSnapshotId.value,
    masterPassword: '',
    restoreMode: profile.restoreMode,
    restoreRoot: profile.restoreRoot,
    selectedItemIds: selectedRestoreItemIds.value,
    overwriteItemIds: [],
  }

  activity.value = 'checking_conflicts'
  status.value = describeSyncActivity(activity.value)
  try {
    const found = await previewApplyConflicts(request)
    if (found.length === 0) {
      activity.value = 'applying_snapshot'
      status.value = describeSyncActivity(activity.value)
      const result = await applySnapshot(request)
      status.value = `应用完成：成功 ${result.applied}，跳过 ${result.skipped}`
      return
    }
    conflicts.value = found
    pendingApplyRequest.value = request
    conflictVisible.value = true
    status.value = `检测到 ${found.length} 个冲突，请确认覆盖策略`
  } catch (error) {
    status.value = `应用失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

async function submitConflictDecision(overwriteItemIds: string[]): Promise<void> {
  const request = pendingApplyRequest.value
  if (!request) {
    return
  }
  activity.value = 'applying_snapshot'
  status.value = describeSyncActivity(activity.value)
  try {
    const result = await applySnapshot({
      ...request,
      overwriteItemIds,
    })
    conflictVisible.value = false
    pendingApplyRequest.value = null
    conflicts.value = []
    status.value = `应用完成：成功 ${result.applied}，跳过 ${result.skipped}`
  } catch (error) {
    status.value = `应用失败: ${String(error)}`
  } finally {
    activity.value = ''
  }
}

function closeConflictDialog(): void {
  conflictVisible.value = false
  pendingApplyRequest.value = null
}

async function manualSyncSnapshots(): Promise<void> {
  const profileId = store.activeProfile.value?.id
  if (!profileId) {
    status.value = '请先选择配置集'
    return
  }
  await syncSnapshots(profileId)
}

const filteredUploadItems = computed(() => {
  const items = store.activeProfile.value?.items || []
  if (!uploadKeyword.value.trim()) {
    return items
  }
  const kw = uploadKeyword.value.toLowerCase().trim()
  return items.filter((item) => 
    (item.relativePath || '').toLowerCase().includes(kw) ||
    (item.sourcePathTemplate || '').toLowerCase().includes(kw)
  )
})

const filteredRestoreItems = computed(() => {
  const items = store.activeProfile.value?.items || []
  if (!restoreKeyword.value.trim()) {
    return items
  }
  const kw = restoreKeyword.value.toLowerCase().trim()
  return items.filter((item) =>
    (item.relativePath || '').toLowerCase().includes(kw) ||
    (item.sourcePathTemplate || '').toLowerCase().includes(kw)
  )
})

function selectUpload(mode: 'all' | 'none' | 'enabled'): void {
  const items = filteredUploadItems.value
  if (mode === 'none') {
    selectedUploadItemIds.value = []
    return
  }
  selectedUploadItemIds.value = items
    .filter((item) => mode === 'all' || item.enabled)
    .map((item) => item.id)
}

function selectRestore(mode: 'all' | 'none' | 'enabled'): void {
  const items = filteredRestoreItems.value
  if (mode === 'none') {
    selectedRestoreItemIds.value = []
    return
  }
  selectedRestoreItemIds.value = items
    .filter((item) => mode === 'all' || item.enabled)
    .map((item) => item.id)
}
</script>

<template>
  <section class="space-y-6">
    <!-- Dropdowns selector -->
    <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="mb-4 border-b border-slate-100 pb-3">
        <h2 class="text-base font-bold text-slate-900">配置选择与快照归档</h2>
        <p class="mt-1 text-xs text-slate-500">选择要操作的配置集与历史快照，定制细粒度的备份和恢复。</p>
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div class="flex flex-col gap-1">
          <span class="text-xs font-semibold text-slate-500">当前配置集</span>
          <select
            :value="store.state.value?.activeProfileId || ''"
            class="h-10 w-full rounded-lg border border-slate-300 px-3 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none"
            :disabled="busy"
            @change="switchProfile(($event.target as HTMLSelectElement).value)"
          >
            <option v-for="profile in store.state.value?.profiles || []" :key="profile.id" :value="profile.id">{{ profile.name }}</option>
          </select>
        </div>
        <div class="flex flex-col gap-1">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold text-slate-500">快照选择（历史归档）</span>
            <button
              type="button"
              class="text-[11px] font-semibold text-indigo-600 hover:text-indigo-800 disabled:text-slate-400"
              :disabled="busy || snapshotSyncing"
              @click="manualSyncSnapshots"
            >
              {{ snapshotSyncing ? '同步中...' : '同步快照' }}
            </button>
          </div>
          <select v-model="selectedSnapshotId" class="h-10 w-full rounded-lg border border-slate-300 px-3 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none" :disabled="busy">
            <option value="" disabled>请选择快照（默认最新）</option>
            <option v-for="snapshot in localizedSnapshots" :key="snapshot.id" :value="snapshot.id">
              {{ snapshot.displayCreatedAt }} - {{ snapshot.id.slice(0, 8) }}...
            </option>
          </select>
          <p v-if="snapshotSyncError" class="text-[11px] text-rose-600">{{ snapshotSyncError }}</p>
        </div>
      </div>
    </article>

    <!-- Busy indicator -->
    <article v-if="busy" class="rounded-2xl border border-sky-100 bg-sky-50/50 p-4 shadow-sm">
      <div class="flex items-center gap-3">
        <span class="h-5 w-5 animate-spin rounded-full border-2 border-sky-200 border-t-indigo-600" />
        <div>
          <p class="text-sm font-bold text-slate-900">{{ busyDescription }}</p>
          <p class="text-xs text-slate-500">同步操作运行中，部分选项暂时处于锁定状态。</p>
        </div>
      </div>
    </article>

    <!-- Upload & Download columns -->
    <section class="grid gap-6 xl:grid-cols-2">
      <!-- Upload Card -->
      <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm flex flex-col justify-between">
        <div>
          <div class="mb-3 flex items-center justify-between border-b border-slate-100 pb-3">
            <h3 class="text-sm font-bold text-slate-900">上传到云端</h3>
            <div class="relative">
              <input
                v-model="uploadKeyword"
                class="h-8 w-40 rounded-lg border border-slate-300 pl-7 pr-3 text-[11px] outline-none transition focus:border-indigo-500"
                placeholder="搜索条目..."
              >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-3 h-3 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400">
                <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.602 10.602z" />
              </svg>
            </div>
          </div>
          
          <div class="mb-2 flex items-center justify-between text-[11px]">
            <div class="flex items-center gap-2">
              <button class="font-semibold text-indigo-600 hover:text-indigo-800 disabled:text-slate-300" :disabled="busy" @click="selectUpload('all')">全选</button>
              <span class="text-slate-300">·</span>
              <button class="font-semibold text-indigo-600 hover:text-indigo-800 disabled:text-slate-300" :disabled="busy" @click="selectUpload('enabled')">仅启用</button>
              <span class="text-slate-300">·</span>
              <button class="font-semibold text-slate-500 hover:text-slate-700 disabled:text-slate-300" :disabled="busy" @click="selectUpload('none')">清空</button>
            </div>
            <span class="text-slate-400">已选 {{ selectedUploadItemIds.length }} / {{ filteredUploadItems.length }}</span>
          </div>

          <div class="mb-4 max-h-[300px] overflow-y-auto rounded-xl border border-slate-200 bg-slate-50/50 p-2 space-y-1">
            <label
              v-for="item in filteredUploadItems"
              :key="`upload-${item.id}`"
              class="flex items-center gap-3 rounded-lg border border-slate-200/80 bg-white px-3 py-2 text-xs transition hover:bg-slate-50 cursor-pointer shadow-sm"
            >
              <input v-model="selectedUploadItemIds" :disabled="busy" :value="item.id" type="checkbox" class="rounded text-indigo-600 focus:ring-indigo-500">
              <div class="truncate flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="font-semibold text-slate-800 truncate">{{ item.relativePath }}</span>
                  <span v-if="!item.enabled" class="shrink-0 rounded-full bg-slate-100 px-1.5 py-0.5 text-[9px] font-semibold text-slate-500 border border-slate-200">已暂停</span>
                </div>
                <div class="text-[10px] text-slate-400 font-mono truncate">{{ item.sourcePathTemplate }}</div>
              </div>
            </label>
            <div v-if="filteredUploadItems.length === 0" class="text-center py-8 text-xs text-slate-400">
              暂无匹配的可上传文件
            </div>
          </div>
        </div>

        <button 
          class="h-10 w-full inline-flex items-center justify-center gap-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 px-4 text-sm font-semibold text-white transition shadow-sm disabled:opacity-60" 
          :disabled="busy" 
          @click="uploadSelectedItems"
        >
          <span v-if="activity === 'uploading'" class="h-4 w-4 animate-spin rounded-full border-2 border-emerald-200 border-t-white" />
          <span>{{ uploadButtonLabel }}</span>
        </button>
      </article>

      <!-- Restore Card -->
      <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm flex flex-col justify-between">
        <div>
          <div class="mb-3 flex items-center justify-between border-b border-slate-100 pb-3">
            <h3 class="text-sm font-bold text-slate-900">从云端恢复</h3>
            <div class="relative">
              <input
                v-model="restoreKeyword"
                class="h-8 w-40 rounded-lg border border-slate-300 pl-7 pr-3 text-[11px] outline-none transition focus:border-indigo-500"
                placeholder="搜索条目..."
              >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-3 h-3 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400">
                <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.602 10.602z" />
              </svg>
            </div>
          </div>
          
          <div class="mb-2 flex items-center justify-between text-[11px]">
            <div class="flex items-center gap-2">
              <button class="font-semibold text-indigo-600 hover:text-indigo-800 disabled:text-slate-300" :disabled="busy" @click="selectRestore('all')">全选</button>
              <span class="text-slate-300">·</span>
              <button class="font-semibold text-indigo-600 hover:text-indigo-800 disabled:text-slate-300" :disabled="busy" @click="selectRestore('enabled')">仅启用</button>
              <span class="text-slate-300">·</span>
              <button class="font-semibold text-slate-500 hover:text-slate-700 disabled:text-slate-300" :disabled="busy" @click="selectRestore('none')">清空</button>
            </div>
            <span class="text-slate-400">已选 {{ selectedRestoreItemIds.length }} / {{ filteredRestoreItems.length }}</span>
          </div>

          <div class="mb-4 max-h-[300px] overflow-y-auto rounded-xl border border-slate-200 bg-slate-50/50 p-2 space-y-1">
            <label
              v-for="item in filteredRestoreItems"
              :key="`restore-${item.id}`"
              class="flex items-center gap-3 rounded-lg border border-slate-200/80 bg-white px-3 py-2 text-xs transition hover:bg-slate-50 cursor-pointer shadow-sm"
            >
              <input v-model="selectedRestoreItemIds" :disabled="busy" :value="item.id" type="checkbox" class="rounded text-indigo-600 focus:ring-indigo-500">
              <div class="truncate flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="font-semibold text-slate-800 truncate">{{ item.relativePath }}</span>
                  <span v-if="!item.enabled" class="shrink-0 rounded bg-slate-100 px-1.5 py-0.5 text-[9px] font-semibold text-slate-500 border border-slate-200">已暂停</span>
                </div>
                <div class="text-[10px] text-slate-400 font-mono truncate">{{ item.sourcePathTemplate }}</div>
              </div>
            </label>
            <div v-if="filteredRestoreItems.length === 0" class="text-center py-8 text-xs text-slate-400">
              暂无匹配的可恢复文件
            </div>
          </div>
        </div>

        <button 
          class="h-10 w-full inline-flex items-center justify-center gap-2 rounded-lg bg-amber-600 hover:bg-amber-700 px-4 text-sm font-semibold text-white transition shadow-sm disabled:opacity-60" 
          :disabled="busy" 
          @click="startApplyFlow"
        >
          <span v-if="activity === 'checking_conflicts' || activity === 'applying_snapshot'" class="h-4 w-4 animate-spin rounded-full border-2 border-amber-200 border-t-white" />
          <span>{{ downloadButtonLabel }}</span>
        </button>
      </article>
    </section>

    <!-- Log/Status Display -->
    <div v-if="status" class="rounded-xl border border-slate-200 bg-slate-50 p-4 border-l-4 border-l-indigo-600">
      <p class="text-xs font-semibold uppercase tracking-wider text-indigo-600 mb-1">系统日志 / 同步状态</p>
      <p class="text-sm text-slate-700 whitespace-pre-wrap">{{ status }}</p>
    </div>

    <!-- Conflict Dialog -->
    <ConflictResolverDialog
      :visible="conflictVisible"
      :conflicts="conflicts"
      :submitting="activity === 'applying_snapshot'"
      @close="closeConflictDialog"
      @confirm="submitConflictDecision"
    />
  </section>
</template>
