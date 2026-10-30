<template>
  <div class="nd-page">
    <div class="nd-page__header">
      <div>
        <p class="nd-page__title">回收站</p>
        <p class="nd-page__subtitle num">共 {{ items.length }} 项待清理，回收站内容不计入已用空间</p>
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
        <el-button type="danger" plain :disabled="!items.length" @click="onEmpty">
          <el-icon><Delete /></el-icon>
          <span>清空回收站</span>
        </el-button>
      </div>
    </div>

    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="notice"
      title="删除到回收站的文件仍占用物理存储，彻底删除或清空回收站后才会释放"
    />

    <FileGrid
      v-if="fileStore.viewMode === 'grid'"
      :items="items"
      :loading="loading"
      mode="recycle"
      :selectable="false"
      empty-title="回收站是空的"
      empty-desc="删除的文件会先进入这里，可随时还原"
      @open="onOpen"
      @action="onAction"
    />

    <FileTable
      v-else
      :items="items"
      :loading="loading"
      mode="recycle"
      :selectable="false"
      @open="onOpen"
      @action="onAction"
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
import { useUserStore } from '@/stores/user'
import { useFileActions } from '@/composables/useFileActions'
import * as fileApi from '@/api/file'

const fileStore = useFileStore()
const user = useUserStore()

const items = ref([])
const loading = ref(false)

const actions = useFileActions({ onChange: load })

async function load() {
  loading.value = true
  try {
    const data = await fileApi.recycleList()
    items.value = Array.isArray(data) ? data : []
  } finally {
    loading.value = false
  }
}

// 回收站里的条目不允许打开或预览，只保留还原 / 彻底删除
function onOpen() {}

function onAction(key, item) {
  actions.handleAction(key, item)
}

async function onEmpty() {
  await actions.emptyRecycle()
  user.fetchSpace()
}

onMounted(load)
</script>

<style scoped>
.head-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
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
.notice {
  margin-bottom: 14px;
  border-radius: 12px;
}
</style>
