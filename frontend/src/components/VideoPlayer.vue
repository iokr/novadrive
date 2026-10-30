<template>
  <div
    ref="rootRef"
    class="video-player"
    :class="{ 'is-idle': idle }"
    @mousemove="onActivity"
    @touchstart.passive="onActivity"
  >
    <div class="video-player__stage" @click="onStageClick" @dblclick="toggleFullscreen">
      <video
        v-if="!failed"
        ref="videoRef"
        :src="src"
        class="video-player__video"
        playsinline
        preload="metadata"
        @loadedmetadata="syncDuration"
        @durationchange="syncDuration"
        @timeupdate="onTimeUpdate"
        @progress="syncBuffered"
        @play="playing = true"
        @pause="playing = false"
        @waiting="buffering = true"
        @canplay="buffering = false"
        @playing="buffering = false"
        @ended="playing = false"
        @error="onError"
      />

      <!-- 暂停态的大号播放按钮，避免原生控件不可见时"点了没反应" -->
      <button
        v-if="!failed && !playing && !buffering"
        type="button"
        class="video-player__hint"
        title="播放"
        @click.stop="togglePlay"
      >
        <el-icon><VideoPlay /></el-icon>
      </button>

      <div v-if="buffering && !failed" class="video-player__loading">
        <el-icon class="spin"><Loading /></el-icon>
      </div>

      <div v-if="failed" class="video-player__fallback">
        <el-icon><VideoCamera /></el-icon>
        <p>当前浏览器无法播放该视频格式（{{ extText }}）</p>
        <el-button type="primary" size="small" @click.stop="download">下载后本地播放</el-button>
      </div>

      <div v-if="!failed" class="video-player__dock" @click.stop @dblclick.stop>
        <button type="button" :title="playing ? '暂停（空格）' : '播放（空格）'" @click="togglePlay">
          <el-icon>
            <VideoPause v-if="playing" />
            <VideoPlay v-else />
          </el-icon>
        </button>

        <span class="dock__time num">{{ timeText }}</span>

        <div ref="barRef" class="dock__progress" @mousedown.prevent="onSeekStart">
          <div class="dock__buffer" :style="{ width: `${bufferedPercent}%` }" />
          <div class="dock__played" :style="{ width: `${playedPercent}%` }" />
          <span class="dock__knob" :class="{ 'is-on': seeking }" :style="{ left: `${playedPercent}%` }" />
        </div>

        <span class="dock__total num">{{ totalText }}</span>

        <button type="button" class="dock__rate" title="播放速度" @click="cycleRate">{{ rate }}x</button>

        <div class="dock__volume">
          <button type="button" :title="muted ? '取消静音' : '静音'" @click="toggleMute">
            <el-icon :class="{ 'is-off': muted }"><Mute /></el-icon>
          </button>
          <input
            v-model.number="volume"
            class="dock__slider"
            type="range"
            min="0"
            max="1"
            step="0.01"
            aria-label="音量"
            @input="onVolume"
          />
        </div>

        <button type="button" title="下载" @click="download">
          <el-icon><Download /></el-icon>
        </button>
        <button type="button" :title="isFullscreen ? '退出全屏（Esc）' : '全屏'" @click="toggleFullscreen">
          <el-icon><FullScreen /></el-icon>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { rawUrl, downloadUrl } from '@/api/file'
import { downloadUrl as triggerUrl } from '@/utils/download'
import { extOf } from '@/utils/fileType'

const props = defineProps({
  file: { type: Object, default: null }
})

const rates = [0.75, 1, 1.25, 1.5, 2]

const rootRef = ref(null)
const videoRef = ref(null)
const barRef = ref(null)

const failed = ref(false)
const buffering = ref(true)
const playing = ref(false)
const duration = ref(0)
const current = ref(0)
const bufferedEnd = ref(0)
const rate = ref(1)
const volume = ref(1)
const muted = ref(false)
const isFullscreen = ref(false)
const seeking = ref(false)
// 播放中鼠标 / 手指长时间不动时淡出控制条，移动后再显示
const idle = ref(false)
const IDLE_MS = 5000
let idleTimer = null

const src = computed(() => (props.file ? rawUrl(props.file.id) : ''))
const extText = computed(() => (extOf(props.file) || '未知').toUpperCase())

const playedPercent = computed(() => {
  if (!duration.value) return 0
  return Math.min(100, Math.max(0, (current.value / duration.value) * 100))
})

const bufferedPercent = computed(() => {
  if (!duration.value) return 0
  return Math.min(100, Math.max(0, (bufferedEnd.value / duration.value) * 100))
})

const timeText = computed(() => fmtTime(current.value))
const totalText = computed(() => fmtTime(duration.value))

