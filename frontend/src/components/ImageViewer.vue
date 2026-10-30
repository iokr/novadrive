<template>
  <div ref="rootRef" class="img-viewer" tabindex="0" @keydown="onKeydown">
    <div
      class="img-viewer__stage"
      :class="{ 'is-grab': zoom > 1 }"
      @wheel.prevent="onWheel"
      @mousedown="onMouseDown"
      @mousemove="onMouseMove"
      @mouseup="onMouseUp"
      @mouseleave="onMouseUp"
      @touchstart.passive="onTouchStart"
      @touchmove="onTouchMove"
      @touchend="onTouchEnd"
    >
      <div v-if="loading && !failed" class="img-viewer__loading">
        <el-icon class="spin"><Loading /></el-icon>
      </div>

      <img
        v-if="!failed"
        :key="current?.id"
        :src="current ? rawUrl(current.id) : ''"
        :style="imageStyle"
        :alt="current?.name"
        draggable="false"
        @load="onLoad"
        @error="onError"
      />
      <div v-else class="img-viewer__fallback">
        <el-icon><PictureRounded /></el-icon>
        <p>图片加载失败</p>
        <el-button type="primary" size="small" @click="download">下载原图</el-button>
      </div>

      <button
        v-if="files.length > 1"
        type="button"
        class="nav nav--prev"
        title="上一张"
        @click.stop="prev"
      >
        <el-icon><ArrowLeft /></el-icon>
      </button>
      <button
        v-if="files.length > 1"
        type="button"
        class="nav nav--next"
        title="下一张"
        @click.stop="next"
      >
        <el-icon><ArrowRight /></el-icon>
      </button>
    </div>

    <div class="img-viewer__dock">
      <button type="button" title="缩小" :disabled="zoom <= 0.2" @click="setZoom(zoom - 0.2)">
        <el-icon><ZoomOut /></el-icon>
      </button>
      <button type="button" title="放大" :disabled="zoom >= 8" @click="setZoom(zoom + 0.2)">
        <el-icon><ZoomIn /></el-icon>
      </button>
      <span class="dock__zoom num">{{ Math.round(zoom * 100) }}%</span>
      <span v-if="natural" class="dock__dim num">{{ natural.width }}×{{ natural.height }}</span>
      <span class="dock__sep" />
      <button type="button" title="旋转 90°" @click="rotate">
        <el-icon><RefreshRight /></el-icon>
      </button>
      <button type="button" title="实际大小" @click="reset">
        <span class="dock__ratio">1:1</span>
      </button>
      <button type="button" title="下载原图" @click="download">
        <el-icon><Download /></el-icon>
      </button>
      <template v-if="files.length > 1">
        <span class="dock__sep" />
        <span class="dock__counter num">{{ index + 1 }} / {{ files.length }}</span>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { rawUrl, downloadUrl } from '@/api/file'
import { downloadUrl as triggerUrl } from '@/utils/download'

const props = defineProps({
  files: { type: Array, default: () => [] },
  index: { type: Number, default: 0 }
})

const emit = defineEmits(['update:index'])

const rootRef = ref(null)
const zoom = ref(1)
const rotation = ref(0)
const offset = reactive({ x: 0, y: 0 })
const natural = ref(null)
const failed = ref(false)
const loading = ref(true)

const current = computed(() => props.files[props.index] || null)

const imageStyle = computed(() => ({
  transform: `translate(${offset.x}px, ${offset.y}px) scale(${zoom.value}) rotate(${rotation.value}deg)`,
  cursor: zoom.value > 1 ? 'grab' : 'default'
}))

watch(
  () => current.value?.id,
  () => {
    failed.value = false
    loading.value = true
    natural.value = null
    resetView()
    preload()
  },
  { immediate: true }
)

function resetView() {
  zoom.value = 1
  rotation.value = 0
  offset.x = 0
  offset.y = 0
}

function reset() {
  resetView()
}

function setZoom(value) {
  zoom.value = Math.min(8, Math.max(0.2, Math.round(value * 10) / 10))
  if (zoom.value <= 1) {
    offset.x = 0
    offset.y = 0
  }
}

function onWheel(event) {
  const step = event.deltaY < 0 ? 0.15 : -0.15
  setZoom(zoom.value + step)
}

function rotate() {
  rotation.value = (rotation.value + 90) % 360
}

function onLoad(event) {
  natural.value = { width: event.target.naturalWidth, height: event.target.naturalHeight }
  failed.value = false
  loading.value = false
}

function onError() {
  failed.value = true
  loading.value = false
}

// ---------- 拖拽平移 ----------
let dragging = false
let dragStart = { x: 0, y: 0, ox: 0, oy: 0 }

function onMouseDown(event) {
  if (zoom.value <= 1) return
  dragging = true
  dragStart = { x: event.clientX, y: event.clientY, ox: offset.x, oy: offset.y }
}

function onMouseMove(event) {
  if (!dragging) return
  offset.x = dragStart.ox + (event.clientX - dragStart.x)
  offset.y = dragStart.oy + (event.clientY - dragStart.y)
}

function onMouseUp() {
  dragging = false
}

// ---------- 触摸缩放 / 滑动切换 ----------
let touchStart = null

function onTouchStart(event) {
  if (event.touches.length === 1) {
    touchStart = { x: event.touches[0].clientX, y: event.touches[0].clientY, ox: offset.x, oy: offset.y }
  }
}

