<template>
  <div class="nd-page">
    <div class="nd-page__header">
      <div>
        <p class="nd-page__title">收藏</p>
        <p class="nd-page__subtitle num">共 {{ items.length }} 个收藏项目，随时快速访问</p>
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

    <div v-if="selected.length" class="bulk">
      <span>已选择 <b class="num">{{ selected.length }}</b> 项</span>
      <el-button size="small" @click="downloadSelected">下载选中</el-button>
      <el-button size="small" @click="unfavoriteSelected">取消收藏</el-button>
      <el-button size="small" text @click="selected = []">取消选择</el-button>
    </div>

    <FileGrid
      v-if="fileStore.viewMode === 'grid'"
      :items="items"
      :selected="selected"
      :loading="loading"
      mode="favorites"
      empty-title="还没有收藏任何文件"
      empty-desc="在文件卡片上点击 ☆ 即可加入收藏"
      @open="onOpen"
      @action="onAction"
      @toggle-select="toggleSelect"
      @toggle-favorite="actions.toggleFavorite"
    />

    <FileTable
      v-else
      :items="items"
      :selected="selected"
      :loading="loading"
      mode="favorites"
      @open="onOpen"
      @action="onAction"
      @toggle-select="toggleSelect"
      @toggle-select-all="toggleSelectAll"
      @toggle-favorite="actions.toggleFavorite"
    />

    <FileDialogs :actions="actions" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import FileGrid from '@/components/FileGrid.vue'
import FileTable from '@/components/FileTable.vue'
import FileDialogs from '@/components/FileDialogs.vue'
import { useFileStore } from '@/stores/file'
import { useFileActions } from '@/composables/useFileActions'
import * as fileApi from '@/api/file'

const fileStore = useFileStore()

const items = ref([])
const loading = ref(false)
const selected = ref([])

const actions = useFileActions({ onChange: load })

async function load() {
  loading.value = true
  try {
    const data = await fileApi.favorites()
    items.value = Array.isArray(data) ? data : []
    // 收藏被取消后同步移除选中态
    const ids = new Set(items.value.map((it) => it.id))
    selected.value = selected.value.filter((id) => ids.has(id))
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

function toggleSelect(id) {
  const idx = selected.value.indexOf(id)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(id)
}

function toggleSelectAll() {
  selected.value =
    selected.value.length === items.value.length ? [] : items.value.map((it) => it.id)
}

async function downloadSelected() {
  const ids = [...selected.value]
  if (ids.length === 1) {
    const item = items.value.find((it) => it.id === ids[0])
    if (item) actions.download(item)
    return
  }
  await actions.downloadBatch(ids, 'novadrive-favorites.zip')
}

async function unfavoriteSelected() {
  const targets = items.value.filter((it) => selected.value.includes(it.id))
  await Promise.allSettled(targets.map((it) => fileApi.setFavorite(it.id, false)))
  ElMessage.success('已取消收藏')
  selected.value = []
  load()
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
.bulk {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin-bottom: 14px;
  border-radius: 12px;
  border: 1px solid var(--nd-primary);
  background: rgba(99, 102, 241, 0.12);
  font-size: 13px;
}
</style>
