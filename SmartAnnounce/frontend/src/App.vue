<script setup>
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { ExportBroadcast, GenerateBroadcast, GetMinimaxSettings, SaveMinimaxSettings } from '../wailsjs/go/main/App'
import HistoryDialog from './HistoryDialog.vue'

const voices = [
  { id: 'female-chengshu', name: '成熟女性音色', description: '日常促销' },
  { id: 'Chinese (Mandarin)_News_Anchor', name: '新闻女声', description: '正式通知' },
  { id: 'male-qn-jingying', name: '精英青年音色', description: '男声播报' },
  { id: 'Chinese (Mandarin)_Radio_Host', name: '电台男主播', description: '男声主持' },
  { id: 'female-yujie', name: '御姐音色', description: '鲜明风格' },
  { id: 'female-tianmei', name: '甜美女性音色', description: '柔和风格' },
  { id: 'Chinese (Mandarin)_Reliable_Executive', name: '沉稳高管', description: '沉稳男声' },
  { id: 'female-shaonv', name: '少女音色', description: '轻快风格' },
]

const example = `亲爱的顾客朋友们，大家好！\n欢迎光临本超市。今日精选苹果每斤 5.99 元，鲜鸡蛋每盒 3.99 元。更多优惠，欢迎到店选购。祝您购物愉快！`
const audioRef = ref(null)
const editorRef = ref(null)
const settingsButtonRef = ref(null)
const historyButtonRef = ref(null)
const historyOpen = ref(false)
const apiKeyRef = ref(null)
const settings = reactive({
  open: false, loading: true, saving: false, error: '',
  loaded: false, hasApiKey: false, model: 'speech-2.8-hd',
  proxyMode: 'direct', proxyUrl: '',
  draftKey: '', draftModel: 'speech-2.8-hd', draftProxyMode: 'direct', draftProxyUrl: '',
})
const state = reactive({
  text: '', voiceId: voices[0].id, speed: 1, volume: 1,
  playbackVolume: 0.8, loop: false,
  generating: false, exporting: false, playing: false,
  error: '', notice: '', audioUri: '', fileName: '', filePath: '',
  generatedInput: '', currentTime: 0, duration: 0,
})

const length = computed(() => [...state.text.trim()].length)
const currentInput = computed(() => JSON.stringify([state.text.trim(), state.voiceId, state.speed, state.volume, settings.model]))
const outOfDate = computed(() => Boolean(state.audioUri) && currentInput.value !== state.generatedInput)
const canGenerate = computed(() => length.value > 0 && length.value <= 2000 && settings.loaded && settings.hasApiKey && !state.generating)
const progress = computed(() => state.duration > 0 ? Math.min(100, state.currentTime / state.duration * 100) : 0)

function message(error) {
  if (typeof error === 'string') return error
  return error && typeof error.message === 'string' ? error.message : '发生未知错误，请检查后端日志。'
}

async function loadSettings() {
  settings.loading = true
  settings.error = ''
  try {
    const result = await GetMinimaxSettings()
    settings.model = result.model
    settings.hasApiKey = result.hasApiKey
    settings.proxyMode = result.proxyMode
    settings.proxyUrl = result.proxyUrl
    settings.loaded = true
    settings.draftModel = result.model
    settings.draftProxyMode = result.proxyMode
    settings.draftProxyUrl = result.proxyUrl
    if (state.error.startsWith('读取 Minimax 配置失败：')) state.error = ''
    return result
  } catch (error) {
    settings.loaded = false
    settings.error = message(error)
    state.error = `读取 Minimax 配置失败：${settings.error}`
    return null
  } finally {
    settings.loading = false
  }
}

async function openSettings() {
  settings.open = true
  settings.draftKey = ''
  await loadSettings()
  await nextTick()
  apiKeyRef.value?.focus()
}

function closeHistory() {
  historyOpen.value = false
  nextTick(() => historyButtonRef.value?.focus())
}

function applyHistoryText(text) {
  state.text = text
  historyOpen.value = false
  nextTick(() => {
    state.notice = '已应用历史文案，请按需重新生成播报'
    editorRef.value?.focus()
  })
}

function closeSettings() {
  if (settings.saving) return
  settings.open = false
  settings.draftKey = ''
  settings.error = ''
  nextTick(() => settingsButtonRef.value?.focus())
}

