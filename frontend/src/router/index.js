import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const MainLayout = () => import('@/layouts/MainLayout.vue')
const AdminLayout = () => import('@/layouts/AdminLayout.vue')

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { guestOnly: true, title: '登录' }
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/Register.vue'),
    meta: { guestOnly: true, title: '注册' }
  },
  {
    path: '/s/:token',
    name: 'share-access',
    component: () => import('@/views/ShareAccess.vue'),
    meta: { title: '分享' }
  },
  {
    path: '/',
    component: MainLayout,
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'home', component: () => import('@/views/Home.vue'), meta: { title: '我的文件' } },
      { path: 'favorites', name: 'favorites', component: () => import('@/views/Favorites.vue'), meta: { title: '收藏' } },
      { path: 'recent', name: 'recent', component: () => import('@/views/Recent.vue'), meta: { title: '最近访问' } },
      { path: 'recycle', name: 'recycle', component: () => import('@/views/Recycle.vue'), meta: { title: '回收站' } },
      { path: 'shares', name: 'shares', component: () => import('@/views/Shares.vue'), meta: { title: '我的分享' } },
      { path: 'profile', name: 'profile', component: () => import('@/views/Profile.vue'), meta: { title: '个人中心' } }
    ]
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { requiresAuth: true, requiresAdmin: true },
    children: [
      {
        path: '',
        name: 'admin-dashboard',
        component: () => import('@/views/admin/Dashboard.vue'),
        meta: { title: '管理概览' }
      },
      {
        path: 'users',
        name: 'admin-users',
        component: () => import('@/views/admin/Users.vue'),
        meta: { title: '用户管理' }
      }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

router.beforeEach(async (to) => {
  const user = useUserStore()

  if (to.meta.requiresAuth && !user.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly && user.isLoggedIn) {
    return { name: 'home' }
  }

  if (to.meta.requiresAdmin) {
    // 刷新页面后需要先拉取用户信息才能判断角色
    const info = await user.fetchProfile()
    if (!info || Number(info.role) !== 1) {
      return { name: 'home' }
    }
  } else if (user.isLoggedIn && !user.info) {
    await user.fetchProfile()
  }

  return true
})

router.afterEach((to) => {
  const title = to.meta.title ? `${to.meta.title} · NovaDrive` : 'NovaDrive · 星云盘'
  document.title = title
})

export default router
