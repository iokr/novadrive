<template>
  <div class="layout" :class="{ 'is-selecting': fileStore.hasSelection }">
    <aside class="sidebar" :class="{ 'is-open': drawerOpen }">
      <div class="sidebar__brand nd-logo">
        <span class="nd-logo__mark">
          <el-icon><Cloudy /></el-icon>
        </span>
        <span class="sidebar__brand-text">NovaDrive</span>
      </div>

      <nav class="nav">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav__item"
          :class="{ 'is-active': isActive(item) }"
          @click="drawerOpen = false"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span class="nav__label">{{ item.label }}</span>
          <span v-if="item.badge" class="nav__badge">{{ item.badge }}</span>
        </router-link>
      </nav>

      <div class="sidebar__foot">
        <router-link v-if="user.isAdmin" to="/admin" class="nav__item nav__item--admin" @click="drawerOpen = false">
          <el-icon><Setting /></el-icon>
          <span class="nav__label">管理后台</span>
        </router-link>

        <router-link to="/profile" class="space" @click="drawerOpen = false">
          <div class="space__head">
            <span class="nav__label">存储空间</span>
            <span class="space__percent num">{{ user.usedPercent }}%</span>
          </div>
          <div class="space__bar">
            <div class="space__bar-fill" :style="{ width: `${user.usedPercent}%` }" />
          </div>
          <p class="space__text num">
            {{ formatBytes(user.space.used) }} / {{ quotaText }}
          </p>
        </router-link>
      </div>
    </aside>

    <div v-if="drawerOpen" class="layout__mask" @click="drawerOpen = false" />

    <div class="layout__main">
      <header class="topbar">
        <button type="button" class="topbar__menu" @click="drawerOpen = true">
          <el-icon><Expand /></el-icon>
        </button>

        <button type="button" class="search-trigger" @click="searchOpen = true">
          <el-icon><Search /></el-icon>
          <span class="search-trigger__text">搜索我的文件…</span>
          <kbd class="search-trigger__kbd">Ctrl K</kbd>
        </button>

        <div class="topbar__actions">
          <el-dropdown trigger="click" @command="onUploadCommand">
            <el-button type="primary" class="upload-btn">
              <el-icon><UploadFilled /></el-icon>
              <span class="upload-btn__text">上传</span>
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="file">
                  <el-icon><Document /></el-icon>上传文件
                </el-dropdown-item>
                <el-dropdown-item command="folder">
                  <el-icon><FolderOpened /></el-icon>上传文件夹
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <el-tooltip :content="theme.isDark ? '切换浅色' : '切换深色'" placement="bottom">
            <button type="button" class="topbar__icon" @click="theme.toggle()">
              <el-icon><Sunny v-if="theme.isDark" /><Moon v-else /></el-icon>
            </button>
          </el-tooltip>

          <el-tooltip content="收藏" placement="bottom">
            <button type="button" class="topbar__icon topbar__star" @click="router.push('/favorites')">
              <el-icon><Star /></el-icon>
            </button>
          </el-tooltip>

          <el-dropdown trigger="click" @command="onUserCommand">
            <button type="button" class="user-chip">
              <span class="user-chip__avatar">{{ user.avatarText }}</span>
              <span class="user-chip__name">{{ user.displayName }}</span>
              <el-icon class="user-chip__caret"><ArrowDown /></el-icon>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>个人中心
                </el-dropdown-item>
                <el-dropdown-item v-if="user.isAdmin" command="admin">
                  <el-icon><Setting /></el-icon>管理后台
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="content">
        <router-view v-slot="{ Component }">
          <transition name="view" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>

    <input
      ref="fileInputRef"
      type="file"
      multiple
      class="hidden-input"
      @change="onPickFiles"
    />
    <input
      ref="folderInputRef"
      type="file"
      webkitdirectory
      multiple
      class="hidden-input"
      @change="onPickFolder"
    />

    <UploadPanel />
    <GlobalSearch v-model="searchOpen" @open="onSearchOpen" />
    <FileDialogs :actions="layoutActions" />

    <transition name="drop">
      <div v-if="dragging" class="drop-mask">
        <div class="drop-mask__box glass-card">
          <el-icon><UploadFilled /></el-icon>
          <p>释放鼠标，上传到当前目录</p>
          <p class="text-secondary">支持多文件与整个文件夹</p>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import UploadPanel from '@/components/UploadPanel.vue'
