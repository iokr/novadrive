<template>
  <transition name="panel">
    <section v-if="upload.hasTasks && upload.panelOpen" class="panel glass-card" :class="{ 'is-collapsed': collapsed }">
      <header class="panel__head">
        <div class="panel__title">
          <el-icon class="panel__icon"><UploadFilled /></el-icon>
          <span>上传列表</span>
          <span class="panel__count num">
            {{ upload.activeCount + upload.pendingCount }} 进行中 · {{ upload.tasks.length }} 个任务
          </span>
        </div>
        <div class="panel__head-actions">
          <el-button text size="small" @click="upload.clearFinished()">清除已完成</el-button>
          <el-button text size="small" @click="collapsed = !collapsed">
            <el-icon><ArrowDown v-if="collapsed" /><ArrowUp v-else /></el-icon>
          </el-button>
          <el-button text size="small" @click="upload.panelOpen = false">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
      </header>

      <div v-show="!collapsed" class="panel__body">
        <div class="panel__overall">
          <el-progress
            :percentage="upload.overallProgress"
            :stroke-width="6"
            :show-text="false"
            :color="progressColors"
          />
          <span class="num">{{ upload.overallProgress }}%</span>
        </div>

        <ul class="task-list">
          <li v-for="task in upload.tasks" :key="task.id" class="task">
            <div class="task__row">
              <el-icon class="task__icon" :class="`is-${task.status}`">
                <Loading v-if="task.status === 'hashing' || task.status === 'merging'" class="spin" />
                <UploadFilled v-else-if="task.status === 'uploading'" />
                <VideoPause v-else-if="task.status === 'paused'" />
                <CircleCheck v-else-if="task.status === 'done'" />
                <CircleClose v-else-if="task.status === 'error'" />
                <Clock v-else />
              </el-icon>
              <div class="task__main">
                <div class="task__title">
                  <span class="ellipsis" :title="task.name">{{ task.name }}</span>
                  <span class="task__status" :class="`is-${task.status}`">{{ statusText(task) }}</span>
                </div>
                <div class="task__bar">
                  <div class="task__bar-fill" :class="`is-${task.status}`" :style="{ width: `${Math.round((task.progress || 0) * 100)}%` }" />
                </div>
                <div class="task__meta num">
                  <span>{{ formatBytes(task.uploadedBytes || 0) }} / {{ formatBytes(task.size) }}</span>
                  <span v-if="task.kind === 'file' && task.status === 'uploading' && task.totalChunks">
                    分片 {{ task.doneChunks }}/{{ task.totalChunks }}
                  </span>
                  <span v-if="task.kind === 'batch'">
                    文件 {{ task.doneFiles }}/{{ task.totalFiles }}
                  </span>
                  <span v-if="task.speed">{{ formatSpeed(task.speed) }}</span>
                  <span v-if="task.error" class="task__error">{{ task.error }}</span>
                </div>
              </div>
              <div class="task__actions">
                <el-button
                  v-if="['uploading', 'hashing', 'waiting'].includes(task.status)"
                  text
                  size="small"
                  title="暂停"
                  @click="upload.pause(task.id)"
                >
                  <el-icon><VideoPause /></el-icon>
                </el-button>
                <el-button
                  v-if="task.status === 'paused'"
                  text
                  size="small"
                  title="继续"
                  @click="upload.resume(task.id)"
                >
                  <el-icon><VideoPlay /></el-icon>
                </el-button>
                <el-button v-if="task.status === 'error'" text size="small" title="重试" @click="upload.retry(task.id)">
                  <el-icon><RefreshRight /></el-icon>
                </el-button>
                <el-button text size="small" title="移除" @click="upload.remove(task.id)">
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </section>
  </transition>
</template>

<script setup>
import { ref } from 'vue'
import { useUploadStore } from '@/stores/upload'
import { formatBytes, formatSpeed } from '@/utils/format'

const upload = useUploadStore()
const collapsed = ref(false)
const progressColors = [
  { color: '#2B7FFF', percentage: 60 },
  { color: '#22D3EE', percentage: 100 }
]

function statusText(task) {
  if (task.instant) return '秒传完成'
  switch (task.status) {
    case 'waiting':
      return '排队中'
    case 'hashing':
      return '计算 MD5'
    case 'uploading':
      return '上传中'
    case 'merging':
      return '合并中'
    case 'paused':
      return '已暂停'
    case 'done':
      return '已完成'
    case 'error':
      return '失败'
    case 'canceled':
      return '已取消'
    default:
      return ''
  }
}
</script>

<style scoped>
.panel {
  position: fixed;
  right: 22px;
  bottom: 22px;
  width: 396px;
  z-index: 1500;
  background: var(--el-bg-color-overlay);
  border-radius: var(--nd-radius-card);
  border: 1px solid var(--nd-border);
  box-shadow: var(--nd-shadow);
  overflow: hidden;
}
.panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--nd-border);
}
.panel__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
}
.panel__icon {
  color: var(--nd-primary);
}
.panel__count {
  font-weight: 400;
  font-size: 12px;
  color: var(--nd-text-secondary);
}
.panel__head-actions :deep(.el-button) {
  padding: 4px 6px;
}
.panel__body {
  max-height: 46vh;
  overflow: auto;
}
.panel__overall {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px 4px;
  font-size: 12px;
  color: var(--nd-text-secondary);
}
.panel__overall :deep(.el-progress) {
  flex: 1;
}
.task-list {
  list-style: none;
  margin: 0;
  padding: 4px 8px 10px;
}
.task {
  padding: 6px 6px;
  border-radius: 10px;
}
.task:hover {
  background: var(--nd-surface-hover);
}
.task__row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.task__icon {
  margin-top: 2px;
  font-size: 16px;
  color: var(--nd-text-secondary);
}
.task__icon.is-done {
  color: #34d399;
}
.task__icon.is-error {
  color: #f87171;
}
.task__main {
  flex: 1;
  min-width: 0;
}
.task__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.task__title .ellipsis {
  flex: 1;
  min-width: 0;
}
.task__status {
  font-size: 11.5px;
  color: var(--nd-text-secondary);
  flex: none;
}
.task__status.is-done {
  color: #34d399;
}
.task__status.is-error {
  color: #f87171;
}
.task__bar {
  height: 4px;
  margin: 6px 0 4px;
  border-radius: 4px;
  background: var(--nd-border);
  overflow: hidden;
}
.task__bar-fill {
  height: 100%;
  border-radius: 4px;
  background: var(--nd-gradient);
  transition: width 200ms ease;
}
.task__bar-fill.is-done {
  background: linear-gradient(135deg, #34d399, #22d3ee);
}
.task__bar-fill.is-error {
  background: #f87171;
}
.task__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 11.5px;
  color: var(--nd-text-muted);
}
.task__error {
  color: #f87171;
}
.task__actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
.spin {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.panel-enter-active,
.panel-leave-active {
  transition: opacity 200ms ease, transform 200ms ease;
}
.panel-enter-from,
.panel-leave-to {
  opacity: 0;
  transform: translateY(16px);
}
@media (max-width: 767px) {
  .panel {
    right: 8px;
    left: 8px;
    width: auto;
    bottom: 8px;
  }
}
</style>
