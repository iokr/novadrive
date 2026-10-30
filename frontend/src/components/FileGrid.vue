<template>
  <div class="grid-wrap">
    <div v-if="loading" class="file-grid">
      <div v-for="n in 8" :key="n" class="file-card is-skeleton">
        <el-skeleton animated>
          <template #template>
            <el-skeleton-item variant="rect" style="width: 100%; height: 96px; border-radius: 12px" />
            <el-skeleton-item variant="text" style="width: 70%; margin-top: 12px" />
            <el-skeleton-item variant="text" style="width: 45%" />
          </template>
        </el-skeleton>
      </div>
    </div>

    <EmptyState v-else-if="!items.length" :title="emptyTitle" :description="emptyDesc" />

    <div
      v-else
      class="file-grid"
      :class="{ 'is-plain-actions': plainActions }"
    >
      <div
        v-for="item in items"
        :key="item.id"
        class="file-card"
        :class="{ 'is-selected': selected.includes(item.id) }"
        @click="onCardClick(item)"
        @contextmenu.prevent="openMenuAt($event.clientX, $event.clientY, item)"
        @touchstart.passive="onTouchStart($event, item)"
        @touchend="clearPress"
        @touchmove="clearPress"
      >
        <button
          v-if="selectable"
          type="button"
          class="file-card__check"
          :class="{ 'is-on': selected.includes(item.id) }"
          @click.stop="$emit('toggle-select', item.id)"
        >
          <el-icon v-if="selected.includes(item.id)"><Check /></el-icon>
        </button>

        <div class="file-card__media" :class="{ 'is-video': kindOf(item) === 'video' }">
          <img
            v-if="showThumb(item)"
            :key="`${item.id}-${item.updated_at}`"
            :src="thumbSrc(item)"
            class="file-card__thumb"
            :alt="item.name"
            loading="lazy"
            @error="onThumbError(item)"
          />
          <el-icon v-else-if="kindOf(item) === 'video'" class="file-card__play"><VideoPlay /></el-icon>
          <FileAvatar v-else :item="item" size="lg" />
          <span v-if="kindOf(item) === 'video'" class="file-card__kind">视频</span>
        </div>

        <div class="file-card__body">
          <p class="file-card__name ellipsis" :title="item.name">{{ item.name }}</p>
          <p class="file-card__meta num">
            <span>{{ typeLabel(item) }}</span>
            <span class="dot">·</span>
            <span>{{ isDir(item) ? relativeTime(item.updated_at) : formatBytes(item.size) }}</span>
          </p>
        </div>

        <div class="file-card__actions" @click.stop>
          <button
            type="button"
            class="icon-btn"
            :title="Number(item.is_favorite) === 1 ? '取消收藏' : '收藏'"
            @click="$emit('toggle-favorite', item)"
          >
            <el-icon :class="{ 'is-star': Number(item.is_favorite) === 1 }">
              <StarFilled v-if="Number(item.is_favorite) === 1" />
              <Star v-else />
            </el-icon>
          </button>
          <button type="button" class="icon-btn" title="下载" @click="$emit('action', 'download', item)">
            <el-icon><Download /></el-icon>
          </button>
          <button type="button" class="icon-btn" title="更多" @click="openMenuAtButton($event, item)">
            <el-icon><MoreFilled /></el-icon>
          </button>
        </div>
      </div>
    </div>

    <FileActionMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :item="menu.item"
      :mode="mode"
      @action="(key, item) => $emit('action', key, item)"
      @close="closeMenu"
    />
  </div>
</template>

<script setup>
import { computed, reactive } from 'vue'
import EmptyState from './EmptyState.vue'
import FileActionMenu from './FileActionMenu.vue'
import FileAvatar from './FileAvatar.vue'
import { formatBytes, relativeTime } from '@/utils/format'
import { isDir, previewKind, typeLabel, extOf } from '@/utils/fileType'
import { thumbUrl, rawUrl } from '@/api/file'

const props = defineProps({
  items: { type: Array, default: () => [] },
  selected: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  mode: { type: String, default: 'default' },
  selectable: { type: Boolean, default: true },
  emptyTitle: { type: String, default: '还没有文件' },
  emptyDesc: { type: String, default: '拖拽文件到此处，或点击右上角「上传」开始' }
})

const emit = defineEmits(['open', 'action', 'toggle-select', 'toggle-favorite'])

const menu = reactive({ visible: false, x: 0, y: 0, item: null })

// 可多选的常规文件列表（我的文件）：手机端卡片不放收藏 / 下载 / 更多，操作走底部操作栏
const plainActions = computed(() => props.selectable && props.mode === 'default')

const IMAGE_EXT = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg', 'avif']
const failedThumbs = reactive(new Set())
const failedRaw = reactive(new Set())

// 卡片用大图区分目录 / 视频 / 图片
function kindOf(item) {
  return isDir(item) ? 'folder' : previewKind(item)
}

function showThumb(item) {
  if (isDir(item) || failedRaw.has(item.id)) return false
  if (item.has_thumb) return true
  // 服务端异步生成缩略图，未就绪时先用原图兜底
  return IMAGE_EXT.includes(extOf(item))
}

function thumbSrc(item) {
  return item.has_thumb && !failedThumbs.has(item.id) ? thumbUrl(item.id) : rawUrl(item.id)
}

// 缩略图失败退回原图，原图也失败则交给 FileAvatar 画图标
function onThumbError(item) {
  if (item.has_thumb && !failedThumbs.has(item.id)) failedThumbs.add(item.id)
  else failedRaw.add(item.id)
}