import GlobalSearch from '@/components/GlobalSearch.vue'
import FileDialogs from '@/components/FileDialogs.vue'
import { useUserStore } from '@/stores/user'
import { useFileStore } from '@/stores/file'
import { useUploadStore } from '@/stores/upload'
import { useThemeStore } from '@/stores/theme'
import { useFileActions } from '@/composables/useFileActions'
import { filesFromInput, filesFromDataTransfer } from '@/utils/folderReader'
import { formatBytes } from '@/utils/format'
import { canPreview, isDir } from '@/utils/fileType'

const route = useRoute()
const router = useRouter()
const user = useUserStore()
const fileStore = useFileStore()
const upload = useUploadStore()
const theme = useThemeStore()

const drawerOpen = ref(false)
const searchOpen = ref(false)
const dragging = ref(false)
const fileInputRef = ref(null)
const folderInputRef = ref(null)

// 布局层只处理搜索结果的预览，不参与列表刷新
const layoutActions = useFileActions({})

const navItems = [
  { to: '/', label: '我的文件', icon: 'FolderOpened', match: 'home' },
  { to: '/favorites', label: '收藏', icon: 'Star', match: 'favorites' },
  { to: '/recent', label: '最近访问', icon: 'Clock', match: 'recent' },
  { to: '/shares', label: '我的分享', icon: 'Share', match: 'shares' },
  { to: '/recycle', label: '回收站', icon: 'Delete', match: 'recycle' }
]

const quotaText = computed(() =>
  Number(user.space.quota) > 0 ? formatBytes(user.space.quota) : '不限'
)

function isActive(item) {
  return route.name === item.match
}

// 上传目标：只有文件首页跟随当前目录，其它页面落到根目录
function uploadTarget() {
  return route.name === 'home' ? fileStore.parentId : 0
}

function onUploadCommand(command) {
  if (command === 'folder') folderInputRef.value?.click()
  else fileInputRef.value?.click()
}

function onPickFiles(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  if (!files.length) return
  upload.addFiles(
    files.map((file) => ({ file, relativePath: file.name })),
    uploadTarget()
  )
}

function onPickFolder(event) {
  const items = filesFromInput(event.target.files)
  event.target.value = ''
  if (!items.length) return
  upload.addFiles(items, uploadTarget())
}

function onSearchOpen(item) {
  if (!item) return
  if (isDir(item)) {
    fileStore.parentId = item.id
    router.push({ path: '/', query: { folder: item.id } })
    return
  }
  if (canPreview(item)) layoutActions.openPreview(item, [item])
  else layoutActions.confirmDownload(item)
}

async function onUserCommand(command) {
  if (command === 'profile') {
    router.push('/profile')
  } else if (command === 'admin') {
    router.push('/admin')
  } else if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定退出当前账号？', '退出登录', {
        type: 'warning',
        confirmButtonText: '退出',
        cancelButtonText: '取消'
      })
    } catch {
      return
    }
    upload.cancelAll()
    user.logout()
    ElMessage.success('已退出登录')
    router.replace('/login')
  }
}

// ---------- 全局拖拽上传 ----------
let dragDepth = 0

function hasFiles(event) {
  const types = event.dataTransfer?.types
  return !!types && Array.from(types).includes('Files')
}

function onDragEnter(event) {
  if (!hasFiles(event)) return
  dragDepth += 1
  dragging.value = true
}

function onDragOver(event) {
  if (!hasFiles(event)) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
}

