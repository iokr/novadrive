<template>
  <div class="dashboard">
    <div class="cards">
      <article v-for="card in cards" :key="card.label" class="card glass-card">
        <div class="card__head">
          <span class="card__icon" :style="{ color: card.color }">
            <el-icon><component :is="card.icon" /></el-icon>
          </span>
          <span class="card__label text-secondary">{{ card.label }}</span>
        </div>
        <p class="card__value num">{{ card.value }}</p>
        <p class="card__extra text-muted num">{{ card.extra }}</p>
      </article>
    </div>

    <div class="panels">
      <section class="panel glass-card">
        <header class="panel__head">
          <h3>存储去重概览</h3>
          <span class="text-muted num">{{ savingRate }}% 已节省</span>
        </header>
        <el-progress
          :percentage="savingRate"
          :stroke-width="10"
          :color="progressColors"
          :show-text="false"
          class="panel__progress"
        />
        <div class="panel__rows">
          <div class="row">
            <span class="text-secondary">逻辑占用（所有用户文件体积之和）</span>
            <b class="num">{{ formatBytes(stats.logical_size) }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">物理占用（blob 实际落盘体积）</span>
            <b class="num">{{ formatBytes(stats.physical_size) }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">物理文件块数量</span>
            <b class="num">{{ stats.blob_count }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">分享链接总数</span>
            <b class="num">{{ stats.share_count }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">进行中的上传会话</span>
            <b class="num">{{ stats.active_sessions }}</b>
          </div>
        </div>
      </section>

      <section class="panel glass-card">
        <header class="panel__head">
          <h3>活跃用户</h3>
          <span class="text-muted num">{{ activeRate }}% 启用</span>
        </header>
        <el-progress
          :percentage="activeRate"
          :stroke-width="10"
          :color="progressColors"
          :show-text="false"
          class="panel__progress"
        />
        <div class="panel__rows">
          <div class="row">
            <span class="text-secondary">注册用户总数</span>
            <b class="num">{{ stats.user_count }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">状态正常用户</span>
            <b class="num">{{ stats.active_user_count }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">已禁用用户</span>
            <b class="num">{{ Math.max(0, stats.user_count - stats.active_user_count) }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">文件与目录总量</span>
            <b class="num">{{ stats.file_count }} / {{ stats.dir_count }}</b>
          </div>
          <div class="row">
            <span class="text-secondary">人均逻辑占用</span>
            <b class="num">{{ formatBytes(avgSize) }}</b>
          </div>
        </div>
      </section>
    </div>

    <section class="panel glass-card">
      <header class="panel__head">
        <h3>上传会话监控</h3>
        <div class="panel__tools">
          <el-select v-model="sessionStatus" size="small" class="panel__select" @change="loadSessions">
            <el-option label="全部状态" :value="-1" />
            <el-option label="上传中" :value="0" />
            <el-option label="已合并" :value="1" />
            <el-option label="已取消" :value="2" />
            <el-option label="已过期" :value="3" />
          </el-select>
          <el-button size="small" :loading="sessionLoading" @click="loadSessions">刷新</el-button>
        </div>
      </header>

      <el-table v-loading="sessionLoading" :data="sessions" style="width: 100%" empty-text="暂无上传会话">
        <el-table-column label="文件名" min-width="200">
          <template #default="{ row }">
            <span class="ellipsis">{{ row.file_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="用户" width="130">
          <template #default="{ row }">{{ row.username || `#${row.user_id}` }}</template>
        </el-table-column>
        <el-table-column label="大小" width="110" align="right">
          <template #default="{ row }">
            <span class="num text-secondary">{{ formatBytes(row.file_size) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="分片" width="120" align="center">
          <template #default="{ row }">
            <span class="num text-secondary">{{ formatBytes(row.chunk_size) }} × {{ row.total_chunks }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="104">
          <template #default="{ row }">
            <el-tag :type="sessionType(row.status)" size="small" effect="light">
              {{ sessionText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            <span class="text-secondary">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="160">
          <template #default="{ row }">
            <span class="text-secondary">{{ formatTime(row.expire_at) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import * as adminApi from '@/api/admin'
import { formatBytes, formatTime, percent } from '@/utils/format'

const stats = ref({
  user_count: 0,
  active_user_count: 0,
  file_count: 0,
  dir_count: 0,
  logical_size: 0,
  physical_size: 0,
  blob_count: 0,
  share_count: 0,
  active_sessions: 0
})
const sessions = ref([])
const sessionStatus = ref(-1)
const sessionLoading = ref(false)

const progressColors = [
  { color: '#6366F1', percentage: 60 },
  { color: '#22D3EE', percentage: 100 }
]

const cards = computed(() => [
  {
    label: '注册用户',
    icon: 'User',
    color: '#6366F1',
    value: stats.value.user_count,
    extra: `${stats.value.active_user_count} 个启用中`
  },
  {
    label: '文件总数',
    icon: 'Document',
    color: '#22D3EE',
    value: stats.value.file_count,
    extra: `${stats.value.dir_count} 个目录`
  },
  {
    label: '逻辑占用',
    icon: 'Coin',
    color: '#A78BFA',
    value: formatBytes(stats.value.logical_size),
    extra: '用户视角的存储占用'
  },
  {
    label: '物理占用',
    icon: 'Files',
    color: '#34D399',
    value: formatBytes(stats.value.physical_size),
    extra: `${stats.value.blob_count} 个物理文件块`
  }
])

const savingRate = computed(() => {
  const logical = Number(stats.value.logical_size) || 0
  if (logical <= 0) return 0
  const physical = Number(stats.value.physical_size) || 0
  return Math.max(0, Math.min(100, Math.round((1 - physical / logical) * 1000) / 10))
})

const activeRate = computed(() =>
  percent(stats.value.active_user_count, stats.value.user_count)
)

const avgSize = computed(() => {
  const users = Number(stats.value.user_count) || 0
  if (!users) return 0
  return Math.round((Number(stats.value.logical_size) || 0) / users)
})

function sessionText(status) {
  switch (status) {
    case 0:
      return '上传中'
    case 1:
      return '已合并'
    case 2:
      return '已取消'
    case 3:
      return '已过期'
    default:
      return '未知'
  }
}

function sessionType(status) {
  if (status === 0) return 'primary'
  if (status === 1) return 'success'
  if (status === 2) return 'info'
  return 'warning'
}

async function loadStats() {
  stats.value = await adminApi.stats()
}

async function loadSessions() {
  sessionLoading.value = true
  try {
    const params = {}
    if (sessionStatus.value >= 0) params.status = sessionStatus.value
    const data = await adminApi.uploadSessions(params)
    sessions.value = data?.items || []
  } finally {
    sessionLoading.value = false
  }
}

function reload() {
  loadStats()
  loadSessions()
}

// 顶栏刷新按钮
const registerRefresh = inject('admin-refresh', null)
if (registerRefresh) registerRefresh(reload)

onMounted(reload)
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.card {
  padding: 16px 18px;
}
.card__head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
}
.card__icon {
  font-size: 18px;
}
.card__value {
  margin-top: 10px;
  font-size: 26px;
  font-weight: 650;
  letter-spacing: -0.5px;
}
.card__extra {
  margin-top: 4px;
  font-size: 11.5px;
}
.panels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
}
.panel {
  padding: 18px 20px;
}
.panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}
.panel__head h3 {
  font-size: 14.5px;
  font-weight: 650;
}
.panel__head span {
  font-size: 11.5px;
}
.panel__tools {
  display: flex;
  align-items: center;
  gap: 8px;
}
.panel__select {
  width: 128px;
}
.panel__progress {
  margin-bottom: 14px;
}
.panel__rows {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 12.5px;
}
.row b {
  font-weight: 600;
}
.ellipsis {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
