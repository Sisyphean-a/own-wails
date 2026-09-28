<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { ClipboardSetText } from '../wailsjs/runtime/runtime'
import { DeleteTextHistory, ListTextHistory } from '../wailsjs/go/main/App'
import { buildHistoryTree, expandedPath } from './historyTree'

const props = defineProps({ currentText: { type: String, default: '' }, generating: Boolean })
const emit = defineEmits(['close', 'apply'])
const entries = ref([])
const selectedId = ref('')
const expanded = ref(new Set())
const loading = ref(true)
const error = ref('')
const notice = ref('')
const pendingAction = ref('')
const busy = ref(false)
const confirmError = ref('')
const closeButton = ref(null)
const confirmCancel = ref(null)
const deleteButton = ref(null)
function message(error) {
  return typeof error === 'string' ? error : error?.message || '操作失败，请稍后重试。'
}
const history = computed(() => buildHistoryTree(entries.value))
const selected = computed(() => history.value.sorted.find(entry => entry.id === selectedId.value))
const visibleNodes = computed(() => {
  const nodes = []
  function visit(items, depth) {
    for (const node of items) {
      nodes.push({ ...node, depth })
      if (node.children && expanded.value.has(node.key)) visit(node.children, depth + 1)
    }
  }
  visit(history.value.tree, 0)
  return nodes
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    entries.value = await ListTextHistory() || []
    if (!entries.value.some(entry => entry.id === selectedId.value)) selectedId.value = history.value.sorted[0]?.id || ''
    expanded.value = new Set(expandedPath(history.value.tree, selectedId.value) || [])
  } catch (e) {
    entries.value = []
    selectedId.value = ''
    error.value = message(e)
  } finally {
    loading.value = false
  }
}

function selectNode(node) {
  notice.value = ''
  error.value = ''
  if (node.children) {
    const next = new Set(expanded.value)
    if (next.has(node.key)) next.delete(node.key)
    else next.add(node.key)
    expanded.value = next
  } else {
    selectedId.value = node.entry.id
  }
}

function close() {
  if (!busy.value && !pendingAction.value) emit('close')
}

function startAction(action) {
  if (!selected.value || busy.value) return
  confirmError.value = ''
  if (action === 'apply' && (!props.currentText.trim() || props.currentText.trim() === selected.value.text)) {
    emit('apply', selected.value.text)
    return
  }
  pendingAction.value = action
  nextTick(() => confirmCancel.value?.focus())
}

function cancelAction() {
  if (busy.value) return
  const action = pendingAction.value
  pendingAction.value = ''
  confirmError.value = ''
  nextTick(() => action === 'delete' ? deleteButton.value?.focus() : closeButton.value?.focus())
}

async function confirmAction() {
  if (pendingAction.value === 'apply') {
    emit('apply', selected.value.text)
    return
  }
  if (pendingAction.value !== 'delete' || busy.value || !selected.value) return
  busy.value = true
  confirmError.value = ''
  const index = history.value.sorted.findIndex(entry => entry.id === selectedId.value)
  try {
    await DeleteTextHistory(selectedId.value)
    entries.value = entries.value.filter(entry => entry.id !== selectedId.value)
    selectedId.value = history.value.sorted[Math.min(index, history.value.sorted.length - 1)]?.id || ''
    expanded.value = new Set(expandedPath(history.value.tree, selectedId.value) || [])
    pendingAction.value = ''
    notice.value = '文案记录已删除'
    await nextTick()
    closeButton.value?.focus()
  } catch (e) {
    confirmError.value = message(e)
  } finally {
    busy.value = false
  }
}

async function copyText() {
  if (!selected.value) return
  notice.value = ''
  error.value = ''
  try {
    if (!await ClipboardSetText(selected.value.text)) throw new Error('复制失败，请检查剪贴板权限')
    notice.value = '已复制文案'
  } catch (e) {
    error.value = message(e)
  }
}

