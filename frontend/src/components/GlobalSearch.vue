<template>
  <el-dialog
    :model-value="modelValue"
    width="min(680px, 92vw)"
    top="10vh"
    append-to-body
    :show-close="false"
    class="search-dialog"
    @update:model-value="$emit('update:modelValue', $event)"
    @opened="focusInput"
    @closed="onClosed"
  >
    <div class="search">
      <div class="search__field">
        <el-icon class="search__icon"><Search /></el-icon>
        <input
          ref="inputRef"
          v-model="keyword"
          class="search__input"
          type="text"
          placeholder="搜索文件名，回车打开第一个结果"
          @input="onInput"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.enter.prevent="pickActive"
          @keydown.esc="$emit('update:modelValue', false)"
        />
        <el-icon v-if="loading" class="search__spin"><Loading /></el-icon>
        <button v-else-if="keyword" type="button" class="search__clear" @click="clear">
          <el-icon><Close /></el-icon>
        </button>
      </div>

      <div class="search__body">
        <template v-if="keyword && results.length">
          <p class="search__section">共 {{ results.length }} 个结果</p>
          <ul class="search__list">
            <li
              v-for="(item, idx) in results"
              :key="item.id"
              class="search__item"
              :class="{ 'is-active': idx === active }"
              @mouseenter="active = idx"
              @click="open(item)"
            >
              <el-icon class="search__item-icon" :style="{ color: tintOf(item) }">
                <component :is="iconOf(item)" />
              </el-icon>
              <div class="search__item-main">
                <p class="ellipsis">
                  <template v-for="(part, i) in highlight(item.name)" :key="i">
                    <mark v-if="part.hit">{{ part.text }}</mark>
                    <span v-else>{{ part.text }}</span>
                  </template>
                </p>
                <p class="search__item-meta num">
                  {{ isDir(item) ? '文件夹' : formatBytes(item.size) }} ·
                  {{ relativeTime(item.updated_at) }}
                </p>
              </div>
              <el-icon class="search__item-go"><Right /></el-icon>
            </li>
          </ul>
        </template>

        <EmptyState
          v-else-if="keyword && !loading"
          title="没有找到匹配的文件"
          description="试试其它关键字，或检查文件是否已被删除"
        />

        <template v-else>
          <div v-if="recentFiles.length" class="search__recent">
            <p class="search__section">最近访问</p>
            <ul class="search__list">
              <li
                v-for="item in recentFiles"
                :key="`recent-${item.id}`"
                class="search__item"
                @click="open(item)"
              >
                <el-icon class="search__item-icon" :style="{ color: tintOf(item) }">
                  <component :is="iconOf(item)" />
                </el-icon>
                <div class="search__item-main">
                  <p class="ellipsis">{{ item.name }}</p>
                  <p class="search__item-meta num">
                    {{ isDir(item) ? '文件夹' : formatBytes(item.size) }} ·
                    {{ relativeTime(item.last_access_at || item.updated_at) }}
                  </p>
                </div>
              </li>
            </ul>
          </div>

          <div v-if="history.length" class="search__history">
            <div class="search__history-head">
              <span class="search__section">最近搜索</span>
              <el-button link type="primary" size="small" @click="clearHistory">清空</el-button>
            </div>
            <div class="search__tags">
              <button v-for="word in history" :key="word" type="button" class="tag" @click="useWord(word)">
                {{ word }}
              </button>
            </div>
          </div>
          <p v-if="!recentFiles.length && !history.length" class="search__tip text-muted">
            输入关键字开始搜索，支持按文件名模糊匹配
          </p>
        </template>
      </div>

      <div class="search__footer text-muted">
        <span><kbd>↑</kbd><kbd>↓</kbd> 选择</span>
        <span><kbd>Enter</kbd> 打开</span>
        <span><kbd>Esc</kbd> 关闭</span>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'
import EmptyState from './EmptyState.vue'
import * as fileApi from '@/api/file'
import { formatBytes, relativeTime } from '@/utils/format'
import { iconOf, isDir, tintOf } from '@/utils/fileType'

const HISTORY_KEY = 'novadrive_search_history'

const props = defineProps({
  modelValue: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'open'])

const inputRef = ref(null)
const keyword = ref('')
const results = ref([])
const loading = ref(false)
const active = ref(0)
const history = ref(readHistory())
const recentFiles = ref([])

let timer = null
let seq = 0

function readHistory() {
  try {
    return JSON.parse(localStorage.getItem(HISTORY_KEY) || '[]')
  } catch {
    return []
  }
}

function writeHistory(word) {
  const next = [word, ...history.value.filter((it) => it !== word)].slice(0, 8)
  history.value = next
  localStorage.setItem(HISTORY_KEY, JSON.stringify(next))
}

function clearHistory() {
  history.value = []
  localStorage.removeItem(HISTORY_KEY)
}

