import request, { getToken } from './request'

// ---------- 列表与统计 ----------

export const list = (params) => request.get('/files', { params })

export const breadcrumb = (parentId) => request.get('/files/breadcrumb', { params: { parent_id: parentId } })

export const recent = (limit = 50) => request.get('/files/recent', { params: { limit } })

export const favorites = () => request.get('/files/favorites')

export const stats = () => request.get('/files/stats')

export const detail = (id) => request.get(`/files/${id}`)

// 打开 / 下载文件时记录最近访问：详情接口内部会按节流间隔刷新 last_access_at，失败静默
export const touchAccess = (id) => request.get(`/files/${id}`, { silent: true })

// ---------- 目录与文件操作 ----------

export const mkdir = (payload) => request.post('/files/mkdir', payload)

export const rename = (id, name) => request.put(`/files/${id}/rename`, { name })

export const move = (id, targetParentId) =>
  request.put(`/files/${id}/move`, { target_parent_id: targetParentId })

export const copy = (id, targetParentId, newName = '') =>
  request.post(`/files/${id}/copy`, { target_parent_id: targetParentId, new_name: newName })

export const setFavorite = (id, isFavorite) =>
  request.put(`/files/${id}/favorite`, { is_favorite: isFavorite ? 1 : 0 }, { silent: true })

// ---------- 回收站 ----------

export const remove = (id) => request.delete(`/files/${id}`)

export const recycleList = () => request.get('/files/recycle')

export const restore = (id) => request.post(`/files/${id}/restore`)

export const forceDelete = (id) => request.delete(`/files/${id}/force`)

export const emptyRecycle = () => request.delete('/files/recycle')

// ---------- 秒传预检 ----------

export const check = (hashMd5, size) =>
  request.post('/files/check', { hash_md5: hashMd5, size }, { silent: true })

// ---------- 流式地址（浏览器直接加载，token 走 query） ----------

const withToken = (path) => {
  const token = getToken()
  return token ? `${path}${path.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}` : path
}

export const rawUrl = (id) => withToken(`/api/v1/files/${id}/raw`)

export const thumbUrl = (id) => withToken(`/api/v1/files/${id}/thumb`)

export const downloadUrl = (id) => withToken(`/api/v1/files/${id}/download`)

// 批量下载：POST 需带 body，先取 blob 再触发保存
export const batchDownload = (ids) =>
  request.post('/files/download/batch', { ids }, { responseType: 'blob', silent: true })
