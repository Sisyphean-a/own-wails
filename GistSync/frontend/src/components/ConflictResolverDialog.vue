<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DiffViewer from './DiffViewer.vue'
import type { ApplyConflict } from '../lib/backend'

const props = defineProps<{
  visible: boolean
  conflicts: ApplyConflict[]
  defaultOverwriteAll?: boolean
  submitting?: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: [overwriteItemIds: string[]]
}>()

const decisions = reactive<Record<string, boolean>>({})

watch(
  () => [props.visible, props.conflicts],
  () => {
    if (!props.visible) {
      return
    }
    for (const conflict of props.conflicts) {
      decisions[conflict.itemId] = props.defaultOverwriteAll ?? false
    }
  },
  { deep: true, immediate: true },
)

const overwriteItemIds = computed(() => {
  return props.conflicts.filter((conflict) => decisions[conflict.itemId]).map((conflict) => conflict.itemId)
})
const expanded = reactive<Record<string, boolean>>({})

function setAll(value: boolean): void {
  for (const conflict of props.conflicts) {
    decisions[conflict.itemId] = value
  }
}

function confirm(): void {
  emit('confirm', overwriteItemIds.value)
}

function toggleDiff(itemId: string): void {
  expanded[itemId] = !expanded[itemId]
}

function diffStatusText(status: string): string {
  switch (status) {
    case 'ready':
      return '可预览'
    case 'binary_unsupported':
      return '二进制文件，暂不支持预览'
    case 'too_large':
      return '文件过大，暂不展示 diff'
    case 'decode_failed':
      return '解密失败，无法展示 diff'
    case 'read_failed':
      return '读取失败，无法展示 diff'
    default:
      return '无可用 diff'
  }
}
</script>

<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-sm transition-all duration-300">
    <div class="w-full max-w-2xl rounded-2xl border border-slate-200 bg-white shadow-2xl overflow-hidden flex flex-col transform scale-100 transition-all">
      <!-- Modal Header -->
      <div class="border-b border-slate-100 bg-slate-50/50 px-6 py-4 flex items-start justify-between">
        <div>
          <h3 class="text-base font-bold text-slate-900 flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5 text-amber-500">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            检测到本地文件冲突
          </h3>
          <p class="mt-1 text-xs text-slate-500">当前准备写入的文件在本地已存在。请选择是否覆盖它们，未勾选的项目将跳过恢复。</p>
        </div>
        <button 
          type="button" 
          class="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition" 
          :disabled="props.submitting"
          @click="emit('close')"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Modal Body -->
      <div class="p-6 space-y-4">
        <div class="flex items-center gap-2">
          <button 
            class="inline-flex items-center gap-1 rounded-lg bg-rose-50 hover:bg-rose-100 text-rose-700 border border-rose-200 px-3 py-1.5 text-xs font-semibold transition" 
            :disabled="props.submitting" 
            @click="setAll(true)"
          >
            全部覆盖
          </button>
          <button 
            class="inline-flex items-center gap-1 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 px-3 py-1.5 text-xs font-semibold transition shadow-sm" 
            :disabled="props.submitting" 
            @click="setAll(false)"
          >
            全部跳过
          </button>
        </div>

        <div class="max-h-64 overflow-y-auto rounded-xl border border-slate-200 bg-slate-50/50 p-1">
          <table class="w-full text-left text-xs border-collapse">
            <thead class="bg-slate-100 text-slate-600 font-semibold border-b border-slate-200 sticky top-0">
              <tr>
                <th class="w-16 px-4 py-2.5 text-center">覆盖</th>
                <th class="px-4 py-2.5">文件物理路径</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200/60 bg-white">
              <tr v-for="conflict in conflicts" :key="conflict.itemId" class="hover:bg-slate-50/50 transition">
                <td class="px-4 py-3 text-center">
                  <input v-model="decisions[conflict.itemId]" :disabled="props.submitting" type="checkbox" class="rounded text-rose-600 focus:ring-rose-500">
                </td>
                <td class="px-4 py-3 text-slate-700 whitespace-normal break-all">
                  <div class="font-mono select-all">{{ conflict.targetPath }}</div>
                  <div class="mt-1 flex flex-wrap items-center gap-2">
                    <button
                      v-if="conflict.diffStatus === 'ready'"
                      type="button"
                      class="text-[11px] font-semibold text-indigo-600 hover:text-indigo-800"
                      @click="toggleDiff(conflict.itemId)"
                    >
                      {{ expanded[conflict.itemId] ? '收起差异' : '查看差异' }}
                    </button>
                    <span
                      v-if="conflict.diffStatus === 'ready' && (conflict.addedLines > 0 || conflict.removedLines > 0)"
                      class="inline-flex items-center gap-1.5 text-[11px] font-mono"
                    >
                      <span class="text-emerald-600">+{{ conflict.addedLines }}</span>
                      <span class="text-rose-600">-{{ conflict.removedLines }}</span>
                    </span>
                    <span v-else-if="conflict.diffStatus !== 'ready'" class="text-[11px] text-slate-500">{{ diffStatusText(conflict.diffStatus) }}</span>
                  </div>
                  <DiffViewer
                    v-if="expanded[conflict.itemId] && conflict.diffStatus === 'ready'"
                    :lines="conflict.diffLines"
                    :added-lines="conflict.addedLines"
                    :removed-lines="conflict.removedLines"
                    max-height-class="max-h-40"
                    class="mt-2"
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="flex items-center justify-between border-t border-slate-100 bg-slate-50/50 px-6 py-4">
        <p class="text-xs font-semibold text-slate-500">
          已选择覆盖 <span class="text-rose-600 font-bold">{{ overwriteItemIds.length }}</span> 项，跳过 <span class="text-slate-600 font-bold">{{ conflicts.length - overwriteItemIds.length }}</span> 项
        </p>
        <div class="flex gap-2">
          <button 
            class="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 transition shadow-sm" 
            :disabled="props.submitting" 
            @click="emit('close')"
          >
            取消
          </button>
          <button 
            class="rounded-lg bg-slate-900 hover:bg-slate-800 px-4 py-2 text-sm font-semibold text-white transition shadow-sm inline-flex items-center gap-2" 
            :disabled="props.submitting" 
            @click="confirm"
          >
            <span v-if="props.submitting" class="h-4 w-4 animate-spin rounded-full border-2 border-slate-400 border-t-white" />
            <span>确认执行</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
