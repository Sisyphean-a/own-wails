<script setup lang="ts">
import { computed } from 'vue'
import type { DiffLine } from '../lib/backend'

const props = defineProps<{
  lines: DiffLine[]
  addedLines?: number
  removedLines?: number
  maxHeightClass?: string
}>()

const hasLines = computed(() => props.lines.length > 0)
const heightClass = computed(() => props.maxHeightClass ?? 'max-h-60')

function symbol(kind: DiffLine['kind']): string {
  if (kind === 'add') {
    return '+'
  }
  if (kind === 'delete') {
    return '-'
  }
  return ' '
}
</script>

<template>
  <div class="overflow-hidden rounded-lg border border-slate-700 bg-slate-900">
    <div class="flex items-center justify-between border-b border-slate-700/80 px-3 py-1.5 text-[11px]">
      <div class="flex items-center gap-3 font-semibold">
        <span class="text-rose-300">- 本地 (local)</span>
        <span class="text-emerald-300">+ 云端 (remote)</span>
      </div>
      <div class="flex items-center gap-2 font-mono">
        <span v-if="(addedLines ?? 0) > 0" class="rounded bg-emerald-500/20 px-1.5 py-0.5 text-emerald-300">+{{ addedLines }}</span>
        <span v-if="(removedLines ?? 0) > 0" class="rounded bg-rose-500/20 px-1.5 py-0.5 text-rose-300">-{{ removedLines }}</span>
      </div>
    </div>
    <div class="overflow-auto font-mono text-[11px] leading-5" :class="heightClass">
      <div v-if="!hasLines" class="px-3 py-4 text-center text-slate-500">无文本差异</div>
      <div
        v-for="(line, index) in lines"
        v-else
        :key="index"
        class="flex whitespace-pre px-2"
        :class="line.kind === 'add'
          ? 'bg-emerald-500/15 text-emerald-200'
          : line.kind === 'delete'
          ? 'bg-rose-500/15 text-rose-200'
          : 'text-slate-300'"
      >
        <span
          class="w-4 shrink-0 select-none text-center"
          :class="line.kind === 'add' ? 'text-emerald-400' : line.kind === 'delete' ? 'text-rose-400' : 'text-slate-600'"
        >{{ symbol(line.kind) }}</span>
        <span class="flex-1 break-all">{{ line.text || ' ' }}</span>
      </div>
    </div>
  </div>
</template>
