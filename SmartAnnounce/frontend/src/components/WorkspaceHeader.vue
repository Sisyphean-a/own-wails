<script setup>
const props = defineProps({
  selectedVoiceName: {
    type: String,
    required: true,
  },
  textLength: {
    type: Number,
    required: true,
  },
  outputDirectoryLabel: {
    type: String,
    required: true,
  },
  workspaceStatus: {
    type: String,
    required: true,
  },
  isGenerating: {
    type: Boolean,
    default: false,
  },
  hasGeneratedAudio: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <header class="app-header">
    <div class="hero-block">
      <div class="hero-copy">
        <p class="eyebrow">Smart Announce</p>
        <h1>超市播报台</h1>
        <p class="hero-summary">编辑文案、选择音色、生成语音，然后在右侧即时试听与导出。</p>
      </div>
      <div class="overview-grid">
        <article class="overview-card">
          <span>当前状态</span>
          <strong>{{ workspaceStatus }}</strong>
        </article>
        <article class="overview-card">
          <span>当前音色</span>
          <strong>{{ selectedVoiceName }}</strong>
        </article>
        <article class="overview-card">
          <span>文本长度</span>
          <strong>{{ textLength }} / 2000</strong>
        </article>
        <article class="overview-card">
          <span>输出目录</span>
          <strong>{{ outputDirectoryLabel }}</strong>
        </article>
      </div>
    </div>

    <div class="workflow-strip" aria-label="任务流程">
      <div class="workflow-item current">
        <span>1</span>
        <strong>编辑播报</strong>
      </div>
      <div class="workflow-item" :class="{ current: isGenerating || hasGeneratedAudio }">
        <span>2</span>
        <strong>生成语音文件</strong>
      </div>
      <div class="workflow-item" :class="{ current: hasGeneratedAudio }">
        <span>3</span>
        <strong>试听与导出</strong>
      </div>
    </div>
  </header>
</template>
