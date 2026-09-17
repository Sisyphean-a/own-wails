<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { validateMasterPassword } from '../lib/backend'
import { useSettingsStore } from '../composables/useSettingsStore'

const store = useSettingsStore()
const form = reactive({
  token: '',
  masterPassword: '',
})
const status = ref('')
const testing = ref(false)
const showToken = ref(false)
const showMasterPassword = ref(false)
const passwordValidation = ref<'idle' | 'pending' | 'missing_token' | 'checking' | 'valid' | 'invalid' | 'no_snapshot' | 'error'>('idle')
let validationRequest = 0

onMounted(async () => {
  try {
    await store.ensureLoaded()
    form.token = store.state.value?.token ?? ''
    form.masterPassword = store.state.value?.masterPassword ?? ''
  } catch (error) {
    status.value = `加载设置失败: ${String(error)}`
  }
})

watch(
  () => [form.token, form.masterPassword],
  () => {
    validationRequest++
    if (!form.masterPassword.trim()) {
      passwordValidation.value = 'idle'
      return
    }
    passwordValidation.value = form.token.trim() ? 'pending' : 'missing_token'
  },
)

async function validatePasswordOnBlur(): Promise<void> {
  const request = ++validationRequest
  const token = form.token
  const password = form.masterPassword
  if (!password.trim()) {
    passwordValidation.value = 'idle'
    return
  }
  if (!token.trim()) {
    passwordValidation.value = 'missing_token'
    return
  }

  passwordValidation.value = 'checking'
  try {
    const result = await validateMasterPassword(token, password)
    if (request !== validationRequest) {
      return
    }
    passwordValidation.value = !result.hasSnapshot ? 'no_snapshot' : result.valid ? 'valid' : 'invalid'
  } catch {
    if (request === validationRequest) {
      passwordValidation.value = 'error'
    }
  }
}

async function persistSettings(): Promise<void> {
  if (passwordValidation.value === 'pending' || passwordValidation.value === 'checking') {
    status.value = '请先离开主密码输入框，完成校验后再保存'
    return
  }
  if (passwordValidation.value === 'invalid' || passwordValidation.value === 'error') {
    status.value = '主密码未通过云端校验，未保存'
    return
  }
  try {
    await store.saveCredentials(form.token, form.masterPassword)
    status.value = '设置保存成功'
  } catch (error) {
    status.value = `保存失败: ${String(error)}`
  }
}

async function testConnection(): Promise<void> {
  if (!form.token) {
    status.value = '请先输入 GitHub Token 进行测试'
    return
  }
  testing.value = true
  status.value = '正在测试 GitHub API 连接...'
  try {
    const response = await fetch('https://api.github.com/user', {
      headers: {
        Authorization: `token ${form.token}`,
        'Accept': 'application/vnd.github.v3+json'
      }
    })
    if (response.ok) {
      const data = await response.json()
      status.value = `✅ 连接成功！GitHub 用户名: ${data.login}`
    } else {
      const errData = await response.json().catch(() => ({}))
      status.value = `❌ 连接失败 (HTTP ${response.status}): ${errData.message || response.statusText}`
    }
  } catch (error) {
    status.value = `❌ 网络连接异常: ${String(error)}`
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <section class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm text-left">
    <div class="mb-5 flex items-center justify-between border-b border-slate-100 pb-4">
      <div>
        <h2 class="text-lg font-bold text-slate-900">安全设置</h2>
        <p class="mt-1 text-xs text-slate-500">同步所需的 GitHub 凭证和本地加解密主密码。</p>
      </div>
      <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-600">云存储 & 加密</span>
    </div>

    <div class="space-y-5">
      <div class="space-y-1">
        <label class="block text-sm font-semibold text-slate-700">GitHub Personal Access Token (PAT)</label>
        <div class="relative">
          <input
            v-model="form.token"
            :type="showToken ? 'text' : 'password'"
            placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
            class="w-full rounded-lg border border-slate-300 pl-3 pr-10 py-2.5 text-sm text-slate-900 shadow-sm outline-none transition focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
          >
          <button 
            type="button" 
            class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 focus:outline-none"
            @click="showToken = !showToken"
          >
            <span v-if="showToken">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88" />
              </svg>
            </span>
            <span v-else>
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
              </svg>
            </span>
          </button>
        </div>
        <p class="text-[11px] text-slate-400">用于访问你的 GitHub Gist API，需要勾选 `gist` 权限。</p>
      </div>

      <div class="space-y-1">
        <label class="block text-sm font-semibold text-slate-700">主密码 (Master Password)</label>
        <div class="relative">
          <input
            v-model="form.masterPassword"
            :type="showMasterPassword ? 'text' : 'password'"
            placeholder="请输入本地数据加解密主密码"
            class="w-full rounded-lg border border-slate-300 pl-3 pr-10 py-2.5 text-sm text-slate-900 shadow-sm outline-none transition focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            @blur="validatePasswordOnBlur"
          >
          <button 
            type="button" 
            class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 focus:outline-none"
            @mousedown.prevent
            @click="showMasterPassword = !showMasterPassword"
          >
            <span v-if="showMasterPassword">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88" />
              </svg>
            </span>
            <span v-else>
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
              </svg>
            </span>
          </button>
        </div>
        <p class="text-[11px] text-slate-400">用于加密本地文件后上传。在其他设备恢复时，必须输入相同的主密码。</p>
        <p v-if="passwordValidation === 'pending'" class="text-[11px] text-slate-500">离开输入框后校验主密码</p>
        <p v-else-if="passwordValidation === 'missing_token'" class="text-[11px] text-amber-700">输入 GitHub Token 后可校验主密码</p>
        <p v-else-if="passwordValidation === 'checking'" class="inline-flex items-center gap-2 text-[11px] text-slate-500">
          <span class="h-3 w-3 animate-spin rounded-full border-2 border-slate-200 border-t-indigo-600" />
          正在校验云端快照
        </p>
        <p v-else-if="passwordValidation === 'valid'" class="text-[11px] font-medium text-emerald-700">主密码可解开云端最新快照</p>
        <p v-else-if="passwordValidation === 'invalid'" class="text-[11px] font-medium text-rose-700">主密码无法解开云端最新快照</p>
        <p v-else-if="passwordValidation === 'no_snapshot'" class="text-[11px] text-slate-500">云端暂无快照，首次上传后可校验</p>
        <p v-else-if="passwordValidation === 'error'" class="text-[11px] text-rose-700">无法校验主密码，请检查 Token 和网络连接</p>
      </div>

      <div class="flex flex-wrap items-center gap-3 pt-2">
        <button
          type="button"
          class="rounded-lg bg-indigo-600 hover:bg-indigo-700 px-5 py-2.5 text-sm font-semibold text-white transition shadow-sm disabled:cursor-not-allowed disabled:bg-slate-400"
          :disabled="passwordValidation === 'pending' || passwordValidation === 'checking' || passwordValidation === 'invalid' || passwordValidation === 'error'"
          @click="persistSettings"
        >
          保存设置
        </button>
        
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-5 py-2.5 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 shadow-sm"
          :disabled="testing"
          @click="testConnection"
        >
          <span v-if="testing" class="h-4 w-4 animate-spin rounded-full border-2 border-slate-200 border-t-indigo-600" />
          <span>测试 GitHub 连接</span>
        </button>
      </div>

      <div v-if="status" class="mt-4 rounded-xl bg-slate-50 p-4 border border-slate-200/80">
        <p class="text-sm font-medium text-slate-700 whitespace-pre-wrap">{{ status }}</p>
      </div>
    </div>
  </section>
</template>
