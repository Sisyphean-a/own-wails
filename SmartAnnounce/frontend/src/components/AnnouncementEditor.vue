<script setup>
import { computed } from 'vue'
import VoiceCard from './VoiceCard.vue'

const props = defineProps({
  announcementText: {
    type: String,
    required: true,
  },
  selectedVoiceId: {
    type: String,
    required: true,
  },
  speechRate: {
    type: Number,
    required: true,
  },
  voiceVolume: {
    type: Number,
    required: true,
  },
  textLength: {
    type: Number,
    required: true,
  },
  voicePresets: {
    type: Array,
    required: true,
  },
  isGenerating: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits([
  'clear',
  'generate',
  'update:announcementText',
  'update:selectedVoiceId',
  'update:speechRate',
  'update:voiceVolume',
])

const announcementProxy = computed({
  get: () => props.announcementText,
  set: (value) => emit('update:announcementText', value),
})

const speechRateProxy = computed({
  get: () => props.speechRate,
  set: (value) => emit('update:speechRate', value),
})

const voiceVolumeProxy = computed({
  get: () => props.voiceVolume,
  set: (value) => emit('update:voiceVolume', value),
})
</script>

<template>
  <section class="panel editor-panel">
    <div class="editor-layout">
      <div class="editor-compose">
        <div class="editor-toolbar">
          <span class="text-counter">{{ textLength }} / 2000</span>
          <button class="ghost-button" type="button" @click="emit('clear')">清空</button>
        </div>

        <label class="sr-only" for="announcement-text">播报文案</label>
        <textarea
          id="announcement-text"
          v-model="announcementProxy"
          class="announcement-input"
          maxlength="2000"
        />
      </div>

      <aside class="editor-config">
        <div class="voice-grid">
          <VoiceCard
            v-for="voice in voicePresets"
            :key="voice.id"
            :selected="voice.id === selectedVoiceId"
            :voice="voice"
            @select="emit('update:selectedVoiceId', voice.id)"
          />
        </div>

        <div class="config-group">
          <div class="slider-box">
            <div class="slider-head">
              <label for="speed-range">语速</label>
              <span>{{ speechRate.toFixed(2) }}x</span>
            </div>
            <input
              id="speed-range"
              v-model.number="speechRateProxy"
              class="slider"
              type="range"
              min="0.5"
              max="2"
              step="0.05"
            />
          </div>
          <div class="slider-box">
            <div class="slider-head">
              <label for="voice-volume-range">音量</label>
              <span>{{ Math.round(voiceVolume * 100) }}%</span>
            </div>
            <input
              id="voice-volume-range"
              v-model.number="voiceVolumeProxy"
              class="slider"
              type="range"
              min="0"
              max="1"
              step="0.05"
            />
          </div>
        </div>

        <button class="primary-button" type="button" :disabled="isGenerating" @click="emit('generate')">
          <span v-if="isGenerating">生成中</span>
          <span v-else>生成</span>
        </button>
      </aside>
    </div>
  </section>
</template>

<style scoped src="./AnnouncementEditor.css"></style>
