<template>
  <div class="share-page">
    <header class="share-header">
      <router-link to="/" class="nd-logo share-header__logo">
        <span class="nd-logo__mark">
          <el-icon><Cloudy /></el-icon>
        </span>
        <span>NovaDrive</span>
      </router-link>
      <span class="text-secondary share-header__hint">NovaDrive · 星云盘</span>
    </header>

    <main class="share-main">
      <div v-if="loading" class="share-card glass-card share-card--center">
        <el-icon class="spin"><Loading /></el-icon>
        <p class="text-secondary">正在加载分享内容…</p>
      </div>

      <div v-else-if="errorText" class="share-card glass-card share-card--center">
        <el-icon class="share-card__state-icon"><CircleClose /></el-icon>
        <p class="share-card__error">{{ errorText }}</p>
        <el-button type="primary" @click="router.replace('/')">返回首页</el-button>
      </div>

      <!-- 提取码校验 -->
      <div v-else-if="needPassword" class="share-card glass-card share-card--center share-lock">
        <el-icon class="share-lock__icon"><Lock /></el-icon>
        <p class="share-lock__title">该分享需要提取码</p>
        <p class="share-lock__desc text-secondary">
          {{ share?.file_name }} · 请输入分享者提供的提取码
        </p>
        <div class="share-lock__row">
          <el-input
            v-model="password"
            size="large"
            maxlength="16"
            placeholder="请输入提取码"
            @keyup.enter="verify"
          />
          <el-button type="primary" size="large" :loading="verifying" @click="verify">访问</el-button>
        </div>
        <p v-if="verifyError" class="share-lock__error">{{ verifyError }}</p>
      </div>

      <template v-else-if="share">
        <section class="share-card glass-card">
          <div class="share-summary">
            <div class="share-summary__icon" :style="{ color: share.is_dir ? '#6366F1' : '#22D3EE' }">
              <el-icon><FolderOpened v-if="share.is_dir" /><Document v-else /></el-icon>
            </div>
            <div class="share-summary__main">
              <p class="share-summary__name ellipsis">{{ share.file_name }}</p>
              <p class="share-summary__meta num text-secondary">
                {{ share.is_dir ? '文件夹' : formatBytes(share.file_size) }} ·
                {{ share.expire_at ? remainingTime(share.expire_at) : '永久有效' }} ·
                {{ share.view_count }} 次访问 / {{ share.download_count }} 次下载
              </p>
            </div>
            <el-button
              v-if="!share.is_dir"
              type="primary"
              @click="downloadFile({ id: currentFileId })"
            >
              <el-icon><Download /></el-icon>
              <span>下载文件</span>
            </el-button>
            <el-button v-else type="primary" @click="downloadFile({ id: rootId })">
              <el-icon><Download /></el-icon>
              <span>下载全部</span>
            </el-button>
          </div>

          <template v-if="share.is_dir">
            <nav class="share-crumbs">
              <button
                v-for="(node, idx) in trail"
                :key="`${node.id}-${idx}`"
                type="button"
                class="share-crumbs__item"
                :class="{ 'is-current': idx === trail.length - 1 }"
                @click="goTrail(idx)"
              >
                {{ node.name }}
                <el-icon v-if="idx < trail.length - 1"><ArrowRight /></el-icon>
              </button>
            </nav>

            <div v-if="filesLoading" class="share-list__skeleton">
              <el-skeleton :rows="4" animated />
            </div>

            <ul v-else-if="files.length" class="share-list">
              <li
                v-for="item in files"
                :key="item.id"
                class="share-list__item"
                @click="openItem(item)"
              >
                <el-icon class="share-list__icon" :style="{ color: tintOf(item) }">
                  <component :is="iconOf(item)" />
                </el-icon>
                <div class="share-list__main">
                  <p class="ellipsis">{{ item.name }}</p>
                  <p class="share-list__meta num">
                    {{ isDir(item) ? '文件夹' : formatBytes(item.size) }} ·
                    {{ relativeTime(item.updated_at) }}
                  </p>
                </div>
                <el-button
                  v-if="!isDir(item)"
                  text
                  size="small"
                  class="share-list__download"
                  @click.stop="downloadFile(item)"
                >
                  <el-icon><Download /></el-icon>
                </el-button>
                <el-icon v-else class="share-list__enter"><ArrowRight /></el-icon>
              </li>
            </ul>

            <EmptyState v-else title="这个文件夹是空的" description="分享者还没有往这里放文件" />
          </template>

          <div v-else class="share-preview">
            <img
              v-if="isImageFile"
              :src="shareApi.previewUrl(token, share.file_id)"
              :alt="share.file_name"
              class="share-preview__img"
            />
            <div v-else class="share-preview__placeholder">
              <el-icon><Document /></el-icon>
              <p class="text-secondary">该文件类型暂不支持网页内预览，请下载后查看</p>
              <el-button type="primary" plain @click="openPreview(share.file_id)">在新窗口打开</el-button>
            </div>
          </div>
        </section>
      </template>

      <p class="share-foot text-muted">由 NovaDrive 星云盘提供分享服务</p>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import EmptyState from '@/components/EmptyState.vue'