function fmtTime(sec) {
  const total = Number.isFinite(sec) && sec > 0 ? Math.floor(sec) : 0
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const pad = (n) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`
}

function syncDuration() {
  const video = videoRef.value
  if (!video) return
  duration.value = Number.isFinite(video.duration) ? video.duration : 0
  syncBuffered()
}

function syncBuffered() {
  const video = videoRef.value
  if (!video || !video.buffered || !video.buffered.length) return
  bufferedEnd.value = video.buffered.end(video.buffered.length - 1)
}

function onTimeUpdate() {
  const video = videoRef.value
  if (!video || seeking.value) return
  current.value = video.currentTime
  syncBuffered()
}

function onError() {
  failed.value = true
  buffering.value = false
  playing.value = false
}

function togglePlay() {
  const video = videoRef.value
  if (!video) return
  if (video.paused) {
    buffering.value = true
    video.play().catch(() => {
      buffering.value = false
    })
  } else {
    video.pause()
  }
}

// ---------- 控制条自动隐藏 ----------
function clearIdleTimer() {
  if (idleTimer) {
    clearTimeout(idleTimer)
    idleTimer = null
  }
}

function scheduleIdle() {
  clearIdleTimer()
  // 暂停 / 缓冲中不隐藏，避免用户在“没有控件”的状态下误以为卡死
  if (!playing.value) {
    idle.value = false
    return
  }
  idleTimer = setTimeout(() => {
    if (playing.value && !seeking.value) idle.value = true
  }, IDLE_MS)
}

function onActivity() {
  idle.value = false
  scheduleIdle()
}

function onStageClick() {
  // 控件已隐藏时，第一下点击只负责把它们叫回来，不打断播放
  if (idle.value) {
    onActivity()
    return
  }
  togglePlay()
}

function cycleRate() {
  const idx = rates.indexOf(rate.value)
  const next = rates[(idx + 1) % rates.length]
  rate.value = next
  if (videoRef.value) videoRef.value.playbackRate = next
}

function onVolume() {
  const video = videoRef.value
  if (!video) return
  muted.value = volume.value === 0
  video.muted = muted.value
  video.volume = volume.value
}

function toggleMute() {
  const video = videoRef.value
  if (!video) return
  if (muted.value || volume.value === 0) {
    muted.value = false
    if (volume.value === 0) volume.value = 1
    video.muted = false
    video.volume = volume.value
  } else {
    muted.value = true
    video.muted = true
  }
}

// ---------- 进度条拖拽 ----------
function ratioFromEvent(event) {
  const bar = barRef.value
  if (!bar) return 0
  const rect = bar.getBoundingClientRect()
  if (!rect.width) return 0
  const x = Math.min(Math.max(event.clientX - rect.left, 0), rect.width)
  return x / rect.width
}

function seekTo(ratio) {
  const video = videoRef.value
  if (!video || !duration.value) return
  const target = Math.min(duration.value, Math.max(0, ratio * duration.value))
  current.value = target
  video.currentTime = target
}

function onSeekStart(event) {
  seeking.value = true
  seekTo(ratioFromEvent(event))
  window.addEventListener('mousemove', onSeekMove)
  window.addEventListener('mouseup', onSeekEnd)
}

function onSeekMove(event) {
  if (!seeking.value) return
  seekTo(ratioFromEvent(event))
}

function onSeekEnd() {
  seeking.value = false
  window.removeEventListener('mousemove', onSeekMove)
  window.removeEventListener('mouseup', onSeekEnd)
}

// ---------- 全屏 ----------
// 手机全屏后锁定横屏，否则竖屏下画面只占屏幕中间一小块
async function lockLandscape() {
  const orientation = typeof screen !== 'undefined' ? screen.orientation : null
  if (!orientation?.lock) return
  try {
    await orientation.lock('landscape')
  } catch {
    // 桌面浏览器 / 系统不支持旋转锁定时忽略
  }
}

function unlockOrientation() {
  try {
    screen.orientation?.unlock?.()
  } catch {
    // 忽略
  }
}

async function toggleFullscreen() {
  const el = rootRef.value
  const video = videoRef.value
  if (!el) return
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen()
      return
    }
    if (el.requestFullscreen) {
      await el.requestFullscreen()
      await lockLandscape()
      return
    }
    // iOS Safari 不允许容器全屏，退回原生播放器（原生播放器可跟随设备旋转）
    if (video?.webkitEnterFullscreen) video.webkitEnterFullscreen()
  } catch {
    // 用户拒绝或被浏览器策略拦截时忽略
  }
}

function onFullscreenChange() {
  isFullscreen.value = document.fullscreenElement === rootRef.value
  if (!isFullscreen.value) unlockOrientation()
}

function download() {
  if (props.file) triggerUrl(downloadUrl(props.file.id))
}

// ---------- 快捷键 ----------
function onKeydown(event) {
  const tag = event.target?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || event.target?.isContentEditable) return
  const video = videoRef.value
  if (!video) return
  onActivity()
  switch (event.key) {
    case ' ':
      event.preventDefault()
      togglePlay()
      break
    case 'ArrowLeft':
      event.preventDefault()
      video.currentTime = Math.max(0, video.currentTime - 5)
      break
    case 'ArrowRight':
      event.preventDefault()
      video.currentTime = Math.min(duration.value || video.duration || 0, video.currentTime + 5)
      break
    case 'm':
    case 'M':
      toggleMute()
      break
    case 'f':
    case 'F':
      toggleFullscreen()
      break
    default:
      break
  }
}

watch(playing, (value) => {
  if (value) scheduleIdle()
  else {
    clearIdleTimer()
    idle.value = false
  }
})

watch(
  () => props.file?.id,
  () => {
    failed.value = false
    buffering.value = true
    playing.value = false
    duration.value = 0
    current.value = 0
    bufferedEnd.value = 0
    rate.value = 1
    volume.value = 1
    muted.value = false
    clearIdleTimer()
    idle.value = false
  }
)

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  document.addEventListener('fullscreenchange', onFullscreenChange)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  window.removeEventListener('mousemove', onSeekMove)
  window.removeEventListener('mouseup', onSeekEnd)
  clearIdleTimer()
  if (document.fullscreenElement === rootRef.value) {
    document.exitFullscreen?.().catch(() => {})
  }
  unlockOrientation()
})
</script>

<style scoped>
.video-player {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
/* 播放中长时间无操作：隐藏鼠标并淡出控制条 */
.video-player.is-idle,
.video-player.is-idle .video-player__stage {
  cursor: none;
}
.video-player.is-idle .video-player__dock {
  opacity: 0;
  pointer-events: none;
}
.video-player__stage {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
  cursor: pointer;
}
.video-player:fullscreen {
  background: #000;
}
.video-player:fullscreen .video-player__stage {
  border-radius: 0;
  box-shadow: none;
  height: 100%;
}
.video-player__video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
  display: block;
}
.video-player__hint {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 72px;
  height: 72px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.22);
  background: rgba(10, 14, 26, 0.55);
  color: #fff;
  font-size: 30px;
  cursor: pointer;
  transition: transform 160ms ease, background 160ms ease;
}
.video-player__hint:hover {
  background: rgba(43, 127, 255, 0.78);
  transform: translate(-50%, -50%) scale(1.06);
}
.video-player__loading {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #e5e7eb;
  pointer-events: none;
}
.spin {
  font-size: 30px;
  color: var(--nd-cyan);
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.video-player__fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #cbd5e1;
  font-size: 42px;
  padding: 24px;
  text-align: center;
}
.video-player__fallback p {
  font-size: 13px;
  color: #94a3b8;
}

/* 自绘控制条：原生控件在深色预览层里几乎不可见，改用固定可见的浮层 */
.video-player__dock {
  position: absolute;
  left: 14px;
  right: 14px;
  bottom: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: linear-gradient(180deg, rgba(10, 14, 26, 0.66) 0%, rgba(8, 11, 22, 0.92) 100%);
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.45);
  cursor: default;
  transition: opacity 240ms ease;
}
.video-player__dock button {
  width: 32px;
  height: 32px;
  flex: none;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: rgba(229, 231, 235, 0.92);
  font-size: 17px;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
}
.video-player__dock button:hover {
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
}
.dock__time,
.dock__total {
  flex: none;
  min-width: 38px;
  font-size: 11.5px;
  color: rgba(203, 213, 225, 0.85);
  text-align: center;
}
.dock__progress {
  position: relative;
  flex: 1;
  min-width: 60px;
  height: 4px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.2);
  cursor: pointer;
}
.dock__progress::after {
  content: '';
  position: absolute;
  inset: -9px 0;
}
.dock__buffer {
  position: absolute;
  inset: 0 auto 0 0;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.28);
}
.dock__played {
  position: absolute;
  inset: 0 auto 0 0;
  border-radius: 999px;
  background: linear-gradient(90deg, #2b7fff, #22d3ee);
}
.dock__knob {
  position: absolute;
  top: 50%;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #fff;
  transform: translate(-50%, -50%);
  box-shadow: 0 0 0 3px rgba(43, 127, 255, 0.35);
  opacity: 0;
  transition: opacity 140ms ease;
}
.video-player__dock:hover .dock__knob,
.dock__knob.is-on {
  opacity: 1;
}
.dock__rate {
  width: 42px;
  font-size: 11.5px;
  letter-spacing: 0.2px;
}
.video-player__dock .dock__rate {
  width: 42px;
  font-size: 11.5px;
}
.dock__volume {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: none;
}
.dock__slider {
  width: 64px;
  height: 4px;
  accent-color: #6366f1;
  cursor: pointer;
}
.is-off {
  color: #f87171;
}
@media (max-width: 900px) {
  .dock__volume {
    display: none;
  }
  .dock__time,
  .dock__total {
    min-width: 34px;
    font-size: 11px;
  }
}
@media (max-width: 767px) {
  .video-player__dock {
    left: 8px;
    right: 8px;
    bottom: 8px;
    gap: 4px;
    padding: 6px 8px;
  }
  .video-player__hint {
    width: 60px;
    height: 60px;
    font-size: 26px;
  }
}
</style>
