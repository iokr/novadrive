<template>
  <span class="fa" :class="[`fa--${kind}`, `fa--${size}`]">
    <img
      v-if="showThumb"
      class="fa__thumb"
      :src="thumbSrc"
      :alt="item?.name || ''"
      loading="lazy"
      @error="onError"
    />
    <span v-else-if="kind === 'video'" class="fa__play">
      <el-icon><CaretRight /></el-icon>
    </span>
    <svg v-else-if="kind === 'folder'" class="fa__folder" viewBox="0 0 40 32" aria-hidden="true">
      <path d="M2 7.8A4 4 0 0 1 6 3.8h7.4c1 0 2 .4 2.7 1.2l2.4 2.6H2Z" fill="#ffb020" />
      <rect x="2" y="7" width="36" height="22" rx="3.6" fill="#ffc93c" />
    </svg>
    <el-icon v-else class="fa__glyph"><component :is="iconOf(item)" /></el-icon>
  </span>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { extOf, iconOf, isDir, previewKind } from '@/utils/fileType'
import { rawUrl, thumbUrl } from '@/api/file'

const props = defineProps({
  item: { type: Object, default: () => ({}) },
  // sm: 32px 紧凑、md: 40px 表格、lg: 56px 卡片
  size: { type: String, default: 'md' }
})

const IMAGE_EXT = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg', 'avif']

const failedThumb = ref(false)
const failedRaw = ref(false)
const kind = computed(() => (isDir(props.item) ? 'folder' : previewKind(props.item)))

// 服务端只给图片生成缩略图，视频/文档统一走占位图形
const isImageKind = computed(() => kind.value === 'image')
const canThumb = computed(
  () => isImageKind.value && (Boolean(props.item?.has_thumb) || IMAGE_EXT.includes(extOf(props.item)))
)
const showThumb = computed(() => canThumb.value && !failedRaw.value)
// 缩略图 404 时退回原图，原图也失败才用图标占位
const thumbSrc = computed(() =>
  props.item?.has_thumb && !failedThumb.value ? thumbUrl(props.item.id) : rawUrl(props.item.id)
)

function onError() {
  if (props.item?.has_thumb && !failedThumb.value) failedThumb.value = true
  else failedRaw.value = true
}

// 同一组件复用在不同行时，切换文件要重置失败状态
watch(
  () => props.item?.id,
  () => {
    failedThumb.value = false
    failedRaw.value = false
  }
)
</script>

<style scoped>
.fa {
  display: grid;
  place-items: center;
  flex: none;
  border-radius: 8px;
  overflow: hidden;
  color: #7c8798;
  background: #f1f3f7;
}
.fa--sm {
  width: 32px;
  height: 32px;
  font-size: 16px;
}
.fa--md {
  width: 40px;
  height: 40px;
  font-size: 19px;
}
.fa--lg {
  width: 56px;
  height: 56px;
  font-size: 26px;
}

.fa__thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* 文件夹：无底板，直接呈现黄色文件夹，与网盘一致 */
.fa--folder {
  background: transparent;
  overflow: visible;
}
.fa__folder {
  width: 82%;
  height: auto;
  display: block;
  filter: drop-shadow(0 2px 4px rgba(31, 35, 41, 0.12));
}

/* 视频：深色胶片底 + 播放键，列表里一眼可辨 */
.fa--video {
  background: linear-gradient(155deg, #46557a 0%, #1d2637 100%);
  color: #fff;
}
.fa__play {
  display: grid;
  place-items: center;
  width: 58%;
  height: 58%;
  border-radius: 50%;
  background: #fff;
  color: #16203a;
  font-size: 0.8em;
  padding-left: 1px;
}

.fa--image {
  background: #e8f6fd;
  color: #2aa3d8;
}
.fa--pdf {
  background: #ffefed;
  color: #f5544a;
}
.fa--audio {
  background: #f5edff;
  color: #a855f7;
}
.fa--text {
  background: #eaf8ef;
  color: #16a34a;
}
.fa--other {
  background: #f1f3f7;
  color: #7c8798;
}
</style>
