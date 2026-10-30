import request from './request'

// 用户列表：{ page, size, keyword } → { total, page, size, items }
export const users = (params) => request.get('/admin/users', { params })

export const setStatus = (id, status) => request.put(`/admin/users/${id}/status`, { status })

export const setQuota = (id, storageQuota) =>
  request.put(`/admin/users/${id}/quota`, { storage_quota: storageQuota })

export const stats = () => request.get('/admin/stats')

// 上传会话监控：status 省略表示全部
export const uploadSessions = (params) => request.get('/admin/upload-sessions', { params })
