import { reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as fileApi from '@/api/file'
import { downloadUrl as triggerUrl, triggerDownload, filenameFromDisposition } from '@/utils/download'
import { canPreview, isDir } from '@/utils/fileType'

// 文件操作集合：预览 / 下载 / 收藏 / 分享 / 重命名 / 移动 / 删除
// 首页、收藏、最近、回收站共用，避免各视图重复实现弹窗与确认逻辑
export function useFileActions({ onChange, onOpenDir } = {}) {
  const state = reactive({
    previewOpen: false,
    previewFile: null,
    previewSiblings: [],
    shareOpen: false,
    shareFile: null,
    moveOpen: false,
    moveMode: 'move',
    moveItem: null,
    renameOpen: false,
    renameItem: null,
    renameValue: '',
    detailsOpen: false,
    detailsItem: null
  })

  // notify 统一回调，视图传入自己的刷新函数
  function notify() {
    if (typeof onChange === 'function') onChange()
  }

  // 记录最近访问：详情接口会按间隔节流刷新 last_access_at，失败不影响打开 / 下载
  function touchAccess(item) {
    if (!item?.id) return
    fileApi.touchAccess(item.id).catch(() => {})
  }

  function openPreview(item, siblings = []) {
    touchAccess(item)
    state.previewFile = item
    state.previewSiblings = siblings.length ? siblings : [item]
    state.previewOpen = true
  }

  function openItem(item, siblings = []) {
    if (!item) return
    if (isDir(item)) {
      if (typeof onOpenDir === 'function') onOpenDir(item)
      return
    }
    if (canPreview(item)) openPreview(item, siblings)
    else confirmDownload(item)
  }

  // 不支持的格式不直接下载：先提示，用户确认后再下载
  async function confirmDownload(item) {
    if (!item) return
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
    download(item)
  }

  function download(item) {
    if (!item) return
    touchAccess(item)
    triggerUrl(fileApi.downloadUrl(item.id))
  }

  async function downloadBatch(ids, fallbackName = 'novadrive-files.zip') {
    if (!ids.length) return
    try {
      const res = await fileApi.batchDownload(ids)
      const name = filenameFromDisposition(res.headers['content-disposition'], fallbackName)
      triggerDownload(res.data, name)
    } catch {
      ElMessage.error('批量下载失败')
    }
  }

  async function toggleFavorite(item) {
    if (!item) return
    const next = Number(item.is_favorite) === 1 ? 0 : 1
    item.is_favorite = next
    try {
      await fileApi.setFavorite(item.id, next === 1)
    } catch {
      item.is_favorite = next === 1 ? 0 : 1
    }
  }

  function askRename(item) {
    if (!item) return
    state.renameItem = item
    state.renameValue = item.name
    state.renameOpen = true
  }

  async function confirmRename() {
    const name = state.renameValue.trim()
    if (!state.renameItem || !name) return
    await fileApi.rename(state.renameItem.id, name)
    ElMessage.success('已重命名')
    state.renameOpen = false
    notify()
  }

  async function removeItem(item) {
    if (!item) return
    try {
      await ElMessageBox.confirm(
        isDir(item)
          ? `目录「${item.name}」及其内容将被移入回收站，确定删除？`
          : `确定将「${item.name}」移入回收站？`,
        '删除确认',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
    await fileApi.remove(item.id)
    ElMessage.success('已移入回收站')
    notify()
  }

  async function removeBatch(ids) {
    if (!ids.length) return
    try {
      await ElMessageBox.confirm(`确定将选中的 ${ids.length} 项移入回收站？`, '删除确认', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
      })
    } catch {
      return
    }
    // 逐个请求，部分失败也继续，最后统一提示
    const results = await Promise.allSettled(ids.map((id) => fileApi.remove(id)))
    const failed = results.filter((r) => r.status === 'rejected').length
    if (failed) ElMessage.warning(`${failed} 项删除失败`)
    else ElMessage.success('已移入回收站')
    notify()
  }

  async function restoreItem(item) {
    if (!item) return
    await fileApi.restore(item.id)
    ElMessage.success('已还原')
    notify()
  }

  async function forceDelete(item) {
    if (!item) return
    try {
      await ElMessageBox.confirm(`彻底删除「${item.name}」后将无法恢复，确定继续？`, '彻底删除', {
        type: 'error',
        confirmButtonText: '彻底删除',
        cancelButtonText: '取消'
      })
    } catch {
      return
    }
    await fileApi.forceDelete(item.id)
    ElMessage.success('已彻底删除')
    notify()
  }

  async function emptyRecycle() {
    try {
      await ElMessageBox.confirm('清空回收站后所有文件将无法恢复，确定继续？', '清空回收站', {
        type: 'error',
        confirmButtonText: '清空',
        cancelButtonText: '取消'
      })
    } catch {
      return
    }
    await fileApi.emptyRecycle()
    ElMessage.success('回收站已清空')
    notify()
  }

  function askMove(item, mode = 'move') {
    if (!item) return
    state.moveItem = item
    state.moveMode = mode
    state.moveOpen = true
  }

  async function confirmMove({ targetId, newName }) {
    const item = state.moveItem
    if (!item) return
    if (state.moveMode === 'copy') {
      await fileApi.copy(item.id, targetId, newName)
      ElMessage.success('已复制')
    } else {
      await fileApi.move(item.id, targetId)
      ElMessage.success('已移动')
    }
    notify()
  }

  function openShare(item) {
    if (!item) return
    state.shareFile = item
    state.shareOpen = true
  }

  function openDetails(item) {
    if (!item) return
    state.detailsItem = item
    state.detailsOpen = true
  }

  // handleAction 处理 FileGrid / FileTable / FileActionMenu 抛出的统一动作
  async function handleAction(key, item, siblings = []) {
    switch (key) {
      case 'open':
        openItem(item, siblings)
        break
      case 'preview':
        openPreview(item, siblings)
        break
      case 'download':
        download(item)
        break
      case 'favorite':
        await toggleFavorite(item)
        break
      case 'share':
        openShare(item)
        break
      case 'details':
        openDetails(item)
        break
      case 'rename':
        askRename(item)
        break
      case 'move':
        askMove(item, 'move')
        break
      case 'copy':
        askMove(item, 'copy')
        break
      case 'delete':
        await removeItem(item)
        break
      case 'restore':
        await restoreItem(item)
        break
      case 'forceDelete':
        await forceDelete(item)
        break
      default:
        break
    }
  }

  return {
    state,
    openItem,
    openPreview,
    download,
    confirmDownload,
    downloadBatch,
    toggleFavorite,
    askRename,
    confirmRename,
    removeItem,
    removeBatch,
    restoreItem,
    forceDelete,
    emptyRecycle,
    askMove,
    confirmMove,
    openShare,
    openDetails,
    handleAction
  }
}
