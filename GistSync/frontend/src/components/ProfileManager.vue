<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import {
  chooseFilesForProfile,
  createProfile,
  deleteProfile,
  pullProfilesFromCloud,
  removeProfileItems,
} from '../lib/backend'
import { useSettingsStore } from '../composables/useSettingsStore'

const store = useSettingsStore()
const profileName = ref('')
const selectedItemIds = ref<string[]>([])
const status = ref('')
const filterKeyword = ref('')
const restore = reactive({
  mode: 'original' as 'original' | 'rooted',
  root: '',
})

onMounted(async () => {
  await refreshState()
})

async function refreshState(): Promise<void> {
  await store.refresh()
  selectedItemIds.value = []
  const profile = store.activeProfile.value
  if (!profile) {
    restore.mode = 'original'
    restore.root = ''
    return
  }
  restore.mode = profile.restoreMode as 'original' | 'rooted'
  restore.root = profile.restoreRoot
}

async function createNewProfile(): Promise<void> {
  try {
    const created = await createProfile(profileName.value)
    profileName.value = ''
    await store.switchActiveProfile(created.id)
    await refreshState()
    status.value = `已创建配置: ${created.name || created.id}`
  } catch (error) {
    status.value = `创建失败: ${String(error)}`
  }
}

async function switchProfile(profileId: string): Promise<void> {
  try {
    await store.switchActiveProfile(profileId)
    await refreshState()
    status.value = ''
  } catch (error) {
    status.value = `切换失败: ${String(error)}`
  }
}

async function pullFromCloud(): Promise<void> {
  try {
    const count = await pullProfilesFromCloud()
    await refreshState()
    status.value = `已从云端同步配置集，本地共 ${count} 个`
  } catch (error) {
    status.value = `从云端拉取失败: ${String(error)}`
  }
}

async function removeProfile(profileId: string): Promise<void> {
  try {
    await deleteProfile(profileId)
    await refreshState()
    status.value = '配置已删除'
  } catch (error) {
    status.value = `删除失败: ${String(error)}`
  }
}

async function addFiles(): Promise<void> {
  const profile = store.activeProfile.value
  if (!profile) {
    status.value = '请先创建或选择配置集'
    return
  }
  try {
    const selected = await chooseFilesForProfile(profile.id)
    await refreshState()
    status.value = selected.length === 0 ? '未选择文件' : `已添加 ${selected.length} 个文件`
  } catch (error) {
    status.value = `添加文件失败: ${String(error)}`
  }
}

async function removeSelectedItems(): Promise<void> {
  const profile = store.activeProfile.value
  if (!profile || selectedItemIds.value.length === 0) {
    return
  }
  try {
    await removeProfileItems(profile.id, selectedItemIds.value)
    await refreshState()
    status.value = '已移除选中文件'
  } catch (error) {
    status.value = `移除失败: ${String(error)}`
  }
}

async function toggleItemEnabled(itemId: string, enabled: boolean): Promise<void> {
  try {
    await store.setItemsEnabled([itemId], enabled)
  } catch (error) {
    status.value = `更新同步开关失败: ${String(error)}`
  }
}

async function setSelectedEnabled(enabled: boolean): Promise<void> {
  if (selectedItemIds.value.length === 0) {
    return
  }
  try {
    await store.setItemsEnabled([...selectedItemIds.value], enabled)
    status.value = enabled ? '已启用选中文件的同步' : '已暂停选中文件的同步'
  } catch (error) {
    status.value = `批量更新失败: ${String(error)}`
  }
}

const enabledCount = computed(() => (store.activeProfile.value?.items || []).filter((item) => item.enabled).length)
const totalCount = computed(() => (store.activeProfile.value?.items || []).length)

async function persistRestoreSettings(): Promise<void> {
  try {
    await store.updateActiveProfileRestore(restore.mode, restore.root)
  } catch (error) {
    status.value = `保存恢复策略失败: ${String(error)}`
  }
}

const filteredItems = computed(() => {
  const items = store.activeProfile.value?.items || []
  if (!filterKeyword.value.trim()) {
    return items
  }
  const kw = filterKeyword.value.toLowerCase().trim()
  return items.filter((item) => 
    (item.sourcePathTemplate || '').toLowerCase().includes(kw) ||
    (item.relativePath || '').toLowerCase().includes(kw)
  )
})
</script>