function handleKeydown(event) {
  if (event.key === 'Escape') {
    event.preventDefault()
    if (pendingAction.value) cancelAction()
    else close()
  }
  if (event.key !== 'Tab') return
  const dialog = pendingAction.value ? event.currentTarget.querySelector('.history-confirm') : event.currentTarget.querySelector('.history-dialog')
  const controls = [...dialog.querySelectorAll('button:not(:disabled)')]
  if (!controls.length) return
  if (event.shiftKey && document.activeElement === controls[0]) {
    event.preventDefault()
    controls.at(-1).focus()
  } else if (!event.shiftKey && document.activeElement === controls.at(-1)) {
    event.preventDefault()
    controls[0].focus()
  }
}

onMounted(() => {
  closeButton.value?.focus()
  load()
})
</script>

<template>
  <div class="dialog-backdrop history-backdrop" @mousedown.self="close" @keydown="handleKeydown">
    <section class="history-dialog" role="dialog" aria-modal="true" aria-labelledby="history-title">
      <header class="dialog-header" :inert="Boolean(pendingAction)" :aria-hidden="pendingAction ? 'true' : undefined"><h2 id="history-title">文案历史</h2><button ref="closeButton" type="button" class="dialog-close" aria-label="关闭文案历史" :disabled="busy" @click="close">×</button></header>
      <div class="history-body" :inert="Boolean(pendingAction)" :aria-hidden="pendingAction ? 'true' : undefined">
        <nav class="history-sidebar" aria-label="按日期浏览文案">
          <p v-if="loading" class="history-hint" role="status">正在读取文案历史…</p>
          <p v-else-if="error && !entries.length" class="history-hint" role="alert">读取失败：{{ error }}<br><button type="button" class="history-link" @click="load">重试</button></p>
          <p v-else-if="!entries.length" class="history-hint">暂无文案记录</p>
          <button v-for="node in visibleNodes" :key="node.key" type="button" class="history-row" :class="{ active: node.entry?.id === selectedId, group: node.children }" :style="{ paddingLeft: `${12 + node.depth * 16}px` }" :aria-expanded="node.children ? expanded.has(node.key) : undefined" :aria-current="node.entry?.id === selectedId ? 'true' : undefined" @click="selectNode(node)">
            <span class="history-chevron" aria-hidden="true">{{ node.children ? (expanded.has(node.key) ? '▾' : '▸') : '·' }}</span><span class="history-row-label" :title="node.label">{{ node.label }}</span>
          </button>
        </nav>
        <div class="history-detail">
          <template v-if="selected">
            <div class="history-meta"><strong>{{ selected.day }} · {{ selected.createdAt.slice(11, 16) }}</strong><span>当次提交的文案</span></div>
            <article class="history-text" aria-label="历史文案正文">{{ selected.text }}</article>
            <div class="history-footer"><span class="history-feedback" role="status" aria-live="polite">{{ error && entries.length ? error : notice }}</span><button ref="deleteButton" class="history-delete" type="button" :disabled="busy" @click="startAction('delete')">删除记录</button><button class="dialog-cancel" type="button" @click="copyText">复制文案</button><button class="dialog-save" type="button" :disabled="generating" @click="startAction('apply')">应用到当前</button></div>
          </template>
          <p v-else class="history-empty">{{ loading ? '正在读取…' : error ? '请重试读取文案历史' : '生成一次播报后，文案会出现在这里。' }}</p>
        </div>
      </div>
      <div v-if="pendingAction" class="history-confirm-overlay">
        <section class="history-confirm" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-description">
          <h3 id="confirm-title">{{ pendingAction === 'delete' ? '确认删除这条文案？' : '覆盖当前文案？' }}</h3>
          <p id="confirm-description">{{ pendingAction === 'delete' ? `将删除 ${selected?.day} 的「${selected?.text.slice(0, 32)}」文案记录。此操作无法撤销，不会删除 MP3。` : '当前编辑区有不同的文案。应用历史文案将覆盖现有内容，不会恢复历史音频。' }}</p>
          <p v-if="confirmError" class="history-confirm-error" role="alert">{{ confirmError }}</p>
          <div class="history-confirm-actions"><button ref="confirmCancel" type="button" class="dialog-cancel" :disabled="busy" @click="cancelAction">取消</button><button type="button" :class="pendingAction === 'delete' ? 'history-danger' : 'dialog-save'" :disabled="busy" @click="confirmAction">{{ busy ? '删除中…' : pendingAction === 'delete' ? '确认删除' : '覆盖并应用' }}</button></div>
        </section>
      </div>
    </section>
  </div>
