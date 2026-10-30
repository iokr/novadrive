import { defineStore } from 'pinia'
import * as fileApi from '@/api/file'

const VIEW_KEY = 'novadrive_view_mode'

// 文件列表状态（首页 / 收藏 / 最近 共用列表能力）
export const useFileStore = defineStore('file', {
  state: () => ({
    parentId: 0,
    items: [],
    total: 0,
    loading: false,
    keyword: '',
    sort: 'name',
    order: 'asc',
    viewMode: localStorage.getItem(VIEW_KEY) || 'list',
    breadcrumb: [],
    selected: []
  }),
  getters: {
    imageItems: (s) => s.items.filter((it) => Number(it.is_dir) === 0 && it.has_thumb),
    hasSelection: (s) => s.selected.length > 0,
    allSelected: (s) => s.items.length > 0 && s.selected.length === s.items.length
  },
  actions: {
    setViewMode(mode) {
      this.viewMode = mode === 'list' ? 'list' : 'grid'
      localStorage.setItem(VIEW_KEY, this.viewMode)
    },
    setSort(sort, order) {
      this.sort = sort
      this.order = order
    },
    async load(parentId = this.parentId) {
      this.parentId = Number(parentId) || 0
      this.loading = true
      this.clearSelection()
      try {
        const data = await fileApi.list({
          parent_id: this.parentId,
          keyword: this.keyword || undefined,
          sort: this.sort,
          order: this.order
        })
        this.items = data?.items || []
        this.total = data?.total || 0
        return this.items
      } finally {
        this.loading = false
      }
    },
    async loadBreadcrumb(parentId = this.parentId) {
      try {
        this.breadcrumb = (await fileApi.breadcrumb(parentId)) || []
      } catch {
        this.breadcrumb = []
      }
      return this.breadcrumb
    },
    async toggleFavorite(item) {
      const next = Number(item.is_favorite) === 1 ? 0 : 1
      // 乐观更新，失败回滚
      item.is_favorite = next
      try {
        await fileApi.setFavorite(item.id, next === 1)
      } catch {
        item.is_favorite = next === 1 ? 0 : 1
      }
    },
    setItems(items) {
      this.items = items || []
      this.total = this.items.length
    },
    toggleSelect(id) {
      const idx = this.selected.indexOf(id)
      if (idx >= 0) this.selected.splice(idx, 1)
      else this.selected.push(id)
    },
    toggleSelectAll() {
      if (this.allSelected) this.selected = []
      else this.selected = this.items.map((it) => it.id)
    },
    clearSelection() {
      this.selected = []
    },
    reset() {
      this.parentId = 0
      this.items = []
      this.total = 0
      this.keyword = ''
      this.breadcrumb = []
      this.selected = []
    }
  }
})