function focusInput() {
  nextTick(() => inputRef.value?.focus())
}

async function loadRecent() {
  try {
    const data = await fileApi.recent(8)
    recentFiles.value = Array.isArray(data) ? data : []
  } catch {
    recentFiles.value = []
  }
}

function onInput() {
  active.value = 0
  if (timer) clearTimeout(timer)
  const word = keyword.value.trim()
  if (!word) {
    results.value = []
    loading.value = false
    return
  }
  loading.value = true
  timer = setTimeout(() => search(word), 260)
}

async function search(word) {
  const mine = ++seq
  try {
    const data = await fileApi.list({ keyword: word, sort: 'updated_at', order: 'desc' })
    // 丢弃过期响应，避免快速输入时结果错乱
    if (mine !== seq) return
    results.value = data?.items || []
    active.value = 0
  } catch {
    if (mine === seq) results.value = []
  } finally {
    if (mine === seq) loading.value = false
  }
}

function highlight(name) {
  const word = keyword.value.trim()
  if (!word) return [{ text: name, hit: false }]
  const lower = name.toLowerCase()
  const target = word.toLowerCase()
  const parts = []
  let from = 0
  for (;;) {
    const idx = lower.indexOf(target, from)
    if (idx < 0) break
    if (idx > from) parts.push({ text: name.slice(from, idx), hit: false })
    parts.push({ text: name.slice(idx, idx + word.length), hit: true })
    from = idx + word.length
  }
  if (from < name.length) parts.push({ text: name.slice(from), hit: false })
  return parts.length ? parts : [{ text: name, hit: false }]
}

function move(step) {
  if (!results.value.length) return
  const next = active.value + step
  if (next < 0) active.value = results.value.length - 1
  else if (next >= results.value.length) active.value = 0
  else active.value = next
}

function pickActive() {
  if (results.value.length) open(results.value[active.value])
}

function useWord(word) {
  keyword.value = word
  onInput()
  focusInput()
}

function open(item) {
  emit('open', item)
  emit('update:modelValue', false)
  if (keyword.value.trim()) writeHistory(keyword.value.trim())
}

function clear() {
  keyword.value = ''
  results.value = []
  loading.value = false
  focusInput()
}

function onClosed() {
  if (timer) clearTimeout(timer)
  keyword.value = ''
  results.value = []
  active.value = 0
}

watch(
  () => props.modelValue,
  (openState) => {
    if (!openState) return
    active.value = 0
    if (!recentFiles.value.length) loadRecent()
  }
)
</script>

<style scoped>
.search {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.search__field {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 12px;
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
}
.search__icon {
  color: var(--nd-text-secondary);
  font-size: 17px;
}
.search__input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--nd-text);
  font-size: 15px;
}
.search__spin {
  color: var(--nd-cyan);
  animation: spin 1s linear infinite;
}
.search__clear {
  border: none;
  background: transparent;
  color: var(--nd-text-secondary);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.search__body {
  min-height: 180px;
  max-height: 52vh;
  overflow: auto;
}
.search__section {
  font-size: 12px;
  color: var(--nd-text-secondary);
  padding: 4px 2px 8px;
}
.search__list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.search__item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 12px;
  cursor: pointer;
}
.search__item.is-active {
  background: var(--nd-surface-hover);
}
.search__item-icon {
  font-size: 20px;
  flex: none;
}
.search__item-main {
  flex: 1;
  min-width: 0;
}
.search__item-main p {
  font-size: 13.5px;
}
.search__item-main mark {
  background: rgba(43, 127, 255, 0.18);
  color: inherit;
  border-radius: 3px;
  padding: 0 1px;
}
.search__item-meta {
  font-size: 11.5px;
  color: var(--nd-text-muted);
  margin-top: 2px;
}
.search__item-go {
  color: var(--nd-text-muted);
  font-size: 13px;
}
.search__history-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.search__recent {
  margin-bottom: 14px;
}
.search__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.tag {
  padding: 5px 12px;
  border-radius: 999px;
  border: 1px solid var(--nd-border);
  background: transparent;
  color: var(--nd-text-secondary);
  font-size: 12.5px;
  cursor: pointer;
}
.tag:hover {
  color: var(--nd-primary);
  border-color: var(--nd-primary);
  background: var(--nd-primary-soft);
}
.search__tip {
  padding: 28px 0;
  text-align: center;
  font-size: 13px;
}
.search__footer {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 11.5px;
  border-top: 1px solid var(--nd-border);
  padding-top: 10px;
}
kbd {
  display: inline-block;
  min-width: 20px;
  padding: 1px 5px;
  margin-right: 3px;
  border-radius: 5px;
  border: 1px solid var(--nd-border);
  font-family: inherit;
  font-size: 11px;
  text-align: center;
}
.spin {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