import * as shareApi from '@/api/share'
import { canPreview, iconOf, isDir, isImage, tintOf } from '@/utils/fileType'
import { formatBytes, relativeTime, remainingTime } from '@/utils/format'
import { downloadUrl as triggerUrl } from '@/utils/download'

const route = useRoute()
const router = useRouter()

const token = String(route.params.token || '')
const share = ref(null)
const files = ref([])
const trail = ref([])
const rootId = ref(0)
const currentFileId = ref(0)
const loading = ref(true)
const filesLoading = ref(false)
const verifying = ref(false)
const password = ref('')
const verifyError = ref('')
const errorText = ref('')
const unlocked = ref(false)

const needPassword = computed(() => !!share.value?.has_password && !unlocked.value)
const isImageFile = computed(() => (share.value ? isImage({ name: share.value.file_name }) : false))

async function load() {
  loading.value = true
  errorText.value = ''
  try {
    const info = await shareApi.info(token)
    share.value = info
    if (info.has_password && !shareApi.getShareKey(token)) {
      unlocked.value = false
      return
    }
    unlocked.value = true
    await afterUnlock()
  } catch (err) {
    errorText.value = err?.message || '分享不存在或已失效'
  } finally {
    loading.value = false
  }
}

async function afterUnlock() {
  if (!share.value) return
  if (share.value.is_dir) {
    await loadRoot()
  } else {
    currentFileId.value = share.value.file_id
  }
}

async function loadRoot() {
  const data = await shareApi.files(token, 0)
  rootId.value = data.root_id
  trail.value = [{ id: data.root_id, name: data.share?.file_name || share.value.file_name }]
  currentFileId.value = data.current_id || data.root_id
  files.value = data.items || []
}

async function loadDir(id) {
  filesLoading.value = true
  try {
    const data = await shareApi.files(token, id)
    currentFileId.value = data.current_id || id
    files.value = data.items || []
  } finally {
    filesLoading.value = false
  }
}

async function verify() {
  const code = password.value.trim()
  if (!code) {
    verifyError.value = '请输入提取码'
    return
  }
  verifying.value = true
  verifyError.value = ''
  try {
    const data = await shareApi.verify(token, code)
    shareApi.setShareKey(token, data.access_key)
    unlocked.value = true
    await afterUnlock()
  } catch (err) {
    verifyError.value = err?.message || '提取码错误'
  } finally {
    verifying.value = false
  }
}

function openItem(item) {
  if (isDir(item)) {
    trail.value.push({ id: item.id, name: item.name })
    loadDir(item.id)
    return
  }
  if (canPreview(item)) openPreview(item.id)
  else confirmDownload(item)
}

