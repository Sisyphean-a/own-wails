<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { ClipboardSetText } from '../wailsjs/runtime/runtime'
import { DeleteTextHistory, ListTextHistory } from '../wailsjs/go/main/App'
import { buildHistoryTree, expandedPath, searchHistory } from './historyTree'

const props = defineProps({ currentText: { type: String, default: '' }, generating: Boolean })
const emit = defineEmits(['close', 'apply'])
const entries = ref([])
const selectedId = ref('')
const expanded = ref(new Set())
const searchQuery = ref('')
const focusedMatch = ref(null)
const loading = ref(true)
const error = ref('')
const notice = ref('')
const pendingAction = ref('')
const busy = ref(false)
const confirmError = ref('')
const closeButton = ref(null)
const searchInput = ref(null)
const historyTextRef = ref(null)
const lineRefs = ref([])
const confirmCancel = ref(null)
const deleteButton = ref(null)

function message(error) {
  return typeof error === 'string' ? error : error?.message || '操作失败，请稍后重试。'
}

const history = computed(() => buildHistoryTree(entries.value))
const selected = computed(() => history.value.sorted.find(entry => entry.id === selectedId.value))
const searchTerm = computed(() => searchQuery.value.trim())
const searchResults = computed(() => searchHistory(entries.value, searchTerm.value))
const selectedLines = computed(() => selected.value?.text.split(/\r?\n/) || [])
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
    focusedMatch.value = null
    lineRefs.value = []
  } catch (e) {
    entries.value = []
    selectedId.value = ''
    error.value = message(e)
  } finally {
    loading.value = false
  }
}

function setLineRef(element, index) {
  if (element) lineRefs.value[index] = element
  else delete lineRefs.value[index]
}

function revealEntry(id) {
  selectedId.value = id
  expanded.value = new Set(expandedPath(history.value.tree, id) || [])
  lineRefs.value = []
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
    focusedMatch.value = null
    revealEntry(node.entry.id)
    nextTick(() => historyTextRef.value?.scrollTo({ top: 0, behavior: 'auto' }))
  }
}

function clearSearch() {
  searchQuery.value = ''
  focusedMatch.value = null
  notice.value = ''
  nextTick(() => searchInput.value?.focus())
}

async function selectSearchResult(result) {
  const query = searchTerm.value
  if (!query) return
  notice.value = ''
  error.value = ''
  focusedMatch.value = { id: result.entry.id, lineIndex: result.lineIndex, query }
  revealEntry(result.entry.id)
  await nextTick()
  lineRefs.value[result.lineIndex]?.scrollIntoView({ block: 'center', behavior: 'auto' })
  notice.value = `已定位到 ${result.entry.day} 的匹配内容`
}

function highlightParts(text, query) {
  const normalizedQuery = query.trim()
  if (!normalizedQuery) return [{ text, match: false }]
  const lowerText = text.toLocaleLowerCase()
  const lowerQuery = normalizedQuery.toLocaleLowerCase()
  const parts = []
  let cursor = 0
  let start = lowerText.indexOf(lowerQuery, cursor)
  while (start >= 0) {
    if (start > cursor) parts.push({ text: text.slice(cursor, start), match: false })
    const end = start + normalizedQuery.length
    parts.push({ text: text.slice(start, end), match: true })
    cursor = end
    start = lowerText.indexOf(lowerQuery, cursor)
  }
  if (cursor < text.length) parts.push({ text: text.slice(cursor), match: false })
  return parts.length ? parts : [{ text, match: false }]
}

function isTargetLine(index) {
  const target = focusedMatch.value
  return Boolean(target && target.id === selected.value?.id && target.lineIndex === index && target.query === searchTerm.value)
}

function isSearchResultActive(result) {
  const target = focusedMatch.value
  return Boolean(target && target.id === result.entry.id && target.lineIndex === result.lineIndex && target.query === searchTerm.value)
}

function searchResultLabel(result) {
  return `${result.entry.day} ${result.entry.createdAt.slice(11, 16)}：${result.line}`
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
    if (selected.value) emit('apply', selected.value.text)
    return
  }
  if (pendingAction.value !== 'delete' || busy.value || !selected.value) return
  busy.value = true
  confirmError.value = ''
  const index = history.value.sorted.findIndex(entry => entry.id === selectedId.value)
  try {
    await DeleteTextHistory(selectedId.value)
    entries.value = entries.value.filter(entry => entry.id !== selectedId.value)
    focusedMatch.value = null
    const nextId = searchTerm.value
      ? searchResults.value[0]?.entry.id || history.value.sorted[0]?.id || ''
      : history.value.sorted[Math.min(index, history.value.sorted.length - 1)]?.id || ''
    revealEntry(nextId)
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
  const controls = [...dialog.querySelectorAll('button:not(:disabled), input:not(:disabled)')]
  if (!controls.length) return
  if (event.shiftKey && document.activeElement === controls[0]) {
    event.preventDefault()
    controls.at(-1).focus()
  } else if (!event.shiftKey && document.activeElement === controls.at(-1)) {
    event.preventDefault()
    controls[0].focus()
  }
}

