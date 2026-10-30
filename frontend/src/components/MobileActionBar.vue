<template>
  <transition name="mab">
    <div v-if="visible && count > 0" class="mab">
      <p v-if="count > 1" class="mab__hint">已选 {{ count }} 项，分享 / 重命名 / 移动等仅适用于单个文件</p>
      <div class="mab__grid">
        <button
          v-for="act in actions"
          :key="act.key"
          type="button"
          class="mab__item"
          :class="{ 'is-danger': act.danger, 'is-disabled': act.disabled }"
          :disabled="act.disabled"
          @click="$emit('action', act.key)"
        >
          <el-icon><component :is="act.icon" /></el-icon>
          <span>{{ act.label }}</span>
        </button>
      </div>
    </div>
  </transition>
</template>

<script setup>
import { computed } from 'vue'

// 手机端选择态底部操作栏：选中文件后从底部弹出，替代桌面端顶部批量条
const props = defineProps({
  visible: { type: Boolean, default: false },
  count: { type: Number, default: 0 },
  // 单选时的文件对象，多选传 null：仅下载 / 删除支持批量
  item: { type: Object, default: null }
})

defineEmits(['action'])

const actions = computed(() => {
  const multi = props.count > 1
  const favorited = Number(props.item?.is_favorite) === 1
  return [
    { key: 'download', label: multi ? '打包下载' : '下载', icon: 'Download' },
    { key: 'share', label: '分享', icon: 'Share', disabled: multi },
    { key: 'delete', label: '删除', icon: 'Delete', danger: true },
    {
      key: 'favorite',
      label: favorited ? '取消收藏' : '收藏',
      icon: favorited ? 'StarFilled' : 'Star',
      disabled: multi
    },
    { key: 'rename', label: '重命名', icon: 'EditPen', disabled: multi },
    { key: 'move', label: '移动', icon: 'Rank', disabled: multi },
    { key: 'copy', label: '复制', icon: 'CopyDocument', disabled: multi },
    { key: 'details', label: '详情', icon: 'InfoFilled', disabled: multi }
  ]
})
</script>

<style scoped>
/* 只在手机宽度出现，桌面端仍用顶部批量操作条 */
@media (min-width: 768px) {
  .mab {
    display: none;
  }
}
.mab {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1400;
  padding: 6px 8px calc(6px + env(safe-area-inset-bottom));
  background: var(--nd-surface);
  border-top: 1px solid var(--nd-border);
  box-shadow: 0 -8px 24px rgba(31, 35, 41, 0.08);
}
.mab__hint {
  padding: 2px 6px 4px;
  font-size: 12px;
  color: var(--nd-text-muted);
}
.mab__grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 2px;
}
.mab__item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  padding: 8px 2px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--nd-text);
  font-size: 11.5px;
  cursor: pointer;
}
.mab__item .el-icon {
  font-size: 20px;
}
.mab__item.is-danger {
  color: #f87171;
}
.mab__item.is-disabled {
  opacity: 0.32;
  cursor: default;
}
/* 触摸端只做按下反馈，避免 hover 命中后残留高亮 */
.mab__item:active {
  background: var(--nd-surface-hover);
}
.mab__item.is-disabled:active {
  background: transparent;
}
.mab-enter-active,
.mab-leave-active {
  transition: transform 200ms ease, opacity 200ms ease;
}
.mab-enter-from,
.mab-leave-to {
  opacity: 0;
  transform: translateY(100%);
}
</style>
