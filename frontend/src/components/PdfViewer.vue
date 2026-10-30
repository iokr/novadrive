<template>
  <div class="pdf-viewer">
    <div v-if="loading" class="pdf-viewer__state">
      <el-icon class="spin"><Loading /></el-icon>
      <p>正在加载 PDF…</p>
    </div>

    <div v-else-if="error" class="pdf-viewer__state">
      <el-icon class="pdf-viewer__state-icon"><Document /></el-icon>
      <p>{{ error }}</p>
      <el-button type="primary" size="small" @click="load">重新加载</el-button>
    </div>

    <template v-else>
      <!-- 连续滚动：所有页顺序排列，按滚动位置渲染可视页 -->
      <div ref="stageRef" class="pdf-viewer__stage" @scroll.passive="onScroll">
        <div ref="pagesRef" class="pdf-viewer__pages">
          <div
            v-for="item in pageList"
            :key="item.n"
            :ref="pageRef(item.n)"
            class="pdf-viewer__page"
            :style="{ width: `${item.w}px`, height: `${item.h}px` }"
          >
            <canvas class="pdf-viewer__canvas" />
          </div>
        </div>
      </div>

      <div class="pdf-viewer__toolbar">
        <button type="button" :disabled="currentPage <= 1" title="上一页" @click="scrollToPage(currentPage - 1)">
          <el-icon><ArrowLeft /></el-icon>
        </button>
        <span class="pdf-viewer__pager num">
          <input
            v-model.number="pageInput"
            class="pdf-viewer__input"
            type="text"
            inputmode="numeric"
            @keyup.enter="scrollToPage(pageInput)"
            @blur="pageInput = currentPage"
          />
          / {{ pages }}
        </span>
        <button
          type="button"
          :disabled="currentPage >= pages"
          title="下一页"
          @click="scrollToPage(currentPage + 1)"
        >
          <el-icon><ArrowRight /></el-icon>
        </button>
        <span class="pdf-viewer__divider" />
        <button type="button" title="缩小" @click="zoomBy(-0.15)">
          <el-icon><ZoomOut /></el-icon>
        </button>
        <span class="pdf-viewer__scale num">{{ Math.round(scale * 100) }}%</span>
        <button type="button" title="放大" @click="zoomBy(0.15)">
          <el-icon><ZoomIn /></el-icon>
        </button>
        <button type="button" title="适应宽度" @click="fitWidth">
          <el-icon><FullScreen /></el-icon>
        </button>
        <button type="button" title="下载" @click="download">
          <el-icon><Download /></el-icon>
        </button>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import * as pdfjsLib from 'pdfjs-dist'
import pdfWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { rawUrl, downloadUrl } from '@/api/file'
import { downloadUrl as triggerUrl } from '@/utils/download'

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfWorker

const props = defineProps({
  file: { type: Object, default: null }
})

// 可视区域上下各预渲染一屏左右，滚动时下一页基本已经准备好了
const PREFETCH = 900
// 离当前页超过该距离就释放 canvas，避免长文档占满显存
const KEEP_DISTANCE = 4
const MAX_DPR = 2
const A4 = { w: 595, h: 842 }

const stageRef = ref(null)
const pagesRef = ref(null)
const loading = ref(true)
const error = ref('')
const pages = ref(0)
const currentPage = ref(1)
const pageInput = ref(1)
const scale = ref(1.2)

// 每页在缩放 1.0 时的尺寸；真实尺寸由 pdf.js 逐个读取，未读到的先按首页比例占位
const pageSizes = ref([])
const pageList = computed(() =>
  Array.from({ length: pages.value }, (_, i) => {
    const size = pageSizes.value[i] || pageSizes.value[0] || A4
    return {
      n: i + 1,
      w: Math.max(1, Math.round(size.w * scale.value)),
      h: Math.max(1, Math.round(size.h * scale.value))
    }
  })
)

let pdf = null
let loadingTask = null
let resizeObserver = null
// 每次 load / 释放都自增，用于丢弃过期的异步渲染结果
let loadToken = 0
let rafId = 0
// 缩放为"适应宽度"时按容器宽度重算，手动缩放后失效
let fitMode = true

