<script setup>
defineProps({
  voice: {
    type: Object,
    required: true,
  },
  selected: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['select'])
</script>

<template>
  <button class="voice-card" :class="[voice.accent, { selected }]" type="button" @click="$emit('select')">
    <div class="voice-main">
      <span class="voice-avatar"></span>
      <div class="voice-copy">
        <strong>{{ voice.name }}</strong>
        <span>{{ voice.style }}</span>
      </div>
    </div>
    <span class="voice-state" :class="{ selected }" aria-hidden="true"></span>
  </button>
</template>

<style scoped>
.voice-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.04);
  text-align: left;
  color: inherit;
  cursor: pointer;
  transition: transform 160ms ease, border-color 160ms ease, background 160ms ease;
}

.voice-card:hover {
  transform: translateY(-1px);
}

.voice-card.selected {
  border-color: rgba(83, 132, 255, 0.9);
  box-shadow: inset 0 0 0 1px rgba(83, 132, 255, 0.6);
  background: rgba(34, 58, 116, 0.5);
}

.voice-card.rose .voice-avatar {
  background: linear-gradient(180deg, #ffc4d0, #ff8fa9);
}

.voice-card.blue .voice-avatar {
  background: linear-gradient(180deg, #91bbff, #4568ff);
}

.voice-card.violet .voice-avatar {
  background: linear-gradient(180deg, #e6b5ff, #945dff);
}

.voice-card.teal .voice-avatar {
  background: linear-gradient(180deg, #9ad9d5, #2d8e9d);
}

.voice-avatar {
  width: 46px;
  height: 46px;
  border-radius: 14px;
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.2);
}

.voice-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.voice-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.voice-card strong {
  font-size: 16px;
  line-height: 1.3;
}

.voice-card span {
  color: var(--text-muted);
  font-size: 13px;
}

.voice-state {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.08);
  flex: none;
}

.voice-state.selected {
  border-color: rgba(83, 132, 255, 0.9);
  background: #77a0ff;
  box-shadow: 0 0 0 4px rgba(83, 132, 255, 0.16);
}
</style>