function onDragLeave(event) {
  if (!hasFiles(event)) return
  dragDepth -= 1
  if (dragDepth <= 0) {
    dragDepth = 0
    dragging.value = false
  }
}

async function onDrop(event) {
  if (!hasFiles(event)) return
  event.preventDefault()
  dragDepth = 0
  dragging.value = false
  const items = await filesFromDataTransfer(event.dataTransfer)
  if (!items.length) {
    ElMessage.warning('没有可上传的文件')
    return
  }
  upload.addFiles(items, uploadTarget())
}

// ---------- 快捷键 ----------
function onKeydown(event) {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    searchOpen.value = true
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('dragenter', onDragEnter)
  window.addEventListener('dragover', onDragOver)
  window.addEventListener('dragleave', onDragLeave)
  window.addEventListener('drop', onDrop)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('dragenter', onDragEnter)
  window.removeEventListener('dragover', onDragOver)
  window.removeEventListener('dragleave', onDragLeave)
  window.removeEventListener('drop', onDrop)
})

watch(
  () => route.fullPath,
  () => {
    drawerOpen.value = false
  }
)
</script>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
}
.hidden-input {
  display: none;
}

/* ---------- 侧边栏 ---------- */
.sidebar {
  width: var(--nd-sidebar-width);
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 12px 14px;
  border-right: 1px solid var(--nd-border);
  background: var(--nd-surface);
  position: sticky;
  top: 0;
  height: 100vh;
  z-index: 30;
  transition: transform 220ms ease;
}
.sidebar__brand {
  padding: 4px 6px 10px;
}
.sidebar__brand-text {
  font-size: 16px;
}
.nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  overflow-y: auto;
}
.nav__item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 8px;
  color: var(--nd-text-secondary);
  font-size: 13.5px;
  transition: background 160ms ease, color 160ms ease;
}
.nav__item:hover {
  background: var(--nd-surface-hover);
  color: var(--nd-text);
}
.nav__item.is-active {
  color: var(--nd-primary);
  background: var(--nd-primary-soft);
  font-weight: 600;
}
.nav__badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: #f56c6c;
  color: #fff;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
  flex: none;
}
.nav__item--admin {
  margin-bottom: 8px;
}
.nav__label {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
}
.sidebar__foot {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid var(--nd-border);
  padding-top: 12px;
}
.space {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--nd-surface-sunken);
  color: var(--nd-text-secondary);
  font-size: 12px;
}
.space__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.space__percent {
  color: var(--nd-primary);
  font-size: 11.5px;
}
.space__bar {
  height: 6px;
  border-radius: 999px;
  background: #e8edf5;
  overflow: hidden;
}
.space__bar-fill {
  height: 100%;
  border-radius: 999px;
  background: var(--nd-primary);
  transition: width 240ms ease;
}
.space__text {
  font-size: 11px;
  color: var(--nd-text-muted);
}