const pageEls = new Map()
const refSetters = new Map()
const pageTops = ref([])
const rendered = new Set()
const renderTasks = new Map()
// 正在渲染（含等待 getPage）的页码，避免同一 canvas 被并发渲染
const rendering = new Set()
// 缩放批次号：缩放后丢弃上一批次尚未完成的渲染结果
let renderEpoch = 0

// 稳定的 ref 回调：避免每次渲染都重新绑定导致观察/缓存丢失
function pageRef(n) {
  let setter = refSetters.get(n)
  if (!setter) {
    setter = (el) => {
      if (el) pageEls.set(n, el)
      else pageEls.delete(n)
    }
    refSetters.set(n, setter)
  }
  return setter
}

async function load() {
  release()
  const token = ++loadToken
  loading.value = true
  error.value = ''
  pages.value = 0
  pageSizes.value = []
  pageTops.value = []
  currentPage.value = 1
  pageInput.value = 1
  if (!props.file) {
    loading.value = false
    return
  }
  try {
    loadingTask = pdfjsLib.getDocument({ url: rawUrl(props.file.id) })
    pdf = await loadingTask.promise
    if (token !== loadToken) return
    pages.value = pdf.numPages
    await initPageSizes()
    if (token !== loadToken) return
    // 先关掉 loading：stage / canvas 只在非 loading 分支渲染，否则测量拿不到容器
    loading.value = false
    await nextTick()
    await applyFitWidth()
    measurePages()
    renderWindow()
    // 其余页面的真实尺寸后台补齐，不阻塞首屏
    hydratePageSizes(token)
  } catch (err) {
    if (token !== loadToken) return
    loading.value = false
    error.value = err?.message ? `PDF 加载失败：${err.message}` : 'PDF 加载失败'
  }
}

// 先用首页比例铺满所有占位，保证首屏能立即排版
async function initPageSizes() {
  const first = (await pdf.getPage(1)).getViewport({ scale: 1 })
  pageSizes.value = Array.from({ length: pdf.numPages }, () => ({ w: first.width, h: first.height }))
}

// 再分批读取每页真实尺寸，避免大文档首屏等待
async function hydratePageSizes(token) {
  const total = pdf?.numPages || 0
  for (let start = 2; start <= total; start += 10) {
    const batch = []
    for (let n = start; n < Math.min(start + 10, total + 1); n++) batch.push(n)
    const sizes = pageSizes.value.slice()
    await Promise.all(
      batch.map(async (n) => {
        try {
          const viewport = (await pdf.getPage(n)).getViewport({ scale: 1 })
          sizes[n - 1] = { w: viewport.width, h: viewport.height }
        } catch {
          // 单页尺寸读取失败时沿用首页比例，不影响其余页面
        }
      })
    )
    if (token !== loadToken) return
    // 尺寸变化会改变已有页面高度，重排后按锚点恢复滚动位置
    const anchor = captureAnchor()
    pageSizes.value = sizes
    await nextTick()
    measurePages()
    restoreAnchor(anchor)
    renderWindow()
  }
}

// ---------- 渲染 / 释放 ----------

async function renderPage(n) {
  if (!pdf || n < 1 || n > pages.value) return
  if (rendered.has(n) || rendering.has(n)) return
  const el = pageEls.get(n)
  const canvas = el?.querySelector('canvas')
  if (!canvas) return
  const token = loadToken
  const epoch = renderEpoch
  rendering.add(n)
  let task = null
  try {
    const pdfPage = await pdf.getPage(n)
    if (token !== loadToken || epoch !== renderEpoch) return
    const viewport = pdfPage.getViewport({ scale: scale.value })
    const dpr = Math.min(window.devicePixelRatio || 1, MAX_DPR)
    canvas.width = Math.floor(viewport.width * dpr)
    canvas.height = Math.floor(viewport.height * dpr)
    canvas.style.width = `${Math.floor(viewport.width)}px`
    canvas.style.height = `${Math.floor(viewport.height)}px`
    task = pdfPage.render({
      canvasContext: canvas.getContext('2d'),
      viewport,
      transform: dpr !== 1 ? [dpr, 0, 0, dpr, 0, 0] : null
    })
    renderTasks.set(n, task)
    await task.promise
    if (token === loadToken && epoch === renderEpoch) rendered.add(n)
  } catch (err) {
    // 取消渲染会抛 RenderingCancelledException，属正常流程；单页失败不影响其它页
  } finally {
    if (renderTasks.get(n) === task) renderTasks.delete(n)
    rendering.delete(n)
  }
}

