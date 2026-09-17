import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appSource = readFileSync(new URL('../src/App.vue', import.meta.url), 'utf8')
const styleSource = readFileSync(new URL('../src/style.css', import.meta.url), 'utf8')
const editorSource = readFileSync(new URL('../src/components/AnnouncementEditor.vue', import.meta.url), 'utf8')
const editorStyleSource = readFileSync(new URL('../src/components/AnnouncementEditor.css', import.meta.url), 'utf8')
const previewSource = readFileSync(new URL('../src/components/PreviewPlayer.vue', import.meta.url), 'utf8')
const playerSource = readFileSync(new URL('../src/components/PreviewPlayer.css', import.meta.url), 'utf8')

test('desktop shell removes fake window controls', () => {
  assert.equal(appSource.includes('window-actions'), false)
})

test('app shell removes the top overview banner', () => {
  assert.equal(appSource.includes('WorkspaceHeader'), false)
  assert.equal(appSource.includes('超市播报台'), false)
})

test('editor panel keeps only text and config controls', () => {
  assert.match(editorSource, /text-counter/)
  assert.match(editorSource, /voice-grid/)
  assert.equal(editorSource.includes('编辑播报'), false)
  assert.equal(editorSource.includes('文案内容'), false)
})

test('preview panel is reduced to transport controls', () => {
  assert.match(previewSource, /transport-row/)
  assert.match(previewSource, /player-controls/)
  assert.equal(previewSource.includes('生成结果'), false)
  assert.equal(previewSource.includes('试听与导出'), false)
})

test('desktop shell does not force double 100vh containers', () => {
  assert.equal(styleSource.includes('min-height: 100vh;'), false)
})

test('desktop shell fixes viewport height so panels can scroll internally', () => {
  assert.match(styleSource, /\.app-shell\s*\{[\s\S]*\n\s+height:\s*100%;/)
})

test('workspace uses vertical split layout', () => {
  assert.match(styleSource, /grid-template-rows:\s*minmax\(0, 1fr\) auto;/)
})

test('editor input uses compact minimum height', () => {
  assert.match(editorStyleSource, /min-height:\s*250px;/)
  assert.match(editorStyleSource, /grid-template-columns:\s*minmax\(0, 1\.9fr\) minmax\(300px, 0\.92fr\);/)
  assert.equal(editorStyleSource.includes('overflow: hidden;'), false)
})

test('player controls stay in a compact transport row', () => {
  assert.match(playerSource, /grid-template-columns:\s*auto minmax\(0, 1fr\) auto;/)
})
