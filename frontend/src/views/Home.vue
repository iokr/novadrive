<template>
  <div class="nd-page home" :class="{ 'is-selecting': fileStore.hasSelection }">
    <!-- 手机端选择态头部：取消 / 已选中数量 / 全选，未选中时不出现 -->
    <header v-if="fileStore.hasSelection" class="home__select-head">
      <button type="button" class="home__select-btn" @click="fileStore.clearSelection()">取消</button>
      <p class="home__select-title">已选中 <b class="num">{{ fileStore.selected.length }}</b> 个文件</p>
      <button type="button" class="home__select-btn is-strong" @click="fileStore.toggleSelectAll()">
        {{ allSelected ? '取消全选' : '全选' }}
      </button>
    </header>

    <header class="home__bar">
      <div class="home__actions">
        <el-dropdown trigger="click" @command="onUploadCommand">
          <el-button type="primary" class="home__upload">
            <el-icon><UploadFilled /></el-icon>
            <span>上传</span>
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

        <el-button @click="openNewFolder">
          <el-icon><FolderAdd /></el-icon>
          <span>新建文件夹</span>
        </el-button>
      </div>

      <div class="home__tools">
        <el-input
          v-model="searchInput"
          class="home__search"
          placeholder="搜索当前目录"
          clearable
          :prefix-icon="Search"
          @keyup.enter="applyKeyword"
          @clear="applyKeyword"
        />

        <el-select v-model="sortValue" class="home__sort" @change="applySort">
          <el-option v-for="opt in sortOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>

        <div class="view-switch">
          <button
            type="button"
            :class="{ 'is-on': fileStore.viewMode === 'grid' }"
            title="网格视图"
            @click="fileStore.setViewMode('grid')"
          >
            <el-icon><Grid /></el-icon>
          </button>
          <button
            type="button"
            :class="{ 'is-on': fileStore.viewMode === 'list' }"
            title="列表视图"
            @click="fileStore.setViewMode('list')"
          >
            <el-icon><Menu /></el-icon>
          </button>
        </div>
      </div>
    </header>

    <div class="home__path">
      <Breadcrumb :items="crumbs" :current="fileStore.parentId" @navigate="navigate" />
      <p class="home__hint text-secondary num">
        <template v-if="keyword">「{{ keyword }}」的搜索结果 · 共 {{ fileStore.total }} 项</template>
        <template v-else>{{ fileStore.total }} 个项目 · {{ viewLabel }}</template>
      </p>
    </div>

    <div class="home__bulk" :class="{ 'is-on': fileStore.hasSelection }">
      <template v-if="fileStore.hasSelection">
        <span class="home__bulk-count">已选择 <b class="num">{{ fileStore.selected.length }}</b> 项</span>
        <el-button size="small" @click="downloadSelected">
          <el-icon><Download /></el-icon>
          <span>下载选中</span>
        </el-button>
        <el-button size="small" type="danger" plain @click="removeSelected">
          <el-icon><Delete /></el-icon>
          <span>删除</span>
        </el-button>
        <el-button size="small" text @click="fileStore.clearSelection()">取消选择</el-button>
      </template>
      <template v-else>
        <span class="home__bulk-tip text-muted">
          <el-icon><InfoFilled /></el-icon>
          支持拖拽文件或文件夹到页面直接上传
        </span>
        <el-button v-if="fileStore.viewMode === 'grid' && fileStore.items.length" size="small" text @click="fileStore.toggleSelectAll()">
          全选
        </el-button>
        <el-button size="small" text :loading="fileStore.loading" @click="reload">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </template>
    </div>

    <FileGrid
      v-if="fileStore.viewMode === 'grid'"
      :items="fileStore.items"
      :selected="fileStore.selected"
      :loading="fileStore.loading"
      :empty-title="emptyTitle"
      :empty-desc="emptyDesc"
      @open="onOpen"
      @action="onAction"
      @toggle-select="fileStore.toggleSelect"
      @toggle-favorite="actions.toggleFavorite"
    />

    <FileTable
      v-else
      :items="fileStore.items"
      :selected="fileStore.selected"
      :loading="fileStore.loading"
      :sort="fileStore.sort"
      :order="fileStore.order"
      :empty-text="emptyTitle"
      @open="onOpen"
      @action="onAction"
      @toggle-select="fileStore.toggleSelect"
      @toggle-select-all="fileStore.toggleSelectAll"
      @toggle-favorite="actions.toggleFavorite"
      @sort-change="onSortChange"
    />

    <input ref="fileInputRef" type="file" multiple class="hidden-input" @change="onPickFiles" />
    <input ref="folderInputRef" type="file" webkitdirectory multiple class="hidden-input" @change="onPickFolder" />

    <FileDialogs :actions="actions" />

    <MobileActionBar
      :visible="fileStore.hasSelection"
      :count="fileStore.selected.length"
      :item="singleSelected"
      @action="onBarAction"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import Breadcrumb from '@/components/Breadcrumb.vue'
