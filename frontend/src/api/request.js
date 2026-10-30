import axios from 'axios'
import { ElMessage } from 'element-plus'

export const TOKEN_KEY = 'novadrive_token'

export const getToken = () => localStorage.getItem(TOKEN_KEY) || ''

export const setToken = (token) => {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export const clearToken = () => localStorage.removeItem(TOKEN_KEY)

// 业务错误码：1002 未登录 / 1004 非管理员，需要特殊处理
export const CODE_UNAUTHORIZED = 1002
export const CODE_FORBIDDEN = 1003
export const CODE_NOT_ADMIN = 1004

const service = axios.create({
  baseURL: '/api/v1',
  timeout: 0 // 大文件上传 / 下载不限制
})

service.interceptors.request.use((config) => {
  const token = getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

service.interceptors.response.use(
  (response) => {
    // 二进制响应（zip / 原图）原样透出，由调用方处理文件名
    if (response.config.responseType === 'blob') {
      const type = response.headers['content-type'] || ''
      if (type.includes('application/json')) {
        return response.data.text().then((text) => {
          const body = safeParse(text)
          return Promise.reject(handleBusinessError(body, response.config))
        })
      }
      return response
    }

    const body = response.data
    if (!body || typeof body !== 'object') return body
    if (body.code === 0) return body.data
    return Promise.reject(handleBusinessError(body, response.config))
  },
  (error) => {
    if (axios.isCancel(error)) return Promise.reject(error)
    if (error.code === 'ERR_CANCELED') return Promise.reject(error)
    const silent = error.config?.silent
    if (error.response?.status === 401) {
      clearToken()
      redirectToLogin()
      return Promise.reject(error)
    }
    const message = error.response
      ? `请求失败（HTTP ${error.response.status}）`
      : '网络异常，请检查连接后重试'
    if (!silent) ElMessage.error(message)
    return Promise.reject(error)
  }
)

function safeParse(text) {
  try {
    return JSON.parse(text)
  } catch {
    return { code: 5000, message: text || '未知错误' }
  }
}

function handleBusinessError(body, config) {
  const code = body?.code ?? 5000
  const message = body?.message || '请求失败'

  if (code === CODE_UNAUTHORIZED) {
    clearToken()
    redirectToLogin()
  } else if (!config?.silent) {
    ElMessage.error(message)
  }

  const error = new Error(message)
  error.code = code
  return error
}

function redirectToLogin() {
  const path = window.location.pathname
  if (path.startsWith('/login') || path.startsWith('/register') || path.startsWith('/s/')) return
  const redirect = encodeURIComponent(path + window.location.search)
  window.location.replace(`/login?redirect=${redirect}`)
}

export default service
