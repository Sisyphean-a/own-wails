<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ConflictResolverDialog from './components/ConflictResolverDialog.vue'
import SettingsPanel from './components/SettingsPanel.vue'
import SyncCenter from './components/SyncCenter.vue'
import ProfileManager from './components/ProfileManager.vue'
import { useQuickSyncStore } from './composables/useQuickSyncStore'
import { useSettingsStore } from './composables/useSettingsStore'
import {
  describeSyncActivity,
  isBusyActivity,
  quickDownloadButtonLabel,
  quickUploadButtonLabel,
} from './lib/syncActivity'

type Tab = 'quick' | 'sync' | 'profiles' | 'settings'

const currentTab = ref<Tab>('quick')
const quickStore = useQuickSyncStore()
const settingsStore = useSettingsStore()

const {
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
} = quickStore

const busy = computed(() => isBusyActivity(activity.value))
const busyDescription = computed(() => describeSyncActivity(activity.value))
const uploadButtonLabel = computed(() => quickUploadButtonLabel(activity.value))
const downloadButtonLabel = computed(() => quickDownloadButtonLabel(activity.value))
const downloadBusy = computed(() => activity.value === 'downloading' || activity.value === 'applying_snapshot')
const uploadBusy = computed(() => activity.value === 'uploading')
const startupSyncReady = computed(() => settingsStore.startupSyncReady.value)
const startupSyncBusy = computed(() => settingsStore.snapshotLoading.value)
const startupSyncError = computed(() => settingsStore.snapshotError.value)

// Credentials status
const hasToken = computed(() => !!settingsStore.state.value?.token)
const hasPassword = computed(() => !!settingsStore.state.value?.masterPassword)
const isConfigured = computed(() => hasToken.value && hasPassword.value)

// Active profile metadata
const activeProfileDetails = computed(() => {
  return profiles.value.find((p) => p.id === selectedProfileId.value) || null
})
const enabledItemCount = computed(() => (activeProfileDetails.value?.items ?? []).filter((item) => item.enabled).length)
const totalItemCount = computed(() => activeProfileDetails.value?.items?.length ?? 0)
const noEnabledItems = computed(() => totalItemCount.value > 0 && enabledItemCount.value === 0)
const quickActionsDisabled = computed(
  () => busy.value || !selectedProfileId.value || !startupSyncReady.value || noEnabledItems.value,
)

onMounted(async () => {
  await initialize()
})
</script>