import FileGrid from '@/components/FileGrid.vue'
import FileTable from '@/components/FileTable.vue'
import FileDialogs from '@/components/FileDialogs.vue'
import MobileActionBar from '@/components/MobileActionBar.vue'
import { useFileStore } from '@/stores/file'
import { useUploadStore } from '@/stores/upload'
import { useUserStore } from '@/stores/user'
import { useFileActions } from '@/composables/useFileActions'
import { filesFromInput } from '@/utils/folderReader'
import * as fileApi from '@/api/file'

const route = useRoute()
const router = useRouter()
const fileStore = useFileStore()
const upload = useUploadStore()
const user = useUserStore()

const fileInputRef = ref(null)
const folderInputRef = ref(null)
const searchInput = ref(fileStore.keyword)
const sortValue = ref(`${fileStore.sort}:${fileStore.order}`)

const sortOptions = [
  { label: '名称 A→Z', value: 'name:asc' },
  { label: '名称 Z→A', value: 'name:desc' },
  { label: '最近修改', value: 'updated_at:desc' },
  { label: '最早修改', value: 'updated_at:asc' },
  { label: '体积从大到小', value: 'size:desc' },
  { label: '体积从小到大', value: 'size:asc' }
]

const keyword = computed(() => fileStore.keyword)
const crumbs = computed(() => (fileStore.breadcrumb || []).filter((it) => Number(it.id) !== 0))
const viewLabel = computed(() => (fileStore.viewMode === 'grid' ? '网格视图' : '列表视图'))
const emptyTitle = computed(() => (keyword.value ? '没有找到匹配的文件' : '这里还空空如也'))
const emptyDesc = computed(() =>
  keyword.value ? '换个关键字试试，或清空筛选查看全部内容' : '拖拽文件到这里，或点击右上角「上传」开始'
)
const allSelected = computed(() => fileStore.allSelected)
const selectedItems = computed(() =>
  fileStore.items.filter((it) => fileStore.selected.includes(it.id))
)
const singleSelected = computed(() =>
  selectedItems.value.length === 1 ? selectedItems.value[0] : null
)

const actions = useFileActions({
  onChange: reload,
  onOpenDir: (item) => navigate(item.id)
})

// ---------- 数据加载 ----------
async function reload() {
  await fileStore.load(fileStore.parentId)
  await fileStore.loadBreadcrumb(fileStore.parentId)
}

async function navigate(parentId) {
  const id = Number(parentId) || 0
  fileStore.parentId = id
  fileStore.keyword = ''
  searchInput.value = ''
  if (id === 0) router.replace({ path: '/' })
  else router.replace({ path: '/', query: { folder: id } })
  await reload()
  user.fetchSpace()
}

function applyKeyword() {
  fileStore.keyword = searchInput.value.trim()
  fileStore.load(fileStore.parentId)
}

function applySort() {
  const [sort, order] = sortValue.value.split(':')
  fileStore.setSort(sort, order)
  fileStore.load(fileStore.parentId)
}

function onSortChange(sort, order) {
  sortValue.value = `${sort}:${order}`
  fileStore.setSort(sort, order)
  fileStore.load(fileStore.parentId)
}

// ---------- 交互 ----------
function onOpen(item) {
  // 选择态下点卡片 / 行只切换选中，不再打开文件
  if (fileStore.hasSelection) {
    fileStore.toggleSelect(item.id)
    return
  }
  actions.openItem(item, fileStore.items)
}

function onAction(key, item) {
  actions.handleAction(key, item, fileStore.items)
}

