<template>
  <div class="admin-layout">
    <aside class="admin-side">
      <router-link to="/" class="admin-brand nd-logo">
        <span class="nd-logo__mark">
          <el-icon><Cloudy /></el-icon>
        </span>
        <span class="admin-brand__text">
          NovaDrive
          <small>控制台</small>
        </span>
      </router-link>

      <nav class="admin-nav">
        <router-link to="/admin" class="admin-nav__item" :class="{ 'is-active': route.name === 'admin-dashboard' }">
          <el-icon><DataLine /></el-icon>
          <span>数据概览</span>
        </router-link>
        <router-link
          to="/admin/users"
          class="admin-nav__item"
          :class="{ 'is-active': route.name === 'admin-users' }"
        >
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </router-link>
      </nav>

      <div class="admin-side__foot">
        <router-link to="/" class="admin-nav__item">
          <el-icon><Back /></el-icon>
          <span>返回网盘</span>
        </router-link>
        <button type="button" class="admin-nav__item admin-nav__item--btn" @click="theme.toggle()">
          <el-icon><Sunny v-if="theme.isDark" /><Moon v-else /></el-icon>
          <span>{{ theme.isDark ? '浅色主题' : '深色主题' }}</span>
        </button>
        <div class="admin-user">
          <span class="admin-user__avatar">{{ user.avatarText }}</span>
          <div class="admin-user__meta">
            <p class="ellipsis">{{ user.displayName }}</p>
            <p class="text-muted">管理员</p>
          </div>
        </div>
      </div>
    </aside>

    <div class="admin-main">
      <header class="admin-topbar">
        <div>
          <p class="admin-topbar__title">{{ route.meta.title || '管理后台' }}</p>
          <p class="admin-topbar__sub text-secondary">NovaDrive · 星云盘运维视角</p>
        </div>
        <el-button text @click="refresh">
          <el-icon><Refresh /></el-icon>
          <span>刷新</span>
        </el-button>
      </header>

      <main class="admin-content">
        <router-view v-slot="{ Component }">
          <transition name="view" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup>
import { provide, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'

const route = useRoute()
const user = useUserStore()
const theme = useThemeStore()

// 顶栏刷新按钮：子页面通过 inject 注册自己的刷新方法
const refresher = ref(null)
provide('admin-refresh', (fn) => {
  refresher.value = fn
})

function refresh() {
  if (typeof refresher.value === 'function') refresher.value()
}
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
}
.admin-side {
  width: 232px;
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 14px 16px;
  border-right: 1px solid var(--nd-border);
  background: var(--nd-surface);
  position: sticky;
  top: 0;
  height: 100vh;
}
.admin-brand {
  padding: 0 6px;
}
.admin-brand__text {
  display: flex;
  flex-direction: column;
  font-size: 15px;
  line-height: 1.2;
}
.admin-brand__text small {
  font-size: 11px;
  font-weight: 400;
  color: var(--nd-text-muted);
  margin-top: 2px;
}
.admin-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}
.admin-nav__item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 8px;
  color: var(--nd-text-secondary);
  font-size: 13.5px;
  cursor: pointer;
  transition: background 160ms ease, color 160ms ease;
}
.admin-nav__item:hover {
  background: var(--nd-surface-hover);
  color: var(--nd-text);
}
.admin-nav__item.is-active {
  color: var(--nd-primary);
  background: var(--nd-primary-soft);
  font-weight: 600;
}
.admin-nav__item--btn {
  border: none;
  background: transparent;
  width: 100%;
  text-align: left;
  font: inherit;
}
.admin-side__foot {
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-top: 1px solid var(--nd-border);
  padding-top: 12px;
}
.admin-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--nd-surface-sunken);
  margin-top: 4px;
}
.admin-user__avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--nd-gradient);
  color: #fff;
  display: grid;
  place-items: center;
  font-size: 13px;
  font-weight: 600;
  flex: none;
}
.admin-user__meta {
  min-width: 0;
  font-size: 12.5px;
}
.admin-user__meta p:last-child {
  font-size: 11px;
}
.admin-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.admin-topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--nd-border);
  background: var(--nd-surface);
}
.admin-topbar__title {
  font-size: 18px;
  font-weight: 650;
}
.admin-topbar__sub {
  font-size: 12px;
  margin-top: 3px;
}
.admin-content {
  flex: 1;
  min-height: 0;
  padding: 20px 24px 40px;
}
.view-enter-active,
.view-leave-active {
  transition: opacity 180ms ease, transform 180ms ease;
}
.view-enter-from,
.view-leave-to {
  opacity: 0;
  transform: translateY(6px);
}
@media (max-width: 900px) {
  .admin-side {
    width: 72px;
    padding: 18px 10px 14px;
    align-items: center;
  }
  .admin-brand__text,
  .admin-nav__item span,
  .admin-user__meta {
    display: none;
  }
  .admin-nav__item {
    justify-content: center;
    font-size: 17px;
    width: 100%;
  }
  .admin-user {
    padding: 6px;
    justify-content: center;
  }
  .admin-content,
  .admin-topbar {
    padding-left: 14px;
    padding-right: 14px;
  }
}
</style>
