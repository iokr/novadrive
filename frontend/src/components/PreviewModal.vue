<template>
  <Teleport to="body">
    <Transition name="pv-fade">
      <div
        v-if="modelValue"
        ref="layerRef"
        class="pv"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
      >
        <header class="pv__bar">
          <div class="pv__ident">
            <span class="pv__badge" :style="{ color: tint, background: `${tint}22` }">
              <el-icon><component :is="iconName" /></el-icon>
            </span>
            <div class="pv__ident-text">
              <p class="pv__name ellipsis" :title="file?.name">{{ file?.name || '预览' }}</p>
              <p class="pv__meta num">
                <span class="pv__tag">{{ kindLabel }}</span>
                <span v-if="file?.size">{{ formatBytes(file.size) }}</span>
                <span v-if="file?.updated_at">{{ relativeTime(file.updated_at) }}</span>
              </p>
            </div>
          </div>
          <div class="pv__actions">
            <button type="button" class="pv__action" title="下载" @click="$emit('download', file)">
              <el-icon><Download /></el-icon>
            </button>
            <button type="button" class="pv__action pv__action--close" title="关闭（Esc）" @click="close">
              <el-icon><Close /></el-icon>
            </button>
          </div>
        </header>

        <main class="pv__stage" @click.self="close">
          <div class="pv__frame" :class="`is-${kind}`">
            <ImageViewer
              v-if="kind === 'image'"
              :files="imageFiles"
              :index="imageIndex"
              @update:index="onImageIndex"
            />
            <PdfViewer v-else-if="kind === 'pdf'" :file="file" />
            <VideoPlayer v-else-if="kind === 'video'" :file="file" />
            <AudioPlayer v-else-if="kind === 'audio'" :file="file" />
            <TextViewer v-else-if="kind === 'text'" :file="file" />
            <div v-else class="pv-fallback">
              <el-icon class="pv-fallback__icon"><Document /></el-icon>
              <p>该文件类型暂不支持在线预览</p>
              <el-button type="primary" @click="$emit('download', file)">下载文件</el-button>
            </div>
          </div>
        </main>

        <p class="pv__hint">
          <template v-if="kind === 'image'">滚轮缩放 · 拖拽平移 · ← → 切换 · </template>
          <template v-else-if="kind === 'pdf'">滚动查看全部页面 · </template>
          <template v-else-if="kind === 'video'">空格 播放/暂停 · ← → 快进 5 秒 · 全屏后可横屏 · </template>
          Esc 关闭
        </p>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import ImageViewer from './ImageViewer.vue'
import PdfViewer from './PdfViewer.vue'
import VideoPlayer from './VideoPlayer.vue'
import AudioPlayer from './AudioPlayer.vue'
import TextViewer from './TextViewer.vue'
import { previewKind, isImage, iconOf, tintOf } from '@/utils/fileType'
import { formatBytes, relativeTime } from '@/utils/format'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  file: { type: Object, default: null },
  // 当前目录内的图片集合，用于左右切换
  siblings: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue', 'download', 'change'])

const KIND_LABELS = {
  image: '图片',
  pdf: 'PDF 文档',
  video: '视频',
  audio: '音频',
  text: '文本',
  other: '文件'
}

const layerRef = ref(null)

const kind = computed(() => previewKind(props.file))
const kindLabel = computed(() => KIND_LABELS[kind.value] || '文件')
const iconName = computed(() => iconOf(props.file))
const tint = computed(() => tintOf(props.file))

// 只有图片才需要在同类型列表中切换
const imageFiles = computed(() => props.siblings.filter((it) => isImage(it)))

const imageIndex = ref(0)

watch(
  () => [props.file, props.modelValue],
  () => {
    if (!props.modelValue || !props.file) return
    const idx = imageFiles.value.findIndex((it) => it.id === props.file.id)
    imageIndex.value = idx >= 0 ? idx : 0
  },
  { immediate: true }
)

// 打开时锁定页面滚动并把焦点交给预览层，便于 Esc 关闭
watch(
  () => props.modelValue,
  (open) => {
    if (typeof document === 'undefined') return
    document.body.style.overflow = open ? 'hidden' : ''
    if (open) requestAnimationFrame(() => layerRef.value?.focus())
  }
)

