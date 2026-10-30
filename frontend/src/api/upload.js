import request from './request'

// 小文件直传：multipart/form-data（parent_id / file / relative_path）
export const simple = (file, parentId, relativePath = '', onProgress) => {
  const form = new FormData()
  form.append('parent_id', String(parentId ?? 0))
  form.append('file', file, file.name)
  if (relativePath) form.append('relative_path', relativePath)
  return request.post('/files/upload', form, {
    onUploadProgress: (e) => {
      if (onProgress && e.total) onProgress(e.loaded / e.total)
    }
  })
}

// 批量 / 文件夹上传：files[] + manifest
export const batch = (items, parentId, onProgress) => {
  const form = new FormData()
  form.append('parent_id', String(parentId ?? 0))
  const manifest = items.map((item, index) => ({
    index,
    relative_path: item.relativePath || item.file.name
  }))
  items.forEach((item) => form.append('files', item.file, item.file.name))
  form.append('manifest', JSON.stringify(manifest))
  return request.post('/files/upload/batch', form, {
    onUploadProgress: (e) => {
      if (onProgress && e.total) onProgress(e.loaded / e.total)
    }
  })
}

// ---------- 分片上传 ----------

export const init = (payload) => request.post('/files/upload/init', payload)

export const chunk = (uploadId, index, blob, onProgress) =>
  request.put(`/files/upload/${uploadId}/chunk`, blob, {
    params: { index },
    headers: { 'Content-Type': 'application/octet-stream' },
    // 分片失败由上传面板统一展示并自动重试，不弹全局提示
    silent: true,
    onUploadProgress: (e) => {
      if (onProgress && e.total) onProgress(e.loaded / e.total)
    }
  })

export const status = (uploadId) => request.get(`/files/upload/${uploadId}/status`, { silent: true })

export const complete = (uploadId, hashMd5) =>
  request.post(`/files/upload/${uploadId}/complete`, { hash_md5: hashMd5 })

export const cancel = (uploadId) => request.delete(`/files/upload/${uploadId}`)