<template>
  <section class="space-y-6">
    <section class="grid gap-6 xl:grid-cols-[380px_1fr]">
      <!-- Left sidebar: Profiles list -->
      <aside class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm flex flex-col justify-between">
        <div>
          <div class="mb-4 flex items-center justify-between border-b border-slate-100 pb-3">
            <h2 class="text-base font-bold text-slate-900">配置集列表</h2>
            <span class="rounded-full bg-indigo-50 px-2.5 py-0.5 text-xs font-semibold text-indigo-600 border border-indigo-100">
              {{ store.state.value?.profiles?.length || 0 }} 组
            </span>
          </div>

          <div class="mb-4 space-y-3">
            <div class="flex gap-2">
              <input
                v-model="profileName"
                class="h-10 flex-1 rounded-lg border border-slate-300 px-3 text-sm outline-none transition focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
                placeholder="新建配置集名称"
              >
              <button 
                class="h-10 rounded-lg bg-indigo-600 hover:bg-indigo-700 px-4 text-sm font-semibold text-white transition shadow-sm"
                @click="createNewProfile"
              >
                新建
              </button>
            </div>

            <button 
              class="h-10 w-full inline-flex items-center justify-center gap-1.5 rounded-lg border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-700 hover:bg-slate-50 transition shadow-sm"
              @click="pullFromCloud"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4 text-slate-500">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9.75v6.75m0 0l-3-3m3 3l3-3m-8.25 6a9 9 0 1116.5 0c-1.359-3.083-4.502-5.25-8.25-5.25s-6.891 2.167-8.25 5.25z" />
              </svg>
              从云端拉取配置
            </button>
          </div>

          <!-- List of profiles -->
          <div class="max-h-[300px] space-y-2 overflow-y-auto pr-1">
            <button
              v-for="profile in store.state.value?.profiles || []"
              :key="profile.id"
              class="w-full rounded-xl border p-3.5 text-left text-sm transition"
              :class="profile.id === store.state.value?.activeProfileId 
                ? 'border-indigo-600 bg-indigo-50/50 text-indigo-900 shadow-sm' 
                : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'"
              @click="switchProfile(profile.id)"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="truncate font-semibold">{{ profile.name }}</span>
                <span class="rounded-full px-2 py-0.5 text-xs" 
                  :class="profile.id === store.state.value?.activeProfileId 
                    ? 'bg-indigo-600 text-white font-medium' 
                    : 'bg-slate-100 text-slate-600'"
                >
                  {{ profile.items.length }} 项
                </span>
              </div>
            </button>
          </div>
        </div>

        <div v-if="store.activeProfile.value" class="mt-6 border-t border-slate-100 pt-4">
          <button
            class="h-10 w-full inline-flex items-center justify-center gap-1.5 rounded-lg bg-rose-50 text-rose-700 border border-rose-200 hover:bg-rose-100/80 transition text-sm font-semibold"
            @click="removeProfile(store.activeProfile.value.id)"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
            </svg>
            删除当前配置集
          </button>
        </div>
      </aside>

      <!-- Right side: Profile restore settings -->
      <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm flex flex-col justify-center">
        <div class="mb-4 border-b border-slate-100 pb-3">
          <h3 class="text-sm font-bold text-slate-900">同步恢复策略</h3>
          <p class="mt-1 text-xs text-slate-500">定制此配置集在不同设备上恢复时的路径重构模式（自动保存）。</p>
        </div>
        <div class="grid gap-4 md:grid-cols-[240px_1fr]">
          <div class="flex flex-col gap-1.5">
            <span class="text-xs font-semibold text-slate-500">恢复模式</span>
            <select 
              v-model="restore.mode" 
              class="h-10 rounded-lg border border-slate-300 px-3 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none" 
              @change="persistRestoreSettings"
            >
              <option value="original">原始路径恢复 (Original)</option>
              <option value="rooted">指定根目录恢复 (Rooted)</option>
            </select>
          </div>
          <div class="flex flex-col gap-1.5">
            <span class="text-xs font-semibold text-slate-500">自定义根目录</span>
            <input
              v-model="restore.root"
              class="h-10 rounded-lg border border-slate-300 px-3 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none"
              placeholder="e.g. D:\RecoveredConfigs (rooted模式必填)"
              @blur="persistRestoreSettings"
            >
          </div>
        </div>
      </article>
    </section>

    <!-- Bottom panel: Files / Items list -->
    <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-4 border-b border-slate-100 pb-4">
        <div>
          <h3 class="text-sm font-bold text-slate-900">文件管理条目</h3>
          <p class="mt-1 text-xs text-slate-500">
            为当前配置集指定要追踪的本地配置文件。
            <span v-if="totalCount > 0" class="font-semibold text-slate-600">已启用 {{ enabledCount }} / {{ totalCount }} 个参与一键同步。</span>
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <!-- Search field -->
          <div class="relative mr-2">
            <input
              v-model="filterKeyword"
              class="h-9 w-48 rounded-lg border border-slate-300 pl-8 pr-3 text-xs outline-none transition focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
              placeholder="搜索路径..."
            >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.602 10.602z" />
            </svg>
          </div>

          <button
            v-if="selectedItemIds.length > 0"
            class="h-9 inline-flex items-center gap-1 rounded-lg border border-emerald-200 bg-emerald-50 hover:bg-emerald-100 text-emerald-700 px-3 text-xs font-semibold transition"
            @click="setSelectedEnabled(true)"
          >
            启用选中
          </button>
          <button
            v-if="selectedItemIds.length > 0"
            class="h-9 inline-flex items-center gap-1 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-600 px-3 text-xs font-semibold transition"
            @click="setSelectedEnabled(false)"
          >
            暂停选中
          </button>

          <button
            class="h-9 inline-flex items-center gap-1 rounded-lg bg-indigo-600 hover:bg-indigo-700 px-3 text-xs font-semibold text-white transition shadow-sm"
            @click="addFiles"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-3.5 h-3.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            添加文件 (可多选)
          </button>

          <button
            class="h-9 inline-flex items-center gap-1 rounded-lg border border-rose-200 bg-rose-50 hover:bg-rose-100 text-rose-700 px-3 text-xs font-semibold transition"
            :disabled="selectedItemIds.length === 0"
            @click="removeSelectedItems"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-3.5 h-3.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9" />
            </svg>
            移除选中 ({{ selectedItemIds.length }})
          </button>
        </div>
      </div>

      <div class="overflow-hidden rounded-xl border border-slate-200 shadow-sm">
        <table class="w-full min-w-[800px] text-left text-sm border-collapse">
          <thead class="bg-slate-50 text-slate-600 font-semibold text-xs border-b border-slate-200">
            <tr>
              <th class="w-16 px-4 py-3 text-center">选中</th>
              <th class="w-24 px-4 py-3 text-center">同步</th>
              <th class="px-4 py-3">原始物理路径 (模板)</th>
              <th class="px-4 py-3">云端相对层级 (Gist内文件名)</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr
              v-for="item in filteredItems"
              :key="item.id"
              class="transition"
              :class="item.enabled ? 'hover:bg-slate-50/50' : 'bg-slate-50/40 hover:bg-slate-100/50'"
            >
              <td class="px-4 py-3 text-center">
                <input v-model="selectedItemIds" :value="item.id" type="checkbox" class="rounded text-indigo-600 focus:ring-indigo-500">
              </td>
              <td class="px-4 py-3 text-center">
                <button
                  type="button"
                  role="switch"
                  :aria-checked="item.enabled"
                  class="relative inline-flex h-5 w-9 items-center rounded-full transition focus:outline-none focus:ring-2 focus:ring-indigo-400 focus:ring-offset-1"
                  :class="item.enabled ? 'bg-emerald-500' : 'bg-slate-300'"
                  :title="item.enabled ? '已启用：参与一键同步' : '已暂停：一键同步将跳过'"
                  @click="toggleItemEnabled(item.id, !item.enabled)"
                >
                  <span
                    class="inline-block h-4 w-4 transform rounded-full bg-white shadow transition"
                    :class="item.enabled ? 'translate-x-4' : 'translate-x-1'"
                  />
                </button>
              </td>
              <td class="px-4 py-3 font-mono text-xs select-all" :class="item.enabled ? 'text-slate-700' : 'text-slate-400 line-through'">{{ item.sourcePathTemplate }}</td>
              <td class="px-4 py-3 font-mono text-xs select-all" :class="item.enabled ? 'text-slate-500' : 'text-slate-400'">{{ item.relativePath }}</td>
            </tr>
            <tr v-if="filteredItems.length === 0">
              <td colspan="4" class="px-4 py-12 text-center text-sm text-slate-400">
                <div class="flex flex-col items-center justify-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-8 h-8 text-slate-300">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9z" />
                  </svg>
                  <span>{{ store.activeProfile.value ? '没有匹配的条目，请调整搜索条件或点击“添加文件”' : '未选择或创建配置集，列表为空' }}</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </article>

    <p v-if="status" class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700 whitespace-pre-wrap">{{ status }}</p>
  </section>
</template>