</template>

<style scoped>
.history-dialog { position:relative; width:min(900px,100%); height:min(85vh,680px); min-height:320px; display:flex; flex-direction:column; border:1px solid #3a4648; border-radius:8px; background:var(--surface); box-shadow:0 24px 64px rgba(0,0,0,.5); overflow:hidden; }
.history-dialog .dialog-header { flex:none; }
.history-body { display:grid; grid-template-columns:minmax(220px,260px) minmax(0,1fr); flex:1; min-height:0; }
.history-sidebar { min-width:0; overflow-y:auto; border-right:1px solid var(--line); padding:12px 8px; background:#15181a; }
.history-row { width:100%; min-height:36px; display:flex; align-items:center; gap:8px; padding-right:10px; border:0; border-radius:4px; background:transparent; text-align:left; color:var(--muted); font-size:12px; }
.history-row:hover { background:var(--surface-raised); color:var(--text); }
.history-row.active { background:var(--accent-soft); color:var(--text); box-shadow:inset 2px 0 var(--accent); }
.history-row.group { font-weight:600; color:var(--text); }
.history-chevron { width:10px; flex:none; color:var(--dim); }
.history-row-label { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.history-hint { padding:6px 12px; color:var(--muted); font-size:12px; line-height:1.7; }
.history-link { margin-top:10px; padding:0; border:0; background:transparent; color:var(--accent); text-decoration:underline; }
.history-detail { min-width:0; min-height:0; display:flex; flex-direction:column; }
.history-meta { flex:none; padding:19px 24px 13px; display:flex; align-items:baseline; gap:12px; border-bottom:1px solid var(--line-soft); font-size:13px; }
.history-meta span { color:var(--dim); font-size:11px; }
.history-text { flex:1; min-height:0; overflow-y:auto; white-space:pre-wrap; overflow-wrap:anywhere; padding:22px 24px; font-size:14px; line-height:1.95; }
.history-footer { min-height:64px; padding:12px 20px; display:flex; align-items:center; gap:8px; border-top:1px solid var(--line); }
.history-feedback { min-width:0; flex:1; color:var(--accent); font-size:11px; overflow-wrap:anywhere; }
.history-delete { border:0; background:transparent; color:var(--muted); font-size:12px; padding:8px; }
.history-delete:hover { color:var(--danger); }
.history-footer .dialog-cancel,.history-footer .dialog-save { white-space:nowrap; }
.history-empty { margin:auto; padding:24px; color:var(--muted); font-size:13px; text-align:center; }
.history-confirm-overlay { position:absolute; inset:0; display:grid; place-items:center; padding:20px; background:rgba(0,0,0,.74); }
.history-confirm { width:min(400px,100%); padding:22px; border:1px solid #3a4648; border-radius:8px; background:var(--surface-raised); box-shadow:0 16px 40px rgba(0,0,0,.4); }
.history-confirm h3 { margin:0 0 14px; font-size:16px; }
.history-confirm p { margin:0; color:var(--muted); font-size:12px; line-height:1.7; overflow-wrap:anywhere; }
.history-confirm .history-confirm-error { margin-top:12px; color:var(--danger); }
.history-confirm-actions { display:flex; justify-content:flex-end; gap:8px; margin-top:22px; }
.history-danger { min-height:34px; padding:0 15px; border:1px solid var(--danger); border-radius:4px; background:rgba(240,139,131,.13); color:var(--danger); font-size:12px; font-weight:600; }
.history-danger:hover:not(:disabled) { background:rgba(240,139,131,.23); }
@media(max-width:760px) {
  .history-dialog { height:min(90vh,680px); }
  .history-body { grid-template-columns:1fr; grid-template-rows:minmax(130px,32%) minmax(0,1fr); }
  .history-sidebar { border-right:0; border-bottom:1px solid var(--line); }
  .history-meta { padding:12px 16px; }
  .history-text { padding:14px 16px; }
  .history-footer { padding:10px 12px; flex-wrap:wrap; }
  .history-feedback { flex-basis:100%; }
}
</style>