function releasePage(n) {
  const task = renderTasks.get(n)
  if (task) {
    try {
      task.cancel()
    } catch {
      // 忽略取消异常
    }
    renderTasks.delete(n)
  }
  const canvas = pageEls.get(n)?.querySelector('canvas')
  if (canvas) {
    canvas.width = 0
    canvas.height = 0
  }
  rendered.delete(n)
}

function clearRenders() {
  renderEpoch += 1
  for (const n of Array.from(renderTasks.keys())) releasePage(n)
  for (const n of Array.from(rendered)) releasePage(n)
}

function renderWindow() {
  const stage = stageRef.value
  if (!stage || !pageTops.value.length) return
  const top = stage.scrollTop - PREFETCH
  const bottom = stage.scrollTop + stage.clientHeight + PREFETCH
  const cur = currentPage.value
  for (let n = 1; n <= pages.value; n++) {
    const y = pageTops.value[n]
    if (y == null) continue
    const h = pageList.value[n - 1]?.h || 0
    if (y + h >= top && y <= bottom) renderPage(n)
    else if (rendered.has(n) && Math.abs(n - cur) > KEEP_DISTANCE) releasePage(n)
  }
}

// ---------- 滚动 / 定位 ----------

function measurePages() {
  const stage = stageRef.value
  if (!stage) return
  const stageTop = stage.getBoundingClientRect().top
  const tops = []
  pageEls.forEach((el, n) => {
    tops[n] = el.getBoundingClientRect().top - stageTop + stage.scrollTop
  })
  pageTops.value = tops
}

function updateCurrentPage() {
  const stage = stageRef.value
  if (!stage || !pageTops.value.length) return
  // 以视口 30% 处作为判定线，向上滚动时页码跟随更自然
  const line = stage.scrollTop + stage.clientHeight * 0.3
  let cur = 1
  for (let n = 1; n <= pages.value; n++) {
    const y = pageTops.value[n]
    if (y == null) break
    if (y <= line) cur = n
    else break
  }
  if (cur !== currentPage.value) {
    currentPage.value = cur
    pageInput.value = cur
  }
}

function onScroll() {
  if (rafId) return
  rafId = requestAnimationFrame(() => {
    rafId = 0
    updateCurrentPage()
    renderWindow()
  })
}

function scrollToPage(next) {
  const stage = stageRef.value
  const target = Math.min(pages.value, Math.max(1, Number(next) || 1))
  if (!stage) return
  pageInput.value = target
  const top = pageTops.value[target]
  if (top == null) return
  stage.scrollTo({ top: Math.max(0, top - 12), behavior: 'smooth' })
}

// ---------- 缩放 ----------

function captureAnchor() {
  const stage = stageRef.value
  if (!stage || !pageTops.value.length) return null
  const cur = currentPage.value
  const top = pageTops.value[cur]
  if (top == null) return null
  const height = pageList.value[cur - 1]?.h || 1
  return { page: cur, ratio: Math.min(1, Math.max(0, (stage.scrollTop - top) / height)) }
}

function restoreAnchor(anchor) {
  const stage = stageRef.value
  if (!stage || !anchor) return
  const top = pageTops.value[anchor.page]
  if (top == null) return
  const height = pageList.value[anchor.page - 1]?.h || 1
  stage.scrollTop = Math.max(0, top + anchor.ratio * height)
}

async function setScale(next) {
  const anchor = captureAnchor()
  scale.value = Math.min(5, Math.max(0.3, Math.round(next * 100) / 100))
  clearRenders()
  await nextTick()
  measurePages()
  restoreAnchor(anchor)
  renderWindow()
}

function zoomBy(delta) {
  fitMode = false
  setScale(scale.value + delta)
}

// 适应宽度：以首页宽度为基准（多数文档页面等宽）；个别横版页允许横向滚动
function fitScale() {
  const width = pagesRef.value?.clientWidth || stageRef.value?.clientWidth || A4.w
  const base = pageSizes.value[0]?.w || A4.w
  return Math.min(5, Math.max(0.3, Math.round((width / base) * 100) / 100))
}

async function applyFitWidth() {
  if (!pdf) return
  fitMode = true
  await setScale(fitScale())
}

