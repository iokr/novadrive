<template>
  <div class="nd-page">
    <div class="nd-page__header">
      <div>
        <p class="nd-page__title">最近访问</p>
        <p class="nd-page__subtitle num">最近打开过的 {{ items.length }} 个文件，最多保留 50 条</p>
      </div>
      <div class="head-tools">
        <div class="view-switch">
          <button type="button" :class="{ 'is-on': fileStore.viewMode === 'grid' }" @click="fileStore.setViewMode('grid')">
            <el-icon><Grid /></el-icon>
          </button>
          <button type="button" :class="{ 'is-on': fileStore.viewMode === 'list' }" @click="fileStore.setViewMode('list')">
            <el-icon><Menu /></el-icon>
          </button>
        </div>
        <el-button :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          <span>刷新</span>
        </el-button>
      </div>
    </div>

    <FileGrid
      v-if="fileStore.viewMode === 'grid'"
      :items="items"
      :loading="loading"
      :selectable="false"
      empty-title="最近还没有访问记录"
      empty-desc="打开任意文件后，这里会记录最近访问时间"
      @open="onOpen"
      @action="onAction"
      @toggle-favorite="actions.toggleFavorite"
    />

    <FileTable
      v-else
      :items="items"
      :loading="loading"
      :selectable="false"
      @open="onOpen"
      @action="onAction"
      @toggle-favorite="actions.toggleFavorite"
    />

    <FileDialogs :actions="actions" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import FileGrid from '@/components/FileGrid.vue'
import FileTable from '@/components/FileTable.vue'
import FileDialogs from '@/components/FileDialogs.vue'
import { useFileStore } from '@/stores/file'
import { useFileActions } from '@/composables/useFileActions'
import * as fileApi from '@/api/file'

const fileStore = useFileStore()

const items = ref([])
const loading = ref(false)

const actions = useFileActions({ onChange: load })

async function load() {
  loading.value = true
  try {
    const data = await fileApi.recent(50)
    items.value = Array.isArray(data) ? data : []
  } finally {
    loading.value = false
  }
}

function onOpen(item) {
  actions.openItem(item, items.value)
}

function onAction(key, item) {
  actions.handleAction(key, item, items.value)
}

onMounted(load)
</script>

<style scoped>
.head-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}
.view-switch {
  display: flex;
  padding: 3px;
  border-radius: 10px;
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
}
.view-switch button {
  width: 30px;
  height: 26px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--nd-text-secondary);
  display: grid;
  place-items: center;
  cursor: pointer;
}
.view-switch button.is-on {
  background: var(--nd-gradient);
  color: #fff;
}
</style>