function onTouchMove(event) {
  if (event.touches.length !== 1 || !touchStart) return
  const dx = event.touches[0].clientX - touchStart.x
  const dy = event.touches[0].clientY - touchStart.y
  if (zoom.value > 1) {
    offset.x = touchStart.ox + dx
    offset.y = touchStart.oy + dy
  }
}

function onTouchEnd(event) {
  if (!touchStart) return
  const touch = event.changedTouches?.[0]
  if (touch && zoom.value <= 1) {
    const dx = touch.clientX - touchStart.x
    if (Math.abs(dx) > 60) {
      dx > 0 ? prev() : next()
    }
  }
  touchStart = null
}

function prev() {
  if (!props.files.length) return
  emit('update:index', (props.index - 1 + props.files.length) % props.files.length)
}

function next() {
  if (!props.files.length) return
  emit('update:index', (props.index + 1) % props.files.length)
}

function download() {
  if (current.value) triggerUrl(downloadUrl(current.value.id))
}

function preload() {
  const around = [props.index - 1, props.index + 1]
  for (const idx of around) {
    const file = props.files[(idx + props.files.length) % props.files.length]
    if (file && file.id !== current.value?.id) {
      const img = new Image()
      img.src = rawUrl(file.id)
    }
  }
}

function onKeydown(event) {
  if (event.key === 'ArrowLeft') prev()
  else if (event.key === 'ArrowRight') next()
  else if (event.key === '+' || event.key === '=') setZoom(zoom.value + 0.2)
  else if (event.key === '-') setZoom(zoom.value - 0.2)
}

onMounted(() => {
  rootRef.value?.focus()
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.img-viewer {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  position: relative;
  outline: none;
}
.img-viewer__stage {
  flex: 1;
  min-height: 0;
  position: relative;
  /* 用 flex 居中：grid 的 auto 行会让百分比 max-height 失效，竖图会按原始尺寸溢出 */
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  /* 底部留出悬浮控制条的位置，避免遮挡图片 */
  padding: 4px 64px 76px;
}
.img-viewer__stage.is-grab {
  cursor: grabbing;
}
.img-viewer__stage img {
  width: auto;
  height: auto;
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: var(--pv-image-shadow, 0 26px 60px rgba(0, 0, 0, 0.5));
  transition: transform 140ms ease-out;
  user-select: none;
  animation: img-in 240ms ease;
}
@keyframes img-in {
  from {
    opacity: 0.2;
    transform: scale(0.99);
  }
}
.img-viewer__loading {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  pointer-events: none;
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
.img-viewer__fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  color: var(--nd-text-secondary);
  font-size: 44px;
  text-align: center;
}
.img-viewer__fallback p {
  font-size: 13px;
}
.nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 46px;
  height: 46px;
  border-radius: 50%;
  border: 1px solid var(--pv-border, rgba(255, 255, 255, 0.14));
  background: var(--pv-nav-bg, rgba(17, 22, 38, 0.62));
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  color: var(--pv-text-secondary, #e5e7eb);
  display: grid;
  place-items: center;
  cursor: pointer;
  font-size: 19px;
  opacity: 0.78;
  transition: background 160ms ease, transform 160ms ease, opacity 160ms ease;
}
.nav:hover {
  opacity: 1;
  background: var(--nd-primary);
  color: #ffffff;
  transform: translateY(-50%) scale(1.06);
}
.nav--prev {
  left: 10px;
}
.nav--next {
  right: 10px;
}
.img-viewer__dock {
  position: absolute;
  left: 50%;
  bottom: 18px;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 5px 8px;
  border-radius: 999px;
  border: 1px solid var(--pv-border, rgba(255, 255, 255, 0.14));
  background: var(--pv-dock-bg, rgba(15, 20, 35, 0.74));
  backdrop-filter: blur(18px) saturate(140%);
  -webkit-backdrop-filter: blur(18px) saturate(140%);
  box-shadow: var(--pv-shadow, 0 18px 40px rgba(0, 0, 0, 0.45));
  white-space: nowrap;
}
.img-viewer__dock button {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--pv-text-secondary, rgba(229, 231, 235, 0.9));
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
}
.img-viewer__dock button:hover:not(:disabled) {
  background: var(--pv-chip-hover, rgba(255, 255, 255, 0.14));
  color: var(--pv-text, #ffffff);
}
.img-viewer__dock button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.dock__zoom {
  min-width: 42px;
  text-align: center;
  font-size: 11.5px;
  color: var(--pv-text-secondary, rgba(203, 213, 225, 0.85));
}
.dock__dim {
  padding: 0 6px;
  font-size: 11px;
  color: var(--pv-text-muted, rgba(148, 163, 184, 0.75));
}
.dock__sep {
  width: 1px;
  height: 16px;
  margin: 0 6px;
  background: var(--pv-border, rgba(255, 255, 255, 0.14));
}
.dock__counter {
  padding: 0 6px 0 2px;
  font-size: 11.5px;
  color: var(--pv-text-secondary, rgba(203, 213, 225, 0.85));
}
.dock__ratio {
  font-size: 10.5px;
  letter-spacing: 0.3px;
}
@media (max-width: 900px) {
  .dock__dim {
    display: none;
  }
}
@media (max-width: 767px) {
  .img-viewer__stage {
    padding: 0 8px 72px;
  }
  .nav {
    width: 38px;
    height: 38px;
    font-size: 16px;
    opacity: 0.6;
  }
  .nav--prev {
    left: 4px;
  }
  .nav--next {
    right: 4px;
  }
  .img-viewer__dock {
    bottom: 12px;
    padding: 4px 6px;
  }
}
</style>
