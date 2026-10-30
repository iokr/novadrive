import request from './request'

const KEY_PREFIX = 'novadrive_share_key:'

export const getShareKey = (token) => sessionStorage.getItem(KEY_PREFIX + token) || ''

export const setShareKey = (token, key) => sessionStorage.setItem(KEY_PREFIX + token, key)

export const clearShareKey = (token) => sessionStorage.removeItem(KEY_PREFIX + token)

const keyHeader = (token) => {
  const key = getShareKey(token)
  return key ? { 'X-Share-Key': key } : {}
}

// ---------- 我的分享 ----------

export const create = (payload) => request.post('/shares', payload)

export const mine = () => request.get('/shares')

export const cancel = (id) => request.delete(`/shares/${id}`)

// ---------- 公开访问（免登录） ----------

export const info = (token) => request.get(`/shares/public/${token}`)

export const verify = (token, password) =>
  request.post(
    `/shares/public/${token}/verify`,
    { password },
    { headers: keyHeader(token) }
  )

export const files = (token, parentId = 0) =>
  request.get(`/shares/public/${token}/files`, {
    params: { parent_id: parentId },
    headers: keyHeader(token)
  })

// ---------- 公开流地址（img / video / 下载 走 query 传 access_key） ----------

const accessQuery = (token) => {
  const key = getShareKey(token)
  return key ? `&access_key=${encodeURIComponent(key)}` : ''
}

export const previewUrl = (token, fileId) =>
  `/api/v1/shares/public/${token}/preview/${fileId}?x=1${accessQuery(token)}`

export const downloadUrl = (token, fileId) =>
  `/api/v1/shares/public/${token}/download?file_id=${fileId}${accessQuery(token)}`
