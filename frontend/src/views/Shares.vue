<template>
  <div class="nd-page">
    <div class="nd-page__header">
      <div>
        <p class="nd-page__title">我的分享</p>
        <p class="nd-page__subtitle num">共 {{ items.length }} 条分享链接</p>
      </div>
      <div class="head-tools">
        <el-radio-group v-model="filter" size="small">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="active">有效</el-radio-button>
          <el-radio-button value="invalid">已失效</el-radio-button>
        </el-radio-group>
        <el-button :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          <span>刷新</span>
        </el-button>
      </div>
    </div>

    <div class="stats">
      <div class="stat-card glass-card">
        <span class="stat-card__icon"><el-icon><Share /></el-icon></span>
        <div>
          <p class="stat-card__value num">{{ items.length }}</p>
          <p class="stat-card__label text-secondary">分享总数</p>
        </div>
      </div>
      <div class="stat-card glass-card">
        <span class="stat-card__icon"><el-icon><CircleCheck /></el-icon></span>
        <div>
          <p class="stat-card__value num">{{ activeCount }}</p>
          <p class="stat-card__label text-secondary">当前有效</p>
        </div>
      </div>
      <div class="stat-card glass-card">
        <span class="stat-card__icon"><el-icon><View /></el-icon></span>
        <div>
          <p class="stat-card__value num">{{ totalViews }}</p>
          <p class="stat-card__label text-secondary">累计访问</p>
        </div>
      </div>
      <div class="stat-card glass-card">
        <span class="stat-card__icon"><el-icon><Download /></el-icon></span>
        <div>
          <p class="stat-card__value num">{{ totalDownloads }}</p>
          <p class="stat-card__label text-secondary">累计下载</p>
        </div>
      </div>
    </div>

    <div class="table-card">
      <el-table v-loading="loading" :data="filtered" style="width: 100%" empty-text="还没有创建过分享">
        <el-table-column label="文件" min-width="240">
          <template #default="{ row }">
            <div class="cell-file">
              <el-icon :style="{ color: row.is_dir ? '#6366F1' : '#22D3EE' }">
                <Folder v-if="row.is_dir" />
                <Document v-else />
              </el-icon>
              <span class="ellipsis" :title="row.file_name">{{ row.file_name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="108">
          <template #default="{ row }">
            <el-tag :type="statusType(row)" size="small" effect="light">{{ statusText(row) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="提取码" width="92">
          <template #default="{ row }">
            <span class="text-secondary">{{ row.has_password ? '需要' : '无' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="有效期" width="150">
          <template #default="{ row }">
            <span class="text-secondary">{{ row.expire_at ? remainingTime(row.expire_at) : '永久' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="访问 / 下载" width="126" align="center">
          <template #default="{ row }">
            <span class="num text-secondary">{{ row.view_count }} / {{ row.download_count }}</span>
            <span v-if="row.max_downloads > 0" class="text-muted num"> / 限 {{ row.max_downloads }}</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="150">
          <template #default="{ row }">
            <span class="text-secondary">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="230" align="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="copyLink(row)">复制链接</el-button>
            <el-button link type="primary" size="small" @click="showQr(row)">二维码</el-button>
            <el-button link type="primary" size="small" :disabled="row.status !== 1" @click="openShare(row)">打开</el-button>
            <el-button link type="danger" size="small" @click="cancelShare(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="qrOpen" title="分享二维码" width="360px" append-to-body>
      <div class="qr-box">
        <QrcodeVue v-if="qrRow" :value="absoluteUrl(qrRow.url)" :size="180" level="M" />
        <p class="qr-box__name ellipsis">{{ qrRow?.file_name }}</p>
        <el-input :model-value="absoluteUrl(qrRow?.url)" readonly>
          <template #append>
            <el-button @click="copyLink(qrRow)">复制</el-button>
          </template>
        </el-input>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import QrcodeVue from 'qrcode.vue'
import * as shareApi from '@/api/share'
import { formatTime, remainingTime } from '@/utils/format'

const router = useRouter()

const items = ref([])
const loading = ref(false)
const filter = ref('all')
const qrOpen = ref(false)
const qrRow = ref(null)

const activeCount = computed(() => items.value.filter((it) => it.status === 1).length)
const totalViews = computed(() => items.value.reduce((sum, it) => sum + (it.view_count || 0), 0))
const totalDownloads = computed(() => items.value.reduce((sum, it) => sum + (it.download_count || 0), 0))

const filtered = computed(() => {
  if (filter.value === 'active') return items.value.filter((it) => it.status === 1)
  if (filter.value === 'invalid') return items.value.filter((it) => it.status !== 1)
  return items.value
})

function statusText(row) {
  if (row.status === 1) return '有效'
  if (row.status === 2) return '已过期'
  return '已取消'
}

function statusType(row) {
  if (row.status === 1) return 'success'
  if (row.status === 2) return 'warning'
  return 'info'
}

function absoluteUrl(url) {
  if (!url) return ''
  if (/^https?:\/\//i.test(url)) return url
  return `${window.location.origin}${url.startsWith('/') ? '' : '/'}${url}`
}

async function load() {
  loading.value = true
  try {
    const data = await shareApi.mine()
    items.value = Array.isArray(data) ? data : []
  } finally {
    loading.value = false
  }
}

async function copyLink(row) {
  if (!row) return
  try {
    await navigator.clipboard.writeText(absoluteUrl(row.url))
    ElMessage.success('链接已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

function showQr(row) {
  qrRow.value = row
  qrOpen.value = true
}

function openShare(row) {
  router.push(`/s/${row.token}`)
}

async function cancelShare(row) {
  try {
    await ElMessageBox.confirm(`取消后链接「${row.file_name}」将立即失效，确定继续？`, '取消分享', {
      type: 'warning',
      confirmButtonText: '取消分享',
      cancelButtonText: '返回'
    })
  } catch {
    return
  }
  await shareApi.cancel(row.id)
  ElMessage.success('分享已取消')
  load()
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
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(178px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
}
.stat-card__icon {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: rgba(99, 102, 241, 0.14);
  color: var(--nd-cyan);
  font-size: 18px;
  flex: none;
}
.stat-card__value {
  font-size: 19px;
  font-weight: 650;
}
.stat-card__label {
  font-size: 11.5px;
  margin-top: 2px;
}
.table-card {
  border-radius: var(--nd-radius-card);
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
  backdrop-filter: blur(16px);
  overflow: hidden;
}
.cell-file {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.qr-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.qr-box :deep(canvas) {
  border-radius: 10px;
  background: #fff;
  padding: 8px;
}
.qr-box__name {
  font-size: 13px;
  max-width: 100%;
}
</style>