async function saveSettings() {
  if (settings.saving || settings.loading) return
  settings.saving = true
  settings.error = ''
  try {
    const result = await SaveMinimaxSettings({
      apiKey: settings.draftKey, model: settings.draftModel,
      proxyMode: settings.draftProxyMode,
      proxyUrl: settings.draftProxyMode === 'custom' ? settings.draftProxyUrl : '',
    })
    settings.model = result.model
    settings.hasApiKey = result.hasApiKey
    settings.proxyMode = result.proxyMode
    settings.proxyUrl = result.proxyUrl
    settings.loaded = true
    settings.open = false
    settings.draftKey = ''
    await nextTick()
    settingsButtonRef.value?.focus()
    state.error = ''
    state.notice = 'Minimax 配置已保存'
  } catch (error) {
    settings.error = message(error)
  } finally {
    settings.saving = false
  }
}

function trapSettingsFocus(event) {
  const dialog = event.currentTarget.querySelector('.settings-dialog')
  const controls = [...dialog.querySelectorAll('button:not(:disabled), input:not(:disabled)')]
  if (event.shiftKey && document.activeElement === controls[0]) {
    event.preventDefault()
    controls.at(-1)?.focus()
  } else if (!event.shiftKey && document.activeElement === controls.at(-1)) {
    event.preventDefault()
    controls[0]?.focus()
  }
}

onMounted(async () => {
  const result = await loadSettings()
  if (result && !result.hasApiKey) {
    settings.open = true
    await nextTick()
    apiKeyRef.value?.focus()
  }
})

function formatTime(seconds) {
  const safe = Number.isFinite(seconds) ? Math.max(0, Math.floor(seconds)) : 0
  return `${String(Math.floor(safe / 60)).padStart(2, '0')}:${String(safe % 60).padStart(2, '0')}`
}

async function generate() {
  if (state.generating) return
  if (!canGenerate.value) {
    state.error = !settings.loaded ? '配置尚未就绪，请检查 Minimax 设置' : !settings.hasApiKey ? '请先在设置中配置 Minimax API Key' : length.value > 2000 ? '文案不能超过 2000 个字符' : '请先输入播报文案'
    editorRef.value?.focus()
    return
  }
  const input = currentInput.value
  state.generating = true
  state.error = ''
  state.notice = ''
  try {
    const result = await GenerateBroadcast({
      text: state.text, voiceId: state.voiceId,
      speed: Number(state.speed.toFixed(2)), volume: Number(state.volume.toFixed(2)),
    })
    audioRef.value?.pause()
    state.audioUri = result.audioDataUri
    state.fileName = result.fileName
    state.filePath = result.filePath
    state.generatedInput = input
    state.playing = false
    state.currentTime = 0
    state.duration = 0
    await nextTick()
    audioRef.value?.load()
  } catch (error) {
    state.error = message(error)
  } finally {
    state.generating = false
  }
}

async function togglePlayback() {
  const audio = audioRef.value
  if (!audio || !state.audioUri) return
  state.error = ''
  if (!audio.paused) {
    audio.pause()
    return
  }
  try {
    await audio.play()
  } catch (error) {
    state.error = message(error)
  }
}

async function exportAudio() {
  if (!state.filePath || state.exporting) return
  state.exporting = true
  state.error = ''
  state.notice = ''
  try {
    const result = await ExportBroadcast({ sourcePath: state.filePath, fileName: state.fileName })
    if (!result.cancelled) state.notice = `已导出至 ${result.filePath}`
  } catch (error) {
    state.error = message(error)
  } finally {
    state.exporting = false
  }
}

function seek(event) {
  if (!audioRef.value || !state.duration) return
  audioRef.value.currentTime = Number(event.target.value) / 100 * state.duration
  state.currentTime = audioRef.value.currentTime
}

function updateTime() {
  const audio = audioRef.value
  if (!audio) return
  state.currentTime = audio.currentTime
  state.duration = Number.isFinite(audio.duration) ? audio.duration : 0
}