function close() {
  emit('update:modelValue', false)
}

function onImageIndex(next) {
  imageIndex.value = next
  const target = imageFiles.value[next]
  // 通知父级切换当前预览项，便于标题信息同步
  if (target) emit('change', target)
}

function onKeydown(event) {
  if (event.key !== 'Escape' || !props.modelValue) return
  // 原生全屏时 Esc 先交给浏览器退出全屏，不关闭预览层
  if (typeof document !== 'undefined' && document.fullscreenElement) return
  close()
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<style scoped>
/*
 * 预览层跟随主题：浅色主题用白底 / 深字，深色主题保留沉浸黑。
 * 子查看器统一消费 --pv-* 变量，不再写死深色配色。
 */
.pv {
  --pv-bg:
    radial-gradient(1200px 620px at 50% -12%, rgba(43, 127, 255, 0.1) 0%, transparent 62%),
    radial-gradient(900px 500px at 92% 108%, rgba(34, 211, 238, 0.12) 0%, transparent 60%),
    #f5f7fa;
  --pv-bar-bg: linear-gradient(180deg, rgba(245, 247, 250, 0.96) 0%, rgba(245, 247, 250, 0) 100%);
  --pv-text: var(--nd-text);
  --pv-text-secondary: var(--nd-text-secondary);
  --pv-text-muted: var(--nd-text-muted);
  --pv-border: var(--nd-border-strong);
  --pv-chip: var(--nd-surface);
  --pv-chip-hover: var(--nd-surface-hover);
  --pv-nav-bg: rgba(255, 255, 255, 0.92);
  --pv-dock-bg: rgba(255, 255, 255, 0.94);
  --pv-shadow: var(--nd-shadow);
  --pv-code-bg: var(--nd-surface-sunken);
  --pv-code-text: var(--nd-text);
  --pv-image-shadow: 0 18px 46px rgba(31, 35, 41, 0.16);

  /* 同步覆盖 Element Plus 变量，避免查看器内按钮 / 输入框跟随全局默认值 */
  --el-bg-color: #ffffff;
  --el-bg-color-overlay: #ffffff;
  --el-fill-color-blank: transparent;
  --el-border-color: var(--nd-border);
  --el-text-color-primary: var(--nd-text);
  --el-text-color-regular: var(--nd-text-secondary);

  position: fixed;
  inset: 0;
  z-index: 2100;
  display: flex;
  flex-direction: column;
  color: var(--pv-text);
  outline: none;
  background: var(--pv-bg);
}

/* 深色主题：恢复沉浸式深色预览 */
html.dark .pv {
  --pv-bg:
    radial-gradient(1200px 620px at 50% -12%, rgba(99, 102, 241, 0.22) 0%, transparent 62%),
    radial-gradient(900px 500px at 92% 108%, rgba(34, 211, 238, 0.14) 0%, transparent 60%),
    rgba(5, 8, 17, 0.94);
  --pv-bar-bg: linear-gradient(180deg, rgba(5, 8, 17, 0.9) 0%, rgba(5, 8, 17, 0) 100%);
  --pv-text: #e5e7eb;
  --pv-text-secondary: #9ca3af;
  --pv-text-muted: #6b7280;
  --pv-border: rgba(255, 255, 255, 0.12);
  --pv-chip: rgba(255, 255, 255, 0.06);
  --pv-chip-hover: rgba(255, 255, 255, 0.1);
  --pv-nav-bg: rgba(17, 22, 38, 0.62);
  --pv-dock-bg: rgba(15, 20, 35, 0.74);
  --pv-shadow: 0 18px 40px rgba(0, 0, 0, 0.45);
  --pv-code-bg: rgba(3, 7, 18, 0.35);
  --pv-code-text: #cbd5e1;
  --pv-image-shadow: 0 26px 60px rgba(0, 0, 0, 0.5);

  --el-bg-color: #131a2e;
  --el-bg-color-overlay: #182041;
  --el-border-color: rgba(255, 255, 255, 0.12);
  --el-text-color-primary: #e5e7eb;
  --el-text-color-regular: #cbd5e1;
}

/* 顶部信息条：用渐变蒙层托底，避免文件名压在亮色图片上 */
.pv__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 20px 20px;
  background: var(--pv-bar-bg);
  flex: none;
}

