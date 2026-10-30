<template>
  <Teleport to="body">
    <div v-if="visible" class="fam-mask" @click="close" @contextmenu.prevent="close"></div>
    <transition name="fam">
      <div v-if="visible" class="fam glass-card" :style="positionStyle">
        <button
          v-for="act in actions"
          :key="act.key"
          type="button"
          class="fam__item"
          :class="{ 'is-danger': act.danger }"
          @click="pick(act.key)"
        >
          <el-icon><component :is="act.icon" /></el-icon>
          <span>{{ act.label }}</span>
        </button>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { canPreview, isDir } from '@/utils/fileType'

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  item: { type: Object, default: null },
  // default（文件列表） / favorites（收藏页） / recycle（回收站）
  mode: { type: String, default: 'default' }
})

const emit = defineEmits(['action', 'close'])

const positionStyle = computed(() => {
  const width = 190
  // 注意：actions 是本地 computed，不是 props，漏写 .value 会让菜单渲染直接抛错
  const height = Math.min(actions.value.length * 40 + 12, 360)
  const left = Math.min(props.x, window.innerWidth - width - 12)
  const top = Math.min(props.y, window.innerHeight - height - 12)
  return { left: `${Math.max(8, left)}px`, top: `${Math.max(8, top)}px` }
})

const actions = computed(() => {
  const item = props.item
  if (!item) return []
  const dir = isDir(item)

  if (props.mode === 'recycle') {
    return [
      { key: 'restore', label: '还原', icon: 'RefreshLeft' },
      { key: 'forceDelete', label: '彻底删除', icon: 'Delete', danger: true }
    ]
  }

  const list = []
  if (dir) {
    list.push({ key: 'open', label: '打开', icon: 'FolderOpened' })
  } else if (canPreview(item)) {
    list.push({ key: 'preview', label: '预览', icon: 'View' })
  }
  list.push({ key: 'download', label: dir ? '下载为 zip' : '下载', icon: 'Download' })

  if (props.mode !== 'favorites') {
    list.push({
      key: 'favorite',
      label: Number(item.is_favorite) === 1 ? '取消收藏' : '收藏',
      icon: Number(item.is_favorite) === 1 ? 'StarFilled' : 'Star'
    })
  } else {
    list.push({ key: 'favorite', label: '取消收藏', icon: 'StarFilled' })
  }

  list.push({ key: 'share', label: '分享', icon: 'Share' })
  list.push({ key: 'rename', label: '重命名', icon: 'EditPen' })
  list.push({ key: 'move', label: '移动到…', icon: 'Rank' })
  list.push({ key: 'copy', label: '复制到…', icon: 'CopyDocument' })
  list.push({ key: 'delete', label: '删除', icon: 'Delete', danger: true })
  return list
})

function pick(key) {
  emit('action', key, props.item)
  close()
}

function close() {
  emit('close')
}
</script>

<style scoped>
.fam-mask {
  position: fixed;
  inset: 0;
  z-index: 2000;
}
.fam {
  position: fixed;
  z-index: 2001;
  width: 190px;
  padding: 6px;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--nd-border);
  border-radius: 12px;
  box-shadow: var(--nd-shadow);
  backdrop-filter: blur(18px);
}
.fam__item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border: none;
  background: transparent;
  color: var(--nd-text);
  font-size: 13px;
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
}
.fam__item:hover {
  background: var(--nd-surface-hover);
}
.fam__item.is-danger {
  color: #f87171;
}
.fam-enter-active,
.fam-leave-active {
  transition: opacity 120ms ease, transform 120ms ease;
}
.fam-enter-from,
.fam-leave-to {
  opacity: 0;
  transform: scale(0.96);
}

/* 触摸端：毛玻璃与缩放动画在固定定位层上容易产生重绘闪烁，改为不透明背景 + 纯淡入 */
@media (hover: none), (pointer: coarse) {
  .fam {
    background: var(--el-bg-color-overlay, #fff);
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
  }
  .fam__item:hover {
    background: transparent;
  }
  .fam__item:active {
    background: var(--nd-surface-hover);
  }
  .fam-enter-active,
  .fam-leave-active {
    transition: opacity 100ms ease;
  }
  .fam-enter-from,
  .fam-leave-to {
    transform: none;
  }
}
</style>