function onCardClick(item) {
  // 长按已触发选择，吞掉随之而来的 click，避免刚选中又被点掉
  if (pressed) {
    pressed = false
    return
  }
  emit('open', item)
}

function openMenuAt(x, y, item) {
  menu.visible = true
  menu.x = x
  menu.y = y
  menu.item = item
}

function openMenuAtButton(event, item) {
  const rect = event.currentTarget.getBoundingClientRect()
  openMenuAt(rect.left, rect.bottom + 6, item)
}

function closeMenu() {
  menu.visible = false
  menu.item = null
}

// 移动端长按 500ms 进入选择态（与主流网盘一致），进入后点卡片继续多选
let pressTimer = null
let pressed = false
function onTouchStart(event, item) {
  clearPress()
  const touch = event.touches?.[0]
  if (!touch) return
  pressed = false
  pressTimer = setTimeout(() => {
    pressTimer = null
    pressed = true
    // 已选中的卡片长按不再切换，避免误退出选择态
    if (props.selected.includes(item.id)) return
    if (navigator.vibrate) navigator.vibrate(12)
    emit('toggle-select', item.id)
  }, 500)
}
function clearPress() {
  if (pressTimer) {
    clearTimeout(pressTimer)
    pressTimer = null
  }
}
</script>

<style scoped>
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(184px, 1fr));
  gap: 14px;
}

.file-card {
  position: relative;
  padding: 12px;
  border-radius: var(--nd-radius-card);
  background: var(--nd-surface);
  border: 1px solid var(--nd-border);
  cursor: pointer;
  transition: transform 180ms ease, box-shadow 180ms ease, border-color 180ms ease;
  overflow: hidden;
  user-select: none;
}
.file-card.is-selected {
  border-color: var(--nd-primary);
  background: var(--nd-primary-soft);
}
.file-card.is-skeleton {
  cursor: default;
  transform: none;
  box-shadow: none;
}

.file-card__check {
  position: absolute;
  top: 10px;
  left: 10px;
  z-index: 2;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  border: 1px solid var(--nd-border-strong);
  background: var(--nd-surface);
  color: #fff;
  display: grid;
  place-items: center;
  cursor: pointer;
  opacity: 0;
  transition: opacity 160ms ease, background 160ms ease;
}
.file-card.is-selected .file-card__check {
  opacity: 1;
}
.file-card__check.is-on {
  background: var(--nd-primary);
  border-color: var(--nd-primary);
}

.file-card__media {
  position: relative;
  height: 104px;
  border-radius: 8px;
  background: var(--nd-surface-sunken);
  display: grid;
  place-items: center;
  overflow: hidden;
}
.file-card__media.is-video {
  background: linear-gradient(155deg, #46557a 0%, #1d2637 100%);
}
.file-card__thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.file-card__play {
  font-size: 36px;
  color: #fff;
  opacity: 0.92;
  filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.35));
}
.file-card__kind {
  position: absolute;
  right: 8px;
  bottom: 8px;
  font-size: 11px;
  line-height: 18px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.9);
  color: #1f2329;
}

.file-card__body {
  margin-top: 10px;
}
.file-card__name {
  font-size: 13.5px;
  font-weight: 560;
}
.file-card__meta {
  margin-top: 4px;
  font-size: 11.5px;
  color: var(--nd-text-secondary);
  display: flex;
  align-items: center;
  gap: 5px;
}
.dot {
  opacity: 0.5;
}

.file-card__actions {
  position: absolute;
  right: 8px;
  top: 8px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transform: translateY(-4px);
  transition: opacity 160ms ease, transform 160ms ease;
}
.icon-btn {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
  color: var(--nd-text-secondary);
  display: grid;
  place-items: center;
  cursor: pointer;
  font-size: 13px;
}
.is-star {
  color: #f7ba1e;
}

/* 只在真正支持 hover 的设备上做悬停反馈：
   触摸设备点按会触发 hover，描边 / 阴影 / 图标条位移会随开合菜单来回闪 */
@media (hover: hover) and (pointer: fine) {
  .file-card:hover {
    border-color: #c4ddff;
    box-shadow: 0 10px 26px rgba(43, 127, 255, 0.14);
  }
  .file-card:hover .file-card__check {
    opacity: 1;
  }
  .file-card:hover .file-card__actions {
    opacity: 1;
    transform: translateY(0);
  }
  .icon-btn:hover {
    background: var(--nd-primary-soft);
    border-color: #c4ddff;
    color: var(--nd-primary);
  }
}

@media (max-width: 767px) {
  .file-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 10px;
  }
  .file-card {
    padding: 10px;
  }
  .file-card__media {
    height: 88px;
  }
  /* 手机端勾选框收小成细描边圆形：视觉不压过封面，热区仍保留手指大小 */
  .file-card__check {
    top: 8px;
    left: 8px;
    width: 17px;
    height: 17px;
    border-radius: 50%;
    border-width: 1.5px;
    background: rgba(255, 255, 255, 0.92);
    font-size: 9px;
    opacity: 1;
  }
  .file-card__check::after {
    content: '';
    position: absolute;
    inset: -8px;
  }
  .file-card__check .el-icon {
    font-size: 9px;
  }
  .file-card__check.is-on {
    background: var(--nd-primary);
    border-color: var(--nd-primary);
  }
  /* 触摸端图标条常驻且不位移，避免点按时上下跳 */
  .file-card__actions {
    transform: none;
  }
  /* 我的文件页不常驻收藏 / 下载 / 更多：选中后统一从底部操作栏操作 */
  .file-grid.is-plain-actions .file-card__actions {
    display: none;
  }
}
</style>