watch(currentInput, () => { state.notice = '' })
watch(() => state.loop, value => { if (audioRef.value) audioRef.value.loop = value })
watch(() => state.playbackVolume, value => { if (audioRef.value) audioRef.value.volume = value })
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand"><span class="brand-icon" aria-hidden="true">◖))</span><span>SmartAnnounce</span><span class="brand-divider"></span><span class="brand-subtitle">播报工作台</span></div>
      <div class="topbar-actions"><button ref="historyButtonRef" class="settings-trigger" type="button" @click="historyOpen = true">文案历史</button><button ref="settingsButtonRef" class="settings-trigger" type="button" :aria-label="settings.hasApiKey ? 'Minimax 设置' : 'Minimax 设置，尚未配置 API Key'" @click="openSettings"><span aria-hidden="true">⚙</span> 设置<span v-if="settings.loaded && !settings.hasApiKey" class="settings-indicator">未配置</span></button></div>
    </header>

    <main class="workspace">
      <section class="compose-panel" aria-labelledby="compose-title">
        <div class="section-heading"><h2 id="compose-title">播报文案</h2></div>
        <div class="editor-frame" :class="{ invalid: length > 2000 }">
          <textarea ref="editorRef" v-model="state.text" aria-label="播报文案" placeholder="在这里写下要播报的内容…" spellcheck="false"></textarea>
          <div class="editor-footer"><div class="editor-actions"><button class="text-action" type="button" :disabled="state.generating" @click="state.text = example; editorRef?.focus()">填入示例</button><span class="action-divider"></span><button class="text-action" type="button" :disabled="!state.text || state.generating" @click="state.text = ''; editorRef?.focus()">清空</button></div><span class="counter" :class="{ invalid: length > 2000 }">{{ length }} <span>/ 2000 字</span></span></div>
        </div>
      </section>

      <aside class="settings-panel" aria-label="声音设置">
        <div class="settings-content">
        <div class="section-heading"><h2>选择音色</h2></div>
        <div class="voice-list" role="radiogroup" aria-label="选择音色">
          <button v-for="voice in voices" :key="voice.id" type="button" class="voice-option" :class="{ selected: state.voiceId === voice.id }" role="radio" :aria-checked="state.voiceId === voice.id" :disabled="state.generating" @click="state.voiceId = voice.id">
            <span class="voice-copy"><strong>{{ voice.name }}</strong><small>{{ voice.description }}</small></span><span class="radio-indicator" aria-hidden="true"></span>
          </button>
        </div>
        <div class="settings-divider"></div>
        <div class="section-heading parameter-heading"><h2>声音参数</h2></div>
        <div class="parameter"><div class="parameter-label"><label for="speed">语速 <small>朗读快慢</small></label><output for="speed">{{ state.speed.toFixed(2) }} ×</output></div><input id="speed" v-model.number="state.speed" type="range" min="0.5" max="2" step="0.05" :disabled="state.generating"><div class="range-ends"><span>慢 0.5×</span><span>快 2.0×</span></div></div>
        <div class="parameter"><div class="parameter-label"><label for="voice-volume">生成音量 <small>写入音频</small></label><output for="voice-volume">{{ Math.round(state.volume * 100) }}%</output></div><input id="voice-volume" v-model.number="state.volume" type="range" min="0" max="1" step="0.05" :disabled="state.generating"><div class="range-ends"><span>静音</span><span>最大</span></div></div>
        </div>
        <div class="generate-area"><button class="generate-button" type="button" :disabled="!canGenerate" @click="generate"><span v-if="state.generating" class="spinner" aria-hidden="true"></span><span v-else aria-hidden="true">✦</span> {{ state.generating ? '正在生成语音…' : '生成播报' }} <span v-if="!state.generating" class="button-arrow" aria-hidden="true">↗</span></button><span v-if="!settings.loading && !settings.hasApiKey" class="generate-hint">请先在设置中填写 API Key</span><span v-else-if="length > 2000" class="generate-hint">文案超过 2000 字，请精简后再生成</span></div>
      </aside>
    </main>

    <section class="player-panel" aria-label="试听与导出">
      <div class="player-body"><button class="play-button" type="button" :disabled="!state.audioUri" :aria-label="state.playing ? '暂停试听' : '播放试听'" @click="togglePlayback">{{ state.playing ? 'Ⅱ' : '▶' }}</button><div class="track"><div class="track-name" :title="state.fileName || ''"><strong>{{ state.fileName || '尚未生成音频' }}</strong><span v-if="outOfDate" class="player-state stale"><span class="state-dot"></span>上一版本 · 请重新生成</span></div><div class="timeline"><span>{{ formatTime(state.currentTime) }}</span><input type="range" min="0" max="100" step="0.1" :value="progress" :disabled="!state.audioUri || !state.duration" aria-label="播放进度" :style="{ '--progress': `${progress}%` }" @input="seek"><span>{{ formatTime(state.duration) }}</span></div></div><div class="player-options"><label class="loop-option"><input v-model="state.loop" type="checkbox"><span>循环</span></label><div class="playback-volume"><label for="playback-volume">试听音量</label><input id="playback-volume" v-model.number="state.playbackVolume" type="range" min="0" max="1" step="0.05"><span>{{ Math.round(state.playbackVolume * 100) }}%</span></div></div><button class="export-button" type="button" :disabled="!state.audioUri || state.exporting" @click="exportAudio">{{ state.exporting ? '导出中…' : '导出 MP3' }} <span aria-hidden="true">↗</span></button></div>
    </section>
    <div v-if="state.error || state.notice" class="statusbar" role="status" aria-live="polite" :class="{ 'status-error': state.error, 'status-success': !state.error }">{{ state.error || state.notice }}</div>

    <HistoryDialog v-if="historyOpen" :current-text="state.text" :generating="state.generating" @close="closeHistory" @apply="applyHistoryText" />

    <div v-if="settings.open" class="dialog-backdrop" @mousedown.self="closeSettings" @keydown.esc="closeSettings" @keydown.tab="trapSettingsFocus">
      <section class="settings-dialog" role="dialog" aria-modal="true" aria-labelledby="settings-title">
        <header class="dialog-header"><h2 id="settings-title">Minimax 设置</h2><button type="button" class="dialog-close" aria-label="关闭设置" :disabled="settings.saving" @click="closeSettings">×</button></header>
        <form class="dialog-content" @submit.prevent="saveSettings">
          <label for="minimax-key">API Key</label>
          <input id="minimax-key" ref="apiKeyRef" v-model="settings.draftKey" type="password" autocomplete="off" spellcheck="false" :placeholder="settings.hasApiKey ? '已保存，留空保持不变' : '输入 Minimax API Key'" :disabled="settings.loading || settings.saving">
          <p v-if="settings.hasApiKey" class="dialog-help">已配置密钥。留空只修改模型，输入新密钥可替换。</p>
          <label for="minimax-model">TTS 模型</label>
          <input id="minimax-model" v-model="settings.draftModel" type="text" spellcheck="false" :disabled="settings.loading || settings.saving">
          <p class="dialog-help">默认 speech-2.8-hd。密钥由 Windows 当前用户加密保存在本机。</p>
          <fieldset class="proxy-settings">
            <legend>网络代理</legend>
            <label class="proxy-choice"><input v-model="settings.draftProxyMode" type="radio" value="direct" :disabled="settings.loading || settings.saving"> 直连（默认）</label>
            <label class="proxy-choice"><input v-model="settings.draftProxyMode" type="radio" value="custom" :disabled="settings.loading || settings.saving"> 自定义代理</label>
          </fieldset>
          <template v-if="settings.draftProxyMode === 'custom'">
            <label for="minimax-proxy">代理地址</label>
            <input id="minimax-proxy" v-model="settings.draftProxyUrl" type="url" placeholder="http://127.0.0.1:7890" spellcheck="false" required :disabled="settings.loading || settings.saving">
          </template>
          <p class="dialog-help">仅 TTS 请求使用所选代理；直连不跟随系统代理。不支持带账号密码的代理地址。</p>
          <p v-if="settings.error" class="dialog-error" role="alert">{{ settings.error }}</p>
          <div class="dialog-actions"><button type="button" class="dialog-cancel" :disabled="settings.saving" @click="closeSettings">取消</button><button type="submit" class="dialog-save" :disabled="!settings.loaded || settings.loading || settings.saving || !settings.draftModel.trim() || (!settings.hasApiKey && !settings.draftKey.trim()) || (settings.draftProxyMode === 'custom' && !settings.draftProxyUrl.trim())">{{ settings.saving ? '保存中…' : '保存配置' }}</button></div>
        </form>
      </section>
    </div>
    <audio ref="audioRef" :src="state.audioUri" :loop="state.loop" :volume="state.playbackVolume" preload="metadata" @loadedmetadata="updateTime" @timeupdate="updateTime" @play="state.playing = true" @pause="state.playing = false" @ended="state.playing = false" @error="state.audioUri && (state.error = '音频加载失败，请重新生成或检查文件。')" />
  </div>
</template>
