<template>
  <div class="audio-player">
    <div class="audio-player__cover" :style="{ '--cover-seed': seed }">
      <el-icon class="audio-player__note"><Headset /></el-icon>
      <span class="audio-player__ext">{{ extText }}</span>
    </div>

    <p class="audio-player__name ellipsis" :title="file?.name">{{ file?.name }}</p>
    <p class="audio-player__meta num text-secondary">
      {{ formatBytes(file?.size) }} · {{ extText }} 音频
    </p>

    <audio
      ref="audioRef"
      :src="src"
      class="audio-player__audio"
      controls
      preload="metadata"
      @error="failed = true"
    />

    <p v-if="failed" class="audio-player__error">当前浏览器无法播放该音频格式，请下载后播放</p>

    <div class="audio-player__actions">
      <div class="audio-player__rates">
        <button
          v-for="rate in rates"
          :key="rate"
          type="button"
          :class="{ 'is-on': playbackRate === rate }"
          @click="setRate(rate)"
        >
          {{ rate }}x
        </button>
      </div>
      <el-button text size="small" @click="download">
        <el-icon><Download /></el-icon>
        <span>下载</span>
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { rawUrl, downloadUrl } from '@/api/file'
import { downloadUrl as triggerUrl } from '@/utils/download'
import { extOf } from '@/utils/fileType'
import { formatBytes } from '@/utils/format'

const props = defineProps({
  file: { type: Object, default: null }
})

const rates = [0.75, 1, 1.25, 1.5, 2]

const audioRef = ref(null)
const failed = ref(false)
const playbackRate = ref(1)

const src = computed(() => (props.file ? rawUrl(props.file.id) : ''))
const extText = computed(() => (extOf(props.file) || 'AUDIO').toUpperCase())
// 用文件名生成稳定的色相，让封面略有区分
const seed = computed(() => {
  const name = props.file?.name || ''
  let hash = 0
  for (let i = 0; i < name.length; i += 1) hash = (hash * 31 + name.charCodeAt(i)) % 360
  return `${hash}deg`
})

watch(
  () => props.file?.id,
  () => {
    failed.value = false
    playbackRate.value = 1
  }
)

function setRate(rate) {
  playbackRate.value = rate
  if (audioRef.value) audioRef.value.playbackRate = rate
}

function download() {
  if (props.file) triggerUrl(downloadUrl(props.file.id))
}
</script>

<style scoped>
.audio-player {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 24px;
}
.audio-player__cover {
  width: 168px;
  height: 168px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  position: relative;
  background:
    radial-gradient(circle at 30% 25%, rgba(255, 255, 255, 0.35), transparent 55%),
    conic-gradient(from var(--cover-seed, 0deg), #2b7fff, #22d3ee, #a78bfa, #2b7fff);
  box-shadow: 0 24px 60px rgba(43, 127, 255, 0.28);
  animation: breathe 6s ease-in-out infinite;
}
.audio-player__note {
  font-size: 56px;
  color: rgba(255, 255, 255, 0.92);
  filter: drop-shadow(0 6px 14px rgba(3, 7, 18, 0.35));
}
.audio-player__ext {
  position: absolute;
  bottom: 16px;
  font-size: 10.5px;
  letter-spacing: 1.2px;
  color: rgba(255, 255, 255, 0.9);
}
.audio-player__name {
  max-width: 480px;
  font-size: 15px;
  font-weight: 600;
  text-align: center;
}
.audio-player__meta {
  font-size: 12px;
}
.audio-player__audio {
  width: min(520px, 100%);
  margin-top: 4px;
}
.audio-player__error {
  font-size: 12px;
  color: #f87171;
}
.audio-player__actions {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 2px;
}
.audio-player__rates {
  display: flex;
  gap: 4px;
}
.audio-player__rates button {
  min-width: 40px;
  height: 24px;
  padding: 0 6px;
  border-radius: 8px;
  border: 1px solid var(--nd-border);
  background: transparent;
  color: var(--nd-text-secondary);
  font-size: 11.5px;
  cursor: pointer;
}
.audio-player__rates button:hover,
.audio-player__rates button.is-on {
  border-color: var(--nd-primary);
  color: var(--nd-text);
  background: var(--nd-primary-soft);
}
@keyframes breathe {
  0%,
  100% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.03);
  }
}
@media (max-width: 767px) {
  .audio-player__cover {
    width: 132px;
    height: 132px;
  }
  .audio-player__note {
    font-size: 44px;
  }
}
</style>