watch(searchQuery, () => {
  focusedMatch.value = null
  notice.value = ''
})

onMounted(() => {
  searchInput.value?.focus()
  load()
})
</script>

<template>
  <div class="dialog-backdrop history-backdrop" @mousedown.self="close" @keydown="handleKeydown">
    <section class="history-dialog" role="dialog" aria-modal="true" aria-labelledby="history-title">
      <header class="dialog-header" :inert="Boolean(pendingAction)" :aria-hidden="pendingAction ? 'true' : undefined">
        <div class="history-title-group"><h2 id="history-title">文案历史</h2></div>
        <button ref="closeButton" type="button" class="dialog-close" aria-label="关闭文案历史" :disabled="busy" @click="close">×</button>
      </header>
      <div class="history-searchbar" :inert="Boolean(pendingAction)" :aria-hidden="pendingAction ? 'true' : undefined">
        <label class="history-search">
          <span class="history-search-icon" aria-hidden="true">⌕</span>
          <span class="sr-only">搜索全部文案</span>
          <input ref="searchInput" v-model="searchQuery" type="search" autocomplete="off" spellcheck="false" placeholder="搜索全部文案，例如“红薯”">
          <button v-if="searchQuery" type="button" class="history-search-clear" aria-label="清除搜索" @click="clearSearch">×</button>
        </label>
        <span v-if="searchTerm" class="history-result-count" role="status">{{ searchResults.length }} 条匹配</span>
      </div>
      <div class="history-body" :inert="Boolean(pendingAction)" :aria-hidden="pendingAction ? 'true' : undefined">
        <nav class="history-sidebar" :aria-label="searchTerm ? '文案搜索结果' : '按日期浏览文案'">
          <template v-if="searchTerm">
            <p v-if="loading" class="history-hint" role="status">正在读取文案历史…</p>
            <p v-else-if="error && !entries.length" class="history-hint" role="alert">读取失败：{{ error }}<br><button type="button" class="history-link" @click="load">重试</button></p>
            <p v-else-if="!entries.length" class="history-hint">暂无文案记录</p>
            <p v-else-if="!searchResults.length" class="history-search-empty" role="status">没有找到「{{ searchTerm }}」<br><small>换个关键词试试</small></p>
            <div v-else class="history-search-results">
              <button v-for="result in searchResults" :key="`${result.entry.id}:${result.lineIndex}`" type="button" class="history-search-result" :class="{ active: isSearchResultActive(result) }" :aria-current="isSearchResultActive(result) ? 'true' : undefined" :aria-label="searchResultLabel(result)" @click="selectSearchResult(result)">
                <span class="history-result-meta">{{ result.entry.day }} · {{ result.entry.createdAt.slice(11, 16) }}</span>
                <span class="history-result-line"><template v-for="(part, partIndex) in highlightParts(result.line, searchTerm)" :key="`${result.entry.id}:${result.lineIndex}:${partIndex}`"><mark v-if="part.match">{{ part.text }}</mark><template v-else>{{ part.text }}</template></template></span>
              </button>
            </div>
          </template>
          <template v-else>
            <p v-if="loading" class="history-hint" role="status">正在读取文案历史…</p>
            <p v-else-if="error && !entries.length" class="history-hint" role="alert">读取失败：{{ error }}<br><button type="button" class="history-link" @click="load">重试</button></p>
            <p v-else-if="!entries.length" class="history-hint">暂无文案记录</p>
            <button v-for="node in visibleNodes" :key="node.key" type="button" class="history-row" :class="{ active: node.entry?.id === selectedId, group: node.children }" :style="{ paddingLeft: `${12 + node.depth * 16}px` }" :aria-label="node.children ? `${node.label}，${node.children.length} 条记录` : node.entry ? `${node.entry.day} ${node.label}` : node.label" :aria-expanded="node.children ? expanded.has(node.key) : undefined" :aria-current="node.entry?.id === selectedId ? 'true' : undefined" @click="selectNode(node)">
              <span class="history-chevron" aria-hidden="true">{{ node.children ? (expanded.has(node.key) ? '▾' : '▸') : '·' }}</span><span class="history-row-label" :title="node.label">{{ node.label }}</span>
            </button>
          </template>
        </nav>
        <div class="history-detail">
          <template v-if="selected && (!searchTerm || focusedMatch)">
            <div class="history-meta"><strong>{{ selected.day }} · {{ selected.createdAt.slice(11, 16) }}</strong><span>当次提交的文案</span></div>
            <article ref="historyTextRef" class="history-text" aria-label="历史文案正文">
              <span v-for="(line, index) in selectedLines" :key="`${selected.id}:${index}`" class="history-line" :class="{ 'is-target': isTargetLine(index) }" :ref="element => setLineRef(element, index)">
                <template v-if="line"><template v-for="(part, partIndex) in highlightParts(line, searchTerm)" :key="`${selected.id}:${index}:${partIndex}`"><mark v-if="part.match">{{ part.text }}</mark><template v-else>{{ part.text }}</template></template></template>
                <span v-else aria-hidden="true">&nbsp;</span>
              </span>
            </article>
            <div class="history-footer"><span class="history-feedback" role="status" aria-live="polite">{{ error && entries.length ? error : notice }}</span><button ref="deleteButton" class="history-delete" type="button" :disabled="busy" @click="startAction('delete')">删除记录</button><button class="dialog-cancel" type="button" @click="copyText">复制文案</button><button class="dialog-save" type="button" :disabled="generating" @click="startAction('apply')">应用到当前</button></div>
          </template>
          <p v-else class="history-empty">{{ loading ? '正在读取…' : error ? '请重试读取文案历史' : searchTerm ? '请选择左侧搜索结果查看完整文案。' : '生成一次播报后，文案会出现在这里。' }}</p>
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
.history-title-group { min-width:0; }
.history-searchbar { flex:none; display:flex; align-items:center; gap:12px; padding:10px 16px; border-bottom:1px solid var(--line); background:#15181a; }
.sr-only { position:absolute; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0,0,0,0); white-space:nowrap; border:0; }
.history-search { min-width:0; flex:1; height:36px; display:flex; align-items:center; gap:8px; padding:0 10px; border:1px solid #404a4d; border-radius:5px; background:var(--surface-input); color:var(--muted); }
.history-search:focus-within { border-color:var(--accent); }
.history-search-icon { flex:none; color:var(--dim); font-size:18px; line-height:1; transform:rotate(-20deg); }
.history-search input { min-width:0; flex:1; width:100%; height:32px; border:0; outline:0; background:transparent; color:var(--text); font-size:12px; }
.history-search input::placeholder { color:var(--dim); }
.history-search input::-webkit-search-cancel-button { display:none; }
.history-search-clear { flex:none; width:22px; height:22px; padding:0; border:0; border-radius:3px; background:transparent; color:var(--dim); font-size:18px; line-height:1; }
.history-search-clear:hover { background:var(--surface-raised); color:var(--text); }
.history-result-count { flex:none; color:var(--dim); font-size:11px; white-space:nowrap; }
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
.history-search-empty { padding:28px 12px; color:var(--muted); font-size:12px; line-height:1.8; text-align:center; }
.history-search-empty small { color:var(--dim); font-size:11px; }
.history-search-results { margin:-12px -8px; }
.history-search-result { width:100%; display:flex; flex-direction:column; align-items:stretch; gap:5px; padding:10px 12px; border:0; border-bottom:1px solid var(--line-soft); background:transparent; color:var(--text); text-align:left; }
.history-search-result:hover { background:var(--surface-raised); }
.history-search-result.active { background:var(--accent-soft); box-shadow:inset 2px 0 var(--accent); }
.history-result-meta { color:var(--dim); font:11px Consolas,monospace; }
.history-result-line { color:var(--text); font-size:12px; line-height:1.65; overflow-wrap:anywhere; }
.history-result-line mark,.history-text mark { border-radius:2px; background:rgba(71,197,186,.24); color:var(--text); }
.history-detail { min-width:0; min-height:0; display:flex; flex-direction:column; }
.history-meta { flex:none; padding:19px 24px 13px; display:flex; align-items:baseline; gap:12px; border-bottom:1px solid var(--line-soft); font-size:13px; }
.history-meta span { color:var(--dim); font-size:11px; }
.history-text { flex:1; min-height:0; overflow-y:auto; overflow-x:hidden; white-space:pre-wrap; overflow-wrap:anywhere; padding:22px 24px; font-size:14px; line-height:1.95; }
.history-line { display:block; min-height:1.95em; }
.history-line.is-target { margin:0 -8px; padding:0 8px; border-left:2px solid var(--accent); border-radius:2px; background:var(--accent-soft); }
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
  .history-searchbar { align-items:stretch; flex-direction:column; gap:7px; }
  .history-result-count { align-self:flex-end; }
  .history-body { grid-template-columns:1fr; grid-template-rows:minmax(130px,32%) minmax(0,1fr); }
  .history-sidebar { border-right:0; border-bottom:1px solid var(--line); }
  .history-meta { padding:12px 16px; }
  .history-text { padding:14px 16px; }
  .history-footer { padding:10px 12px; flex-wrap:wrap; }
  .history-feedback { flex-basis:100%; }
}
</style>
