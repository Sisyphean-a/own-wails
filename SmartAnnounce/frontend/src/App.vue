<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ExportBroadcast, GenerateBroadcast } from '../wailsjs/go/main/App'
import AnnouncementEditor from './components/AnnouncementEditor.vue'
import PreviewPlayer from './components/PreviewPlayer.vue'

const voicePresets = [
  { id: 'female-shaonv', name: '中文女声', style: '温柔亲切', accent: 'rose' },
  { id: 'male-qn-jingying', name: '中文男声', style: '稳重自然', accent: 'blue' },
  { id: 'female-yujie', name: '活力女声', style: '活泼明亮', accent: 'violet' },
  { id: 'female-chengshu', name: '标准女声', style: '清晰标准', accent: 'teal' },
]

const sampleAnnouncement = `亲爱的顾客朋友们，大家好！
欢迎光临本超市！今日特价商品多多：
1. 精选苹果 5.99元/斤
2. 鲜鸡蛋 3.99元/盒
3. 优质大米 29.9元/袋
更多优惠，尽在本超市。感谢您的光临，祝您购物愉快！`

const audioRef = ref(null)
const state = reactive({
  announcementText: sampleAnnouncement,
  selectedVoiceId: voicePresets[0].id,
  speechRate: 1,
  voiceVolume: 1,
  playbackVolume: 0.8,
  loopEnabled: true,
  isGenerating: false,
  isExporting: false,
  isPlaying: false,
  errorMessage: '',
  audioDataUri: '',
  fileName: '',
  filePath: '',
  currentTime: 0,
  duration: 0,
})

const textLength = computed(() => state.announcementText.trim().length)
const progressPercent = computed(() => (state.duration === 0 ? 0 : Math.min((state.currentTime / state.duration) * 100, 100)))

async function generateAudio() {
  state.isGenerating = true
  state.errorMessage = ''

  try {
    const result = await GenerateBroadcast({
      text: state.announcementText,
      voiceId: state.selectedVoiceId,
      speed: Number(state.speechRate.toFixed(2)),
      volume: Number(state.voiceVolume.toFixed(2)),
    })

    state.audioDataUri = result.audioDataUri
    state.fileName = result.fileName
    state.filePath = result.filePath
    state.currentTime = 0
    state.duration = 0
    state.isPlaying = false

    await nextTick()
    resetAudioElement()
  } catch (error) {
    state.errorMessage = readErrorMessage(error)
  } finally {
    state.isGenerating = false
  }
}

async function togglePlayback() {
  const audio = audioRef.value
  if (!audio || !state.audioDataUri) {
    state.errorMessage = '请先生成语音文件'
    return
  }

  state.errorMessage = ''
  if (state.isPlaying) {
    audio.pause()
    state.isPlaying = false
    return
  }

  try {
    await audio.play()
    state.isPlaying = true
  } catch (error) {
    state.errorMessage = readErrorMessage(error)
  }
}

async function exportAudio() {
  if (!state.filePath) {
    state.errorMessage = '请先生成语音文件'
    return
  }

  state.isExporting = true
  state.errorMessage = ''

  try {
    const result = await ExportBroadcast({
      sourcePath: state.filePath,
      fileName: state.fileName,
    })

    if (result.cancelled) {
      return
    }
  } catch (error) {
    state.errorMessage = readErrorMessage(error)
  } finally {
    state.isExporting = false
  }
}

function clearAnnouncement() {
  state.announcementText = ''
}

function handleTimeUpdate() {
  const audio = audioRef.value
  if (!audio) {
    return
  }

  state.currentTime = audio.currentTime
  state.duration = Number.isFinite(audio.duration) ? audio.duration : 0
}

function handleLoadedMetadata() {
  const audio = audioRef.value
  if (audio) {
    state.duration = Number.isFinite(audio.duration) ? audio.duration : 0
  }
}

function handlePlaybackEnded() {
  if (!state.loopEnabled) {
    state.isPlaying = false
    state.currentTime = 0
  }
}

function seekPlayback(nextProgress) {
  const audio = audioRef.value
  if (!audio || state.duration === 0) {
    return
  }

  audio.currentTime = (nextProgress / 100) * state.duration
  state.currentTime = audio.currentTime
}

function resetAudioElement() {
  const audio = audioRef.value
  if (!audio) {
    return
  }

  audio.pause()
  audio.currentTime = 0
  audio.load()
  audio.loop = state.loopEnabled
  audio.volume = state.playbackVolume
}

function readErrorMessage(error) {
  if (typeof error === 'string') {
    return error
  }

  if (error && typeof error === 'object' && 'message' in error) {
    return String(error.message)
  }

  return '出现未知错误，请查看后端日志。'
}

function formatTime(seconds) {
  const safeSeconds = Number.isFinite(seconds) ? Math.max(seconds, 0) : 0
  const minutes = Math.floor(safeSeconds / 60)
  const remainSeconds = Math.floor(safeSeconds % 60)
  return `${String(minutes).padStart(2, '0')}:${String(remainSeconds).padStart(2, '0')}`
}

watch(() => state.playbackVolume, (value) => {
  if (audioRef.value) {
    audioRef.value.volume = value
  }
})

watch(() => state.loopEnabled, (value) => {
  if (audioRef.value) {
    audioRef.value.loop = value
  }
})
</script>

<template>
  <div class="app-shell">
    <main class="workspace">
      <section class="workspace-editor">
        <AnnouncementEditor
          v-model:announcement-text="state.announcementText"
          v-model:selected-voice-id="state.selectedVoiceId"
          v-model:speech-rate="state.speechRate"
          v-model:voice-volume="state.voiceVolume"
          :is-generating="state.isGenerating"
          :text-length="textLength"
          :voice-presets="voicePresets"
          @clear="clearAnnouncement"
          @generate="generateAudio"
        />
      </section>

      <section class="workspace-player">
        <PreviewPlayer
          v-model:loop-enabled="state.loopEnabled"
          v-model:playback-volume="state.playbackVolume"
          :audio-ready="Boolean(state.audioDataUri)"
          :current-time-label="formatTime(state.currentTime)"
          :duration-label="formatTime(state.duration)"
          :error-message="state.errorMessage"
          :file-name="state.fileName"
          :is-exporting="state.isExporting"
          :is-playing="state.isPlaying"
          :progress-percent="progressPercent"
          @export-audio="exportAudio"
          @seek="seekPlayback"
          @toggle-playback="togglePlayback"
        />
      </section>
    </main>

    <audio
      ref="audioRef"
      :src="state.audioDataUri"
      preload="metadata"
      @ended="handlePlaybackEnded"
      @loadedmetadata="handleLoadedMetadata"
      @pause="state.isPlaying = false"
      @play="state.isPlaying = true"
      @timeupdate="handleTimeUpdate"
    />
  </div>
</template>