.pv__ident {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.pv__badge {
  width: 38px;
  height: 38px;
  flex: none;
  display: grid;
  place-items: center;
  border-radius: 12px;
  font-size: 19px;
  border: 1px solid var(--pv-border);
}

.pv__ident-text {
  min-width: 0;
}

.pv__name {
  font-size: 14.5px;
  font-weight: 600;
  letter-spacing: 0.2px;
  color: var(--pv-text);
}

.pv__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 3px;
  font-size: 11.5px;
  color: var(--pv-text-secondary);
}

.pv__meta > span + span::before {
  content: '·';
  margin-right: 8px;
  opacity: 0.6;
}

.pv__tag {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--pv-chip);
  border: 1px solid var(--pv-border);
  font-size: 11px;
  color: var(--pv-text-secondary);
}

.pv__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: none;
}

.pv__action {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 11px;
  border: 1px solid var(--pv-border);
  background: var(--pv-chip);
  color: var(--pv-text-secondary);
  font-size: 16px;
  cursor: pointer;
  transition: background 160ms ease, border-color 160ms ease, color 160ms ease, transform 160ms ease;
}

.pv__action:hover {
  background: var(--nd-primary);
  border-color: var(--nd-primary);
  color: #ffffff;
}

.pv__action--close:hover {
  background: rgba(248, 113, 113, 0.28);
  border-color: rgba(248, 113, 113, 0.45);
}

.pv__stage {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
}

.pv__frame {
  width: min(1720px, 100%);
  height: 100%;
  min-height: 0;
  display: flex;
}

/* PDF / 文本自带页面底色，用一个圆角面板收住内容 */
.pv__frame.is-pdf,
.pv__frame.is-text {
  border-radius: 18px;
  border: 1px solid var(--pv-border);
  background: var(--nd-surface);
  box-shadow: var(--pv-shadow);
  overflow: hidden;
}

.pv-fallback {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  color: var(--pv-text-secondary);
  font-size: 13px;
}

.pv-fallback__icon {
  font-size: 54px;
  color: var(--nd-primary);
}

.pv__hint {
  flex: none;
  padding: 8px 24px 12px;
  text-align: center;
  font-size: 11.5px;
  letter-spacing: 0.3px;
  color: var(--pv-text-muted);
}

.pv-fade-enter-active,
.pv-fade-leave-active {
  transition: opacity 200ms ease;
}

.pv-fade-enter-active .pv__bar,
.pv-fade-enter-active .pv__frame {
  transition: transform 260ms cubic-bezier(0.22, 1, 0.36, 1);
}

.pv-fade-enter-from,
.pv-fade-leave-to {
  opacity: 0;
}

.pv-fade-enter-from .pv__bar {
  transform: translateY(-12px);
}

.pv-fade-enter-from .pv__frame {
  transform: scale(0.985);
}

@media (max-width: 767px) {
  .pv__bar {
    padding: 10px 12px 20px;
  }
  .pv__badge {
    width: 34px;
    height: 34px;
    font-size: 17px;
  }
  .pv__name {
    font-size: 13.5px;
  }
  .pv__meta {
    font-size: 11px;
  }
  .pv__stage {
    padding: 0 8px;
  }
  .pv__hint {
    padding-bottom: 10px;
  }
}

/* 手机横屏：隐藏底部提示，顶部信息改成悬浮蒙层，画面占满屏幕 */
@media (orientation: landscape) and (max-height: 560px) {
  .pv__bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 3;
    padding: 8px 12px 18px;
    background: var(--pv-bar-bg);
  }
  .pv__hint {
    display: none;
  }
  .pv__stage {
    padding: 0;
  }
  .pv__frame.is-pdf,
  .pv__frame.is-text {
    border-radius: 0;
    border: none;
  }
}
</style>
