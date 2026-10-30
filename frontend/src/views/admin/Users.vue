<template>
  <div class="users">
    <div class="users__toolbar glass-card">
      <el-input
        v-model="keyword"
        class="users__search"
        placeholder="搜索用户名 / 邮箱 / 昵称"
        clearable
        :prefix-icon="Search"
        @keyup.enter="search"
        @clear="search"
      />
      <el-select v-model="size" class="users__size" @change="search">
        <el-option :value="10" label="10 条/页" />
        <el-option :value="20" label="20 条/页" />
        <el-option :value="50" label="50 条/页" />
      </el-select>
      <el-button type="primary" @click="search">
        <el-icon><Search /></el-icon>
        <span>查询</span>
      </el-button>
      <span class="users__total text-secondary num">共 {{ total }} 位用户</span>
    </div>

    <div class="users__table">
      <el-table v-loading="loading" :data="items" style="width: 100%" empty-text="没有匹配的用户">
        <el-table-column label="用户" min-width="210">
          <template #default="{ row }">
            <div class="cell-user">
              <span class="cell-user__avatar">{{ (row.nickname || row.username || 'N').slice(0, 1).toUpperCase() }}</span>
              <div class="cell-user__meta">
                <p class="ellipsis">
                  {{ row.nickname || row.username }}
                  <el-tag v-if="row.role === 1" size="small" type="warning" effect="plain">管理员</el-tag>
                </p>
                <p class="text-muted ellipsis">@{{ row.username }}</p>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="邮箱" min-width="180">
          <template #default="{ row }">
            <span class="text-secondary ellipsis">{{ row.email }}</span>
          </template>
        </el-table-column>

        <el-table-column label="存储使用" width="210">
          <template #default="{ row }">
            <div class="cell-space">
              <div class="cell-space__bar">
                <div class="cell-space__fill" :style="{ width: `${usagePercent(row)}%` }" />
              </div>
              <span class="num text-secondary">
                {{ formatBytes(row.storage_used) }} / {{ formatBytes(row.storage_quota) }}
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="104" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.status === 1"
              :disabled="row.id === user.info?.id"
              @change="(val) => toggleStatus(row, val)"
            />
          </template>
        </el-table-column>

        <el-table-column label="最近登录" width="160">
          <template #default="{ row }">
            <span class="text-secondary">{{ row.last_login_at ? formatTime(row.last_login_at) : '从未登录' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="注册时间" width="160">
          <template #default="{ row }">
            <span class="text-secondary">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="132" align="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openQuota(row)">调整配额</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="users__pager">
      <el-pagination
        v-model:current-page="page"
        :page-size="size"
        :total="total"
        layout="prev, pager, next, jumper"
        background
        @current-change="load"
      />
    </div>

    <el-dialog v-model="quotaOpen" title="调整存储配额" width="420px" append-to-body>
      <p class="quota__user text-secondary">
        用户：{{ quotaRow?.nickname || quotaRow?.username }} · 已使用 {{ formatBytes(quotaRow?.storage_used) }}
      </p>
      <div class="quota__row">
        <el-input-number v-model="quotaGB" :min="0" :max="10240" :step="10" />
        <span class="text-secondary">GB</span>
      </div>
      <div class="quota__presets">
        <el-button v-for="gb in presets" :key="gb" size="small" @click="quotaGB = gb">{{ gb }} G</el-button>
      </div>
      <p class="quota__hint text-muted">填 0 表示不限制。调整后立即生效，不影响已上传文件。</p>
      <template #footer>
        <el-button @click="quotaOpen = false">取消</el-button>
        <el-button type="primary" :loading="savingQuota" @click="saveQuota">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import * as adminApi from '@/api/admin'
import { useUserStore } from '@/stores/user'
import { formatBytes, formatTime, percent } from '@/utils/format'

const GB = 1024 * 1024 * 1024

const user = useUserStore()

const items = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const keyword = ref('')
const loading = ref(false)

const quotaOpen = ref(false)
const quotaRow = ref(null)
const quotaGB = ref(10)
const savingQuota = ref(false)
const presets = [5, 10, 50, 100, 500]

const quotaBytes = computed(() => Math.round((Number(quotaGB.value) || 0) * GB))

function usagePercent(row) {
  const quota = Number(row.storage_quota) || 0
  if (quota <= 0) return 0
  return percent(row.storage_used, quota)
}

async function load() {
  loading.value = true
  try {
    const data = await adminApi.users({
      page: page.value,
      size: size.value,
      keyword: keyword.value.trim() || undefined
    })
    items.value = data?.items || []
    total.value = data?.total || 0
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}

async function toggleStatus(row, enabled) {
  const next = enabled ? 1 : 0
  if (!enabled) {
    try {
      await ElMessageBox.confirm(
        `禁用后「${row.username}」将无法登录，已上传数据不受影响，确定继续？`,
        '禁用账号',
        { type: 'warning', confirmButtonText: '禁用', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
  }
  await adminApi.setStatus(row.id, next)
  row.status = next
  ElMessage.success(enabled ? '账号已启用' : '账号已禁用')
}

function openQuota(row) {
  quotaRow.value = row
  quotaGB.value = Math.round((Number(row.storage_quota) || 0) / GB)
  quotaOpen.value = true
}

async function saveQuota() {
  if (!quotaRow.value) return
  savingQuota.value = true
  try {
    await adminApi.setQuota(quotaRow.value.id, quotaBytes.value)
    quotaRow.value.storage_quota = quotaBytes.value
    ElMessage.success('配额已更新')
    quotaOpen.value = false
  } finally {
    savingQuota.value = false
  }
}

const registerRefresh = inject('admin-refresh', null)
if (registerRefresh) registerRefresh(load)

onMounted(load)
</script>

<style scoped>
.users {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.users__toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  flex-wrap: wrap;
}
.users__search {
  width: 260px;
}
.users__size {
  width: 116px;
}
.users__total {
  margin-left: auto;
  font-size: 12px;
}
.users__table {
  border-radius: var(--nd-radius-card);
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
  backdrop-filter: blur(16px);
  overflow: hidden;
}
.cell-user {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.cell-user__avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  flex: none;
  display: grid;
  place-items: center;
  background: var(--nd-gradient);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
}
.cell-user__meta {
  min-width: 0;
  font-size: 13px;
}
.cell-user__meta p:last-child {
  font-size: 11.5px;
  margin-top: 2px;
}
.cell-space {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 11.5px;
}
.cell-space__bar {
  height: 5px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.22);
  overflow: hidden;
}
.cell-space__fill {
  height: 100%;
  border-radius: 999px;
  background: var(--nd-gradient);
}
.users__pager {
  display: flex;
  justify-content: flex-end;
}
.quota__user {
  font-size: 12.5px;
  margin-bottom: 12px;
}
.quota__row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.quota__presets {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  flex-wrap: wrap;
}
.quota__hint {
  margin-top: 12px;
  font-size: 11.5px;
  line-height: 1.7;
}
</style>
