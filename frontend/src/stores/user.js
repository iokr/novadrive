import { defineStore } from 'pinia'
import * as authApi from '@/api/auth'
import * as userApi from '@/api/user'
import { getToken, setToken, clearToken } from '@/api/request'

// 当前用户 / 空间用量
export const useUserStore = defineStore('user', {
  state: () => ({
    token: getToken(),
    info: null,
    space: { used: 0, quota: 0, file_count: 0 },
    loading: false
  }),
  getters: {
    isLoggedIn: (s) => !!s.token,
    isAdmin: (s) => Number(s.info?.role) === 1,
    displayName: (s) => s.info?.nickname || s.info?.username || '未登录',
    avatarText: (s) => (s.info?.nickname || s.info?.username || 'N').slice(0, 1).toUpperCase(),
    usedPercent: (s) => {
      const quota = Number(s.space.quota) || 0
      if (quota <= 0) return 0
      return Math.min(100, Math.round((Number(s.space.used) / quota) * 1000) / 10)
    }
  },
  actions: {
    async login(payload) {
      const data = await authApi.login(payload)
      this.token = data.token
      setToken(data.token)
      this.info = data.user
      await this.fetchSpace()
      return data
    },
    async register(payload) {
      return authApi.register(payload)
    },
    async fetchProfile(force = false) {
      if (!this.token) return null
      if (this.info && !force) return this.info
      this.loading = true
      try {
        this.info = await authApi.profile()
        return this.info
      } finally {
        this.loading = false
      }
    },
    async updateProfile(payload) {
      this.info = await authApi.updateProfile(payload)
      return this.info
    },
    async changePassword(payload) {
      return authApi.changePassword(payload)
    },
    async fetchSpace() {
      if (!this.token) return
      try {
        this.space = await userApi.storage()
        // 同步配额到用户信息，避免两处不一致
        if (this.info) {
          this.info.storage_used = this.space.used
          this.info.storage_quota = this.space.quota
        }
      } catch {
        // 空间信息失败不阻断主流程
      }
    },
    logout() {
      this.token = ''
      this.info = null
      this.space = { used: 0, quota: 0, file_count: 0 }
      clearToken()
    }
  }
})