function fitWidth() {
  applyFitWidth()
}

function download() {
  if (props.file) triggerUrl(downloadUrl(props.file.id))
}

// ---------- 生命周期 ----------

function release() {
  loadToken += 1
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = 0
  }
  clearRenders()
  pageEls.clear()
  refSetters.clear()
  if (loadingTask) {
    try {
      loadingTask.destroy()
    } catch {
      // 忽略销毁异常
    }
    loadingTask = null
  }
  pdf = null
}

function observeStage() {
  if (!stageRef.value || typeof ResizeObserver === 'undefined') return
  resizeObserver?.disconnect()
  resizeObserver = new ResizeObserver(() => {
    const anchor = captureAnchor()
    if (fitMode) {
      setScale(fitScale())
      return
    }
    measurePages()
    restoreAnchor(anchor)
    renderWindow()
  })
  resizeObserver.observe(stageRef.value)
}

watch(
  () => props.file?.id,
  () => load(),
  { immediate: true }
)

watch(
  () => stageRef.value,
  (el) => {
    if (el) observeStage()
  }
)

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  release()
})
</script>

<style scoped>
.pdf-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  position: relative;
}
.pdf-viewer__state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--nd-text-secondary);
  font-size: 13px;
}
.pdf-viewer__state-icon {
  font-size: 48px;
  color: var(--nd-primary);
}
.pdf-viewer__stage {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: auto;
  /* 底部为悬浮控制条留白 */
  padding: 18px 16px 84px;
  overscroll-behavior: contain;
}
.pdf-viewer__pages {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
}
.pdf-viewer__page {
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  background: #fff;
  box-shadow: var(--pv-image-shadow, 0 24px 60px rgba(0, 0, 0, 0.5));
}
.pdf-viewer__canvas {
  display: block;
  width: 100%;
  height: 100%;
}
.pdf-viewer__toolbar {
  position: absolute;
  left: 50%;
  bottom: 18px;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 3px;
  padding: 5px 10px;
  border-radius: 999px;
  border: 1px solid var(--pv-border, rgba(255, 255, 255, 0.14));
  background: var(--pv-dock-bg, rgba(15, 20, 35, 0.74));
  backdrop-filter: blur(18px) saturate(140%);
  -webkit-backdrop-filter: blur(18px) saturate(140%);
  box-shadow: var(--pv-shadow, 0 18px 40px rgba(0, 0, 0, 0.45));
  font-size: 12px;
  color: var(--pv-text-secondary, rgba(203, 213, 225, 0.85));
  white-space: nowrap;
  z-index: 2;
}
.pdf-viewer__toolbar button {
  width: 32px;
  height: 32px;
  border-radius: 999px;
  border: none;
  background: transparent;
  color: var(--pv-text-secondary, rgba(229, 231, 235, 0.9));
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
}
.pdf-viewer__toolbar button:hover:not(:disabled) {
  background: var(--pv-chip-hover, rgba(255, 255, 255, 0.14));
  color: var(--pv-text, #ffffff);
}
.pdf-viewer__toolbar button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.pdf-viewer__pager {
  display: flex;
  align-items: center;
  gap: 4px;
}
.pdf-viewer__input {
  width: 44px;
  height: 26px;
  text-align: center;
  border-radius: 8px;
  border: 1px solid var(--pv-border, rgba(255, 255, 255, 0.16));
  background: var(--pv-chip, rgba(255, 255, 255, 0.08));
  color: var(--pv-text, #e5e7eb);
  outline: none;
  font-size: 12px;
}
.pdf-viewer__input:focus {
  border-color: var(--nd-primary);
}
.pdf-viewer__divider {
  width: 1px;
  height: 16px;
  margin: 0 6px;
  background: var(--pv-border, rgba(255, 255, 255, 0.14));
}
.pdf-viewer__scale {
  min-width: 42px;
  text-align: center;
}
@media (max-width: 767px) {
  .pdf-viewer__stage {
    padding: 10px 8px 78px;
  }
  .pdf-viewer__pages {
    gap: 12px;
  }
  .pdf-viewer__toolbar {
    bottom: 12px;
    padding: 4px 6px;
  }
}
.spin {
  font-size: 26px;
  color: var(--nd-cyan);
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
