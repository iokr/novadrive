import { defineStore } from 'pinia'

const THEME_KEY = 'novadrive_theme_v2'

// 主题 store：默认浅色（白底网盘观感），可切换深色（CSS 变量驱动，见 styles/theme.scss）
export const useThemeStore = defineStore('theme', {
  state: () => ({
    mode: localStorage.getItem(THEME_KEY) === 'dark' ? 'dark' : 'light'
  }),
  getters: {
    isDark: (s) => s.mode === 'dark'
  },
  actions: {
    apply() {
      const root = document.documentElement
      root.classList.toggle('dark', this.mode === 'dark')
      root.classList.toggle('light', this.mode === 'light')
    },
    setMode(mode) {
      this.mode = mode === 'dark' ? 'dark' : 'light'
      localStorage.setItem(THEME_KEY, this.mode)
      this.apply()
    },
    toggle() {
      this.setMode(this.mode === 'dark' ? 'light' : 'dark')
    }
  }
})
