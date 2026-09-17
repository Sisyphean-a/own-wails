<script setup>
import { computed } from 'vue'

const props = defineProps({
  audioReady: {
    type: Boolean,
    default: false,
  },
  isPlaying: {
    type: Boolean,
    default: false,
  },
  isExporting: {
    type: Boolean,
    default: false,
  },
  loopEnabled: {
    type: Boolean,
    default: false,
  },
  playbackVolume: {
    type: Number,
    required: true,
  },
  errorMessage: {
    type: String,
    default: '',
  },
  fileName: {
    type: String,
    default: '',
  },
  progressPercent: {
    type: Number,
    required: true,
  },
  currentTimeLabel: {
    type: String,
    required: true,
  },
  durationLabel: {
    type: String,
    required: true,
  },
})

const emit = defineEmits(['export-audio', 'seek', 'toggle-playback', 'update:loopEnabled', 'update:playbackVolume'])

const loopEnabledProxy = computed({
  get: () => props.loopEnabled,
  set: (value) => emit('update:loopEnabled', value),
})

const playbackVolumeProxy = computed({
  get: () => props.playbackVolume,
  set: (value) => emit('update:playbackVolume', value),
})

const statusClass = computed(() => {
  if (props.errorMessage) {
    return 'danger'
  }

  if (props.audioReady) {
    return 'success'
  }

  return 'idle'
})

const statusText = computed(() => {
  if (props.errorMessage) {
    return props.errorMessage
  }

  return props.fileName || '未生成'
})

function handleSeek(event) {
  emit('seek', Number(event.target.value))
}
</script>

<template>
  <section class="panel player-panel">
    <div class="transport-row">
      <button class="play-toggle" type="button" :disabled="!audioReady" @click="emit('toggle-playback')">
        {{ isPlaying ? '暂停' : '播放' }}
      </button>

      <div class="timeline-block">
        <input
          class="slider"
          type="range"
          min="0"
          max="100"
          step="0.1"
          :value="progressPercent"
          :disabled="!audioReady"
          @input="handleSeek"
        />
        <div class="timeline-meta">
          <span>{{ currentTimeLabel }}</span>
          <span>{{ durationLabel }}</span>
        </div>
      </div>

      <button class="export-button" type="button" :disabled="!audioReady || isExporting" @click="emit('export-audio')">
        <span v-if="isExporting">导出中</span>
        <span v-else>导出</span>
      </button>
    </div>

    <div class="player-controls">
      <div class="status-chip" :class="statusClass">
        {{ statusText }}
      </div>

      <label class="switch-row" for="loop-switch">
        <span>循环</span>
        <input id="loop-switch" v-model="loopEnabledProxy" type="checkbox" />
      </label>

      <div class="slider-row">
        <label for="volume-range">音量</label>
        <input
          id="volume-range"
          v-model.number="playbackVolumeProxy"
          class="slider"
          type="range"
          min="0"
          max="1"
          step="0.05"
        />
        <span>{{ Math.round(playbackVolume * 100) }}%</span>
      </div>
    </div>
  </section>
</template>

<style scoped src="./PreviewPlayer.css"></style>
