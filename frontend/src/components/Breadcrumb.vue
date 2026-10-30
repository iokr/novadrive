<template>
  <nav class="crumb">
    <button class="crumb__item" type="button" @click="$emit('navigate', 0)">
      <el-icon><HomeFilled /></el-icon>
      <span>我的文件</span>
    </button>
    <template v-for="node in items" :key="node.id">
      <el-icon class="crumb__sep"><ArrowRight /></el-icon>
      <button
        class="crumb__item"
        type="button"
        :class="{ 'is-current': node.id === current }"
        @click="$emit('navigate', node.id)"
      >
        <span class="crumb__text">{{ node.name }}</span>
      </button>
    </template>
  </nav>
</template>

<script setup>
defineProps({
  items: { type: Array, default: () => [] },
  current: { type: Number, default: 0 }
})
defineEmits(['navigate'])
</script>

<style scoped>
.crumb {
  display: flex;
  align-items: center;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
  padding: 2px 0;
}
.crumb::-webkit-scrollbar {
  display: none;
}
.crumb__item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 220px;
  padding: 4px 8px;
  border: none;
  background: transparent;
  color: var(--nd-text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: background 160ms ease, color 160ms ease;
}
.crumb__item:hover {
  background: var(--nd-primary-soft);
  color: var(--nd-primary);
}
.crumb__item.is-current {
  color: var(--nd-text);
  font-weight: 600;
}
.crumb__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.crumb__sep {
  color: var(--nd-text-muted);
  font-size: 12px;
  flex: none;
}
</style>