/* ---------- 主区域 ---------- */
.layout__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  height: var(--nd-topbar-height);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 24px;
  border-bottom: 1px solid var(--nd-border);
  background: var(--nd-surface);
}
.topbar__menu {
  display: none;
  width: 34px;
  height: 34px;
  border-radius: 10px;
  border: 1px solid var(--nd-border);
  background: transparent;
  color: var(--nd-text);
  place-items: center;
  cursor: pointer;
}
.search-trigger {
  flex: 0 1 420px;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid transparent;
  background: var(--nd-surface-sunken);
  color: var(--nd-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background 160ms ease;
}
.search-trigger:hover {
  border-color: var(--nd-border-strong);
  background: var(--nd-surface);
  color: var(--nd-text);
}
.search-trigger__text {
  flex: 1;
  text-align: left;
}
.search-trigger__kbd {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 5px;
  border: 1px solid var(--nd-border);
  font-family: inherit;
}
.topbar__actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}
.upload-btn {
  height: 36px;
}
.topbar__icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--nd-text-secondary);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 160ms ease, color 160ms ease;
}
.topbar__icon:hover {
  color: var(--nd-text);
  background: var(--nd-surface-hover);
}
.topbar__star:hover {
  color: #fbbf24;
}
.user-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 8px 0 4px;
  border-radius: 999px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--nd-text);
  cursor: pointer;
  font-size: 13px;
  transition: background 160ms ease;
}
.user-chip:hover {
  background: var(--nd-surface-hover);
}
.user-chip__avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--nd-gradient);
  color: #fff;
  display: grid;
  place-items: center;
  font-size: 12px;
  font-weight: 600;
  flex: none;
}
.user-chip__name {
  max-width: 92px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-chip__caret {
  font-size: 11px;
  color: var(--nd-text-secondary);
}

.content {
  flex: 1;
  min-height: 0;
}

.layout__mask {
  position: fixed;
  inset: 0;
  z-index: 25;
  background: rgba(3, 7, 18, 0.55);
  backdrop-filter: blur(2px);
}

/* ---------- 拖拽遮罩 ---------- */
.drop-mask {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: grid;
  place-items: center;
  background: rgba(6, 9, 20, 0.66);
  backdrop-filter: blur(4px);
  pointer-events: none;
}
.drop-mask__box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 36px 48px;
  border-style: dashed;
  border-width: 2px;
  border-color: var(--nd-primary);
  color: var(--nd-text);
  font-size: 15px;
}
.drop-mask__box .el-icon {
  font-size: 40px;
  color: var(--nd-cyan);
}
.drop-mask__box .text-secondary {
  font-size: 12.5px;
}
.drop-enter-active,
.drop-leave-active {
  transition: opacity 160ms ease;
}
.drop-enter-from,
.drop-leave-to {
  opacity: 0;
}

/* ---------- 响应式 ---------- */
.view-enter-active,
.view-leave-active {
  transition: opacity 180ms ease, transform 180ms ease;
}
.view-enter-from,
.view-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

@media (max-width: 1199px) {
  .sidebar {
    width: var(--nd-sidebar-collapsed);
    padding: 18px 10px 14px;
    align-items: center;
  }
  .sidebar__brand-text,
  .nav__label,
  .space,
  .nav__badge {
    display: none;
  }
  .sidebar__brand {
    padding: 4px 0 10px;
  }
  .nav {
    align-items: center;
    width: 100%;
  }
  .nav__item {
    justify-content: center;
    padding: 11px 0;
    width: 100%;
    font-size: 17px;
  }
  .sidebar__foot {
    align-items: center;
    width: 100%;
  }
}

@media (max-width: 767px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    width: var(--nd-sidebar-width);
    align-items: stretch;
    transform: translateX(-100%);
    background: var(--el-bg-color-overlay, #131a2e);
    padding: 18px 14px 16px;
  }
  .sidebar.is-open {
    transform: translateX(0);
  }
  .sidebar.is-open .sidebar__brand-text,
  .sidebar.is-open .nav__label,
  .sidebar.is-open .nav__badge {
    display: block;
  }
  .sidebar.is-open .space {
    display: flex;
  }
  .sidebar.is-open .nav__item {
    justify-content: flex-start;
    padding: 10px 12px;
  }
  .sidebar.is-open .nav {
    align-items: stretch;
  }
  .topbar {
    padding: 0 12px;
    gap: 8px;
  }
  .topbar__menu {
    display: grid;
  }
  .search-trigger {
    flex: none;
    width: 36px;
    padding: 0;
    justify-content: center;
  }
  .search-trigger__text,
  .search-trigger__kbd,
  .upload-btn__text,
  .user-chip__name,
  .user-chip__caret {
    display: none;
  }
  .user-chip {
    padding: 0 4px 0 4px;
  }
  /* 手机端选择文件时，顶部工具栏让位给列表的选择头部（取消 / 已选 / 全选） */
  .layout.is-selecting .topbar {
    display: none;
  }
}
</style>