<template>
  <div class="flex h-full w-full bg-slate-50 text-slate-800 antialiased font-sans">
    <!-- Sleek Left Sidebar -->
    <aside class="w-64 bg-slate-100 text-slate-600 flex flex-col justify-between shrink-0 shadow-sm border-r border-slate-200">
      <div class="flex-1 flex flex-col min-h-0">
        <!-- Logo Header -->
        <div class="flex items-center gap-3 px-6 py-5 border-b border-slate-200/60">
          <div class="h-9 w-9 rounded-xl bg-gradient-to-tr from-indigo-500 to-violet-600 flex items-center justify-center shadow-md">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-5 h-5 text-white">
              <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
            </svg>
          </div>
          <div>
            <h1 class="text-base font-extrabold tracking-wider text-slate-800">GistSync</h1>
            <p class="text-[10px] text-slate-400 font-semibold tracking-widest uppercase">Encryption tool</p>
          </div>
        </div>

        <!-- Global Select Profile -->
        <div class="px-4 py-4 border-b border-slate-200/40">
          <label class="block text-[11px] font-bold text-slate-400 uppercase tracking-wider mb-2">当前配置集</label>
          <div class="relative">
            <select
              :value="selectedProfileId"
              class="h-9 w-full rounded-lg bg-white hover:bg-slate-50 border border-slate-300 px-3 text-xs text-slate-700 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition cursor-pointer appearance-none shadow-sm"
              :disabled="busy"
              @change="switchProfile(($event.target as HTMLSelectElement).value)"
            >
              <option value="" disabled>请选择配置集</option>
              <option v-for="profile in profiles" :key="profile.id" :value="profile.id">{{ profile.name }}</option>
            </select>
            <div class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-slate-400">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4">
                <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
              </svg>
            </div>
          </div>
          <div v-if="activeProfileDetails" class="mt-2 flex items-center justify-between text-[10px] text-slate-500 px-1">
            <span>参与同步:</span>
            <span class="font-bold" :class="noEnabledItems ? 'text-rose-500' : 'text-slate-700'">{{ enabledItemCount }} / {{ totalItemCount }} 个</span>
          </div>
        </div>

        <!-- Navigation Menu -->
        <nav class="flex-1 px-3 py-4 space-y-1.5 overflow-y-auto">
          <!-- Quick Sync Tab -->
          <button
            class="w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-semibold transition"
            :class="currentTab === 'quick' 
              ? 'bg-gradient-to-r from-indigo-600 to-indigo-700 text-white shadow-md' 
              : 'hover:bg-slate-200/80 text-slate-500 hover:text-slate-800'"
            @click="currentTab = 'quick'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
            </svg>
            一键同步
          </button>

          <!-- Advanced Sync Tab -->
          <button
            class="w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-semibold transition"
            :class="currentTab === 'sync' 
              ? 'bg-gradient-to-r from-indigo-600 to-indigo-700 text-white shadow-md' 
              : 'hover:bg-slate-200/80 text-slate-500 hover:text-slate-800'"
            @click="currentTab = 'sync'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
            </svg>
            高级同步
          </button>

          <!-- Profile Manager Tab -->
          <button
            class="w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-semibold transition"
            :class="currentTab === 'profiles' 
              ? 'bg-gradient-to-r from-indigo-600 to-indigo-700 text-white shadow-md' 
              : 'hover:bg-slate-200/80 text-slate-500 hover:text-slate-800'"
            @click="currentTab = 'profiles'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A9 9 0 0112 3v8.25m0 0h8.25c0-1.617-.432-3.134-1.187-4.444L12.5 10.5H21" />
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 13.5A9 9 0 0012 22.5V13.5H2.25z" />
            </svg>
            配置管理
          </button>

          <!-- Security settings Tab -->
          <button
            class="w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-semibold transition"
            :class="currentTab === 'settings' 
              ? 'bg-gradient-to-r from-indigo-600 to-indigo-700 text-white shadow-md' 
              : 'hover:bg-slate-200/80 text-slate-500 hover:text-slate-800'"
            @click="currentTab = 'settings'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.43l-1.003.828c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.43l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.991l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.28z" />
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
            </svg>
            安全设置
          </button>
        </nav>
      </div>

      <!-- Sidebar Footer Settings Badges -->
      <div class="p-4 border-t border-slate-200 bg-slate-200/45 space-y-2">
        <div class="flex items-center justify-between text-xs">
          <span class="text-slate-400">GitHub API:</span>
          <span class="inline-flex items-center gap-1.5 font-semibold text-slate-700">
            <span class="h-2 w-2 rounded-full" :class="hasToken ? 'bg-emerald-500' : 'bg-rose-500'" />
            {{ hasToken ? '已配置' : '未配置' }}
          </span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-slate-400">解密主密码:</span>
          <span class="inline-flex items-center gap-1.5 font-semibold text-slate-700">
            <span class="h-2 w-2 rounded-full" :class="hasPassword ? 'bg-emerald-500' : 'bg-rose-500'" />
            {{ hasPassword ? '已配置' : '未配置' }}
          </span>
        </div>
      </div>
    </aside>

    <!-- Main Content Area -->
    <main class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <!-- Top header bar -->
      <header class="bg-white border-b border-slate-200 px-8 py-4 flex items-center justify-between shadow-sm z-10">
        <div>
          <span class="text-[10px] font-bold text-indigo-600 uppercase tracking-widest">
            {{ currentTab === 'quick' ? 'Quick Operations' : currentTab === 'sync' ? 'Advanced Flow' : currentTab === 'profiles' ? 'Profile Management' : 'System Settings' }}
          </span>
          <h2 class="text-xl font-extrabold text-slate-900 mt-0.5">
            {{ currentTab === 'quick' ? '一键同步' : currentTab === 'sync' ? '高级同步' : currentTab === 'profiles' ? '配置管理' : '安全设置' }}
          </h2>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-xs text-slate-400">GistSync Desktop v1.0</span>
        </div>
      </header>

      <!-- Scrollable panel container -->
      <div class="flex-1 overflow-y-auto p-6 md:p-8">
        <!-- Render page based on tab selection -->
        
        <!-- Quick Sync Page -->
        <div v-if="currentTab === 'quick'" class="space-y-6 max-w-5xl mx-auto">
          <!-- Step 1: Onboarding check -->
          <div v-if="!isConfigured" class="rounded-2xl border border-indigo-100 bg-white p-6 shadow-sm flex flex-col md:flex-row items-center gap-5">
            <div class="h-12 w-12 rounded-xl bg-indigo-50 flex items-center justify-center shrink-0">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-indigo-600">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div class="flex-1 text-center md:text-left">
              <h3 class="text-sm font-bold text-slate-900">⚠️ 系统尚未完全配置</h3>
              <p class="text-xs text-slate-500 mt-1">您需要先配置 GitHub Token 和本地加解密主密码以启用同步功能。</p>
            </div>
            <button class="rounded-lg bg-indigo-600 hover:bg-indigo-700 px-4 py-2 text-xs font-semibold text-white transition shadow-sm" @click="currentTab = 'settings'">
              立即去配置
            </button>
          </div>

          <!-- Step 2: Onboarding profile check -->
          <div v-else-if="profiles.length === 0" class="rounded-2xl border border-indigo-100 bg-white p-6 shadow-sm flex flex-col md:flex-row items-center gap-5">
            <div class="h-12 w-12 rounded-xl bg-indigo-50 flex items-center justify-center shrink-0">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-indigo-600">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
              </svg>
            </div>
            <div class="flex-1 text-center md:text-left">
              <h3 class="text-sm font-bold text-slate-900">📂 未创建配置文件组</h3>
              <p class="text-xs text-slate-500 mt-1">当前凭证已准备就绪，请创建一个配置集以指定需要追踪的本地配置文件。</p>
            </div>
            <button class="rounded-lg bg-indigo-600 hover:bg-indigo-700 px-4 py-2 text-xs font-semibold text-white transition shadow-sm" @click="currentTab = 'profiles'">
              创建配置集
            </button>
          </div>

          <!-- Standard Quick Sync Panels -->
          <div v-else class="space-y-6">
            <!-- Loading active state -->
            <div v-if="startupSyncBusy || busy" class="rounded-2xl border border-indigo-100 bg-indigo-50/40 p-4 shadow-sm">
              <div class="flex items-center gap-3">
                <span class="h-5 w-5 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600" />
                <div>
                  <p class="text-sm font-bold text-indigo-900">{{ startupSyncBusy ? '正在初始化同步数据...' : busyDescription }}</p>
                  <p class="text-xs text-indigo-700">
                    {{ startupSyncBusy ? '启动时正在拉取当前配置与快照，完成前不可执行一键同步。' : '这可能需要几秒钟，在此过程中请勿关闭应用。' }}
                  </p>
                </div>
              </div>
            </div>
            <div v-if="startupSyncError" class="rounded-2xl border border-rose-200 bg-rose-50 p-4 shadow-sm">
              <p class="text-sm font-bold text-rose-700">初始化同步失败</p>
              <p class="text-xs text-rose-600 mt-1 whitespace-pre-wrap">{{ startupSyncError }}</p>
            </div>

            <!-- No enabled items warning -->
            <div v-if="noEnabledItems" class="rounded-2xl border border-amber-200 bg-amber-50 p-4 shadow-sm flex items-start gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-amber-600 shrink-0 mt-0.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
              </svg>
              <div class="flex-1">
                <p class="text-sm font-bold text-amber-800">当前配置集没有启用任何文件</p>
                <p class="text-xs text-amber-700 mt-0.5">一键同步将不会处理任何文件。请前往配置管理启用需要同步的条目。</p>
              </div>
              <button class="rounded-lg bg-amber-600 hover:bg-amber-700 px-3 py-1.5 text-xs font-semibold text-white transition shadow-sm shrink-0" @click="currentTab = 'profiles'">
                去启用
              </button>
            </div>

            <!-- Double Columns Operations -->
            <div class="grid gap-6 md:grid-cols-2">
              <!-- Card: Download -->
              <article class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition flex flex-col justify-between">
                <div>
                  <div class="h-10 w-10 rounded-xl bg-amber-50 flex items-center justify-center mb-4">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.2" stroke="currentColor" class="w-5 h-5 text-amber-600">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                    </svg>
                  </div>
                  <h3 class="text-base font-bold text-slate-900">下载更新本地配置</h3>
                  <p class="mt-2 text-xs text-slate-500 leading-relaxed">
                    从云端 GitHub Gist 下载最新的加密快照。若本地文件有变动，将会提示冲突决策，默认完全覆盖本地文件。
                  </p>
                </div>
                
                <button
                  class="mt-6 h-10 w-full rounded-lg bg-amber-600 hover:bg-amber-700 px-5 text-sm font-semibold text-white transition shadow-sm disabled:opacity-60 flex items-center justify-center gap-2"
                  :disabled="quickActionsDisabled"
                  @click="download"
                >
                  <span v-if="downloadBusy" class="h-4 w-4 animate-spin rounded-full border-2 border-amber-200 border-t-white" />
                  <span>{{ downloadButtonLabel }}</span>
                </button>
              </article>

              <!-- Card: Upload -->
              <article class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm hover:shadow-md transition flex flex-col justify-between">
                <div>
                  <div class="h-10 w-10 rounded-xl bg-emerald-50 flex items-center justify-center mb-4">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.2" stroke="currentColor" class="w-5 h-5 text-emerald-600">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
                    </svg>
                  </div>
                  <h3 class="text-base font-bold text-slate-900">上传新的备份</h3>
                  <p class="mt-2 text-xs text-slate-500 leading-relaxed">
                    将当前选择的配置集中所有选中的本地文件，通过安全主密码加密后整合上传至云端 Gist 归档，并生成新的时间戳快照。
                  </p>
                </div>

                <button
                  class="mt-6 h-10 w-full rounded-lg bg-emerald-600 hover:bg-emerald-700 px-5 text-sm font-semibold text-white transition shadow-sm disabled:opacity-60 flex items-center justify-center gap-2"
                  :disabled="quickActionsDisabled"
                  @click="upload"
                >
                  <span v-if="uploadBusy" class="h-4 w-4 animate-spin rounded-full border-2 border-emerald-200 border-t-white" />
                  <span>{{ uploadButtonLabel }}</span>
                </button>
              </article>
            </div>

            <!-- Sync Activity Logs -->
            <div v-if="status" class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
              <div class="flex items-center justify-between mb-3 border-b border-slate-100 pb-3">
                <div class="flex items-center gap-2">
                  <span class="h-2 w-2 rounded-full bg-indigo-500" />
                  <span class="text-xs font-bold text-slate-500 uppercase tracking-wider">最新操作结果</span>
                </div>
                <button
                  v-if="lastResult"
                  class="inline-flex items-center gap-1 text-xs font-semibold text-indigo-600 hover:text-indigo-800 transition"
                  @click="toggleDetails"
                >
                  <span>{{ showResultDetails ? '收起明细' : '展开明细' }}</span>
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 transition-transform duration-200" :class="showResultDetails ? 'rotate-180' : ''">
                    <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
                  </svg>
                </button>
              </div>
              <p class="text-sm font-semibold text-slate-800 whitespace-pre-wrap">{{ status }}</p>

              <!-- Collapsible table details -->
              <div v-if="lastResult && showResultDetails" class="mt-4 overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50 shadow-sm">
                <table class="w-full text-left text-xs border-collapse">
                  <thead class="bg-slate-100 text-slate-600 font-semibold border-b border-slate-200">
                    <tr>
                      <th class="px-4 py-2.5">同步条目</th>
                      <th class="px-4 py-2.5">状态</th>
                      <th class="px-4 py-2.5">本地目标路径</th>
                      <th class="px-4 py-2.5">说明</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-slate-200/80 bg-white">
                    <tr v-for="item in lastResult.items" :key="`${item.itemId}-${item.targetPath}`" class="hover:bg-slate-50/40 transition">
                      <td class="px-4 py-3 font-mono text-[10px] text-slate-700 font-medium select-all">{{ item.itemId }}</td>
                      <td class="px-4 py-3">
                        <span class="rounded-full px-2 py-0.5 font-bold"
                          :class="item.status === 'applied' || item.status === 'uploaded' || item.status === 'success'
                            ? 'bg-emerald-50 text-emerald-700 border border-emerald-100'
                            : item.status === 'skipped'
                            ? 'bg-slate-100 text-slate-600 border border-slate-200'
                            : 'bg-rose-50 text-rose-700 border border-rose-100'"
                        >
                          {{ item.status }}
                        </span>
                      </td>
                      <td class="px-4 py-3 font-mono text-[10px] text-slate-500 select-all">{{ item.targetPath }}</td>
                      <td class="px-4 py-3 text-slate-600 whitespace-normal break-all">{{ item.reason || '-' }}</td>
                    </tr>
                    <tr v-if="lastResult.items.length === 0">
                      <td colspan="4" class="px-4 py-8 text-center text-slate-400">本次操作没有涉及明细条目变化。</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <!-- Advanced Sync Component -->
        <SyncCenter v-else-if="currentTab === 'sync'" />

        <!-- Profile Management Component -->
        <ProfileManager v-else-if="currentTab === 'profiles'" />

        <!-- Security settings Component -->
        <SettingsPanel v-else />
      </div>
    </main>

    <!-- Global Conflict resolver dialog for Quick sync -->
    <ConflictResolverDialog
      :visible="conflictVisible"
      :conflicts="conflicts"
      :default-overwrite-all="true"
      :submitting="activity === 'applying_snapshot'"
      @close="closeConflictDialog"
      @confirm="submitConflictDecision"
    />
  </div>
</template>