async function openNewFolder() {
  try {
    const { value } = await ElMessageBox.prompt('请输入文件夹名称', '新建文件夹', {
      confirmButtonText: '创建',
      cancelButtonText: '取消',
      inputPlaceholder: '例如：工作资料',
      inputValidator: (val) => {
        const name = String(val || '').trim()
        if (!name) return '名称不能为空'
        if (name.length > 255) return '名称过长'
        if (/[\\/:*?"<>|]/.test(name)) return '不能包含 \\ / : * ? " < > |'
        return true
      }
    })
    await fileApi.mkdir({ parent_id: fileStore.parentId, name: value.trim() })
    await reload()
  } catch {
    // 取消或请求失败
  }
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
    fileStore.parentId
  )
}

function onPickFolder(event) {
  const items = filesFromInput(event.target.files)
  event.target.value = ''
  if (!items.length) return
  upload.addFiles(items, fileStore.parentId)
}

async function downloadSelected() {
  const ids = [...fileStore.selected]
  if (!ids.length) return
  if (ids.length === 1) {
    const item = fileStore.items.find((it) => it.id === ids[0])
    if (item) actions.download(item)
    return
  }
  await actions.downloadBatch(ids)
}

async function removeSelected() {
  const ids = [...fileStore.selected]
  if (!ids.length) return
  await actions.removeBatch(ids)
  fileStore.clearSelection()
}

// 手机端底部操作栏：下载 / 删除支持批量，其余操作只对单个文件生效
async function onBarAction(key) {
  const items = selectedItems.value
  if (!items.length) return
  if (key === 'download') {
    await downloadSelected()
    return
  }
  if (key === 'delete') {
    await removeSelected()
    return
  }
  const item = items[0]
  if (key === 'favorite') {
    await actions.toggleFavorite(item)
    return
  }
  actions.handleAction(key, item, fileStore.items)
}

// ---------- 生命周期 ----------
onMounted(async () => {
  if (route.query.folder) fileStore.parentId = Number(route.query.folder) || 0
  await reload()
  user.fetchSpace()
})

watch(
  () => route.query.folder,
  (folder) => {
    if (folder === undefined) return
    const id = Number(folder) || 0
    if (id !== fileStore.parentId) {
      fileStore.parentId = id
      fileStore.keyword = ''
      searchInput.value = ''
      reload()
    }
  }
)

// 上传完成后刷新列表与配额（批量完成会有多次 revision，做 400ms 合并）
let refreshTimer = null
watch(
  () => upload.revision,
  () => {
    if (refreshTimer) clearTimeout(refreshTimer)
    refreshTimer = setTimeout(() => {
      reload()
      user.fetchSpace()
    }, 400)
  }
)
</script>

<style scoped>
.home__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.home__actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.home__upload {
  padding-left: 14px;
  padding-right: 14px;
}
.home__path {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 12px 0 8px;
  border-bottom: 1px solid var(--nd-border);
}
.home__hint {
  font-size: 12px;
  color: var(--nd-text-muted);
  margin-left: auto;
}
.home__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.home__search {
  width: 220px;
}
.home__sort {
  width: 148px;
}
.view-switch {
  display: flex;
  padding: 2px;
  border-radius: 8px;
  background: var(--nd-surface-sunken);
}
.view-switch button {
  width: 30px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--nd-text-secondary);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 160ms ease, color 160ms ease;
}
.view-switch button:hover {
  color: var(--nd-text);
}
.view-switch button.is-on {
  background: var(--nd-surface);
  color: var(--nd-primary);
  box-shadow: var(--nd-shadow-sm);
}
.home__bulk {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  margin: 10px 0 12px;
  padding: 0 4px;
  flex-wrap: wrap;
}
.home__bulk.is-on {
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid #c4ddff;
  background: var(--nd-primary-soft);
}
.home__bulk-count {
  font-size: 13px;
  margin-right: 4px;
}
.home__bulk-tip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  margin-right: auto;
}
.hidden-input {
  display: none;
}
/* 选择态头部默认隐藏，只在手机端选择文件后出现 */
.home__select-head {
  display: none;
}
.home__select-btn {
  border: none;
  background: transparent;
  padding: 4px 2px;
  color: var(--nd-primary);
  font-size: 14px;
  cursor: pointer;
}
.home__select-btn.is-strong {
  font-weight: 600;
}
.home__select-title {
  font-size: 15px;
  font-weight: 600;
}
@media (max-width: 767px) {
  .home__search,
  .home__sort {
    width: 100%;
  }
  .home__tools {
    width: 100%;
  }
  /* 选择态：顶部工具栏收起，让位给选择头部；操作统一收进底部操作栏 */
  .home.is-selecting {
    padding-bottom: 176px;
  }
  .home.is-selecting .home__bar,
  .home.is-selecting .home__bulk {
    display: none;
  }
  .home__select-head {
    position: sticky;
    top: 0;
    z-index: 22;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin: -14px -12px 10px;
    padding: 12px;
    background: var(--nd-surface);
    border-bottom: 1px solid var(--nd-border);
  }
}
</style>