// 不支持的格式先提示，用户确认后再下载
async function confirmDownload(item) {
  try {
    await ElMessageBox.confirm(
      `「${item.name}」暂不支持在线预览，是否下载到本地查看？`,
      '该文件无法预览',
      {
        type: 'info',
        confirmButtonText: '下载',
        cancelButtonText: '取消',
        // 不锁页面滚动：锁滚动会隐藏滚动条，关闭时再恢复，页面右侧会“抖”一下
        lockScroll: false
      }
    )
  } catch {
    return
  }
  downloadFile(item)
}

function goTrail(index) {
  const node = trail.value[index]
  if (!node) return
  trail.value = trail.value.slice(0, index + 1)
  loadDir(node.id)
}

function openPreview(fileId) {
  window.open(shareApi.previewUrl(token, fileId), '_blank', 'noopener')
}

function downloadFile(item) {
  if (!item?.id) return
  triggerUrl(shareApi.downloadUrl(token, item.id))
  ElMessage.success('开始下载')
}

onMounted(load)
</script>

<style scoped>
.share-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.share-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px clamp(16px, 5vw, 48px);
}
.share-header__hint {
  font-size: 12px;
}
.share-main {
  flex: 1;
  width: min(880px, 92vw);
  margin: 0 auto;
  padding: 12px 0 48px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.share-card {
  padding: 22px;
}
.share-card--center {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 280px;
  text-align: center;
}
.share-card__state-icon {
  font-size: 46px;
  color: #f87171;
}
.share-card__error {
  font-size: 14px;
  color: var(--nd-text-secondary);
  max-width: 420px;
  line-height: 1.7;
}
.share-lock__icon {
  font-size: 38px;
  color: var(--nd-cyan);
}
.share-lock__title {
  font-size: 16px;
  font-weight: 650;
}
.share-lock__desc {
  font-size: 13px;
}
.share-lock__row {
  display: flex;
  gap: 10px;
  width: min(380px, 100%);
  margin-top: 6px;
}
.share-lock__error {
  font-size: 12.5px;
  color: #f87171;
}
.share-summary {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--nd-border);
  flex-wrap: wrap;
}
.share-summary__icon {
  width: 50px;
  height: 50px;
  border-radius: 14px;
  flex: none;
  display: grid;
  place-items: center;
  font-size: 24px;
  background: var(--nd-surface-hover);
}
.share-summary__main {
  flex: 1;
  min-width: 0;
}
.share-summary__name {
  font-size: 16px;
  font-weight: 650;
}
.share-summary__meta {
  font-size: 12px;
  margin-top: 4px;
}
.share-crumbs {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
  padding: 12px 0 4px;
}
.share-crumbs__item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  border: none;
  background: transparent;
  color: var(--nd-text-secondary);
  font-size: 12.5px;
  padding: 4px 6px;
  border-radius: 7px;
  cursor: pointer;
}
.share-crumbs__item:hover {
  background: var(--nd-surface-hover);
  color: var(--nd-text);
}
.share-crumbs__item.is-current {
  color: var(--nd-text);
  font-weight: 600;
}
.share-list {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
}
.share-list__item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 10px;
  border-radius: 12px;
  cursor: pointer;
}
.share-list__item:hover {
  background: var(--nd-surface-hover);
}
.share-list__icon {
  font-size: 21px;
  flex: none;
}
.share-list__main {
  flex: 1;
  min-width: 0;
  font-size: 13.5px;
}
.share-list__meta {
  font-size: 11.5px;
  color: var(--nd-text-muted);
  margin-top: 2px;
}
.share-list__enter {
  color: var(--nd-text-muted);
}
.share-list__skeleton {
  padding: 16px 4px;
}
.share-preview {
  display: flex;
  justify-content: center;
  padding-top: 18px;
}
.share-preview__img {
  max-width: 100%;
  max-height: 62vh;
  border-radius: 12px;
  box-shadow: var(--nd-shadow);
}
.share-preview__placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px 16px;
  color: var(--nd-primary);
  font-size: 46px;
  text-align: center;
}
.share-preview__placeholder p {
  font-size: 13px;
}
.share-foot {
  text-align: center;
  font-size: 11.5px;
}
.spin {
  font-size: 26px;
  color: var(--nd-cyan);
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
