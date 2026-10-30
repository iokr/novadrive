<template>
  <AuthPanel>
    <h2 class="title">欢迎回来</h2>
    <p class="subtitle text-secondary">登录后继续管理你的文件</p>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      class="form"
      @submit.prevent="submit"
    >
      <el-form-item label="用户名 / 邮箱" prop="account">
        <el-input
          v-model="form.account"
          size="large"
          placeholder="请输入用户名或邮箱"
          autocomplete="username"
          :prefix-icon="User"
        />
      </el-form-item>

      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          size="large"
          show-password
          placeholder="请输入密码"
          autocomplete="current-password"
          :prefix-icon="Lock"
          @keyup.enter="submit"
        />
      </el-form-item>

      <div class="form__row">
        <el-checkbox v-model="remember">记住账号</el-checkbox>
        <span class="text-muted form__hint">忘记密码请联系管理员重置</span>
      </div>

      <el-button type="primary" size="large" class="form__submit" :loading="loading" @click="submit">
        登录
      </el-button>
    </el-form>

    <p class="switch text-secondary">
      还没有账号？
      <router-link to="/register" class="switch__link">立即注册</router-link>
    </p>
  </AuthPanel>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import AuthPanel from '@/components/AuthPanel.vue'
import { useUserStore } from '@/stores/user'

const REMEMBER_KEY = 'novadrive_last_account'

const route = useRoute()
const router = useRouter()
const user = useUserStore()

const formRef = ref(null)
const loading = ref(false)
const remember = ref(true)
const form = reactive({ account: '', password: '' })

const rules = {
  account: [{ required: true, message: '请输入用户名或邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

onMounted(() => {
  const saved = localStorage.getItem(REMEMBER_KEY)
  if (saved) form.account = saved
})

async function submit() {
  if (loading.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await user.login({ username: form.account.trim(), password: form.password })
    if (remember.value) localStorage.setItem(REMEMBER_KEY, form.account.trim())
    else localStorage.removeItem(REMEMBER_KEY)
    ElMessage.success(`欢迎回来，${user.displayName}`)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    // 只允许站内相对路径，避免开放重定向
    router.replace(redirect.startsWith('/') ? redirect : '/')
  } catch {
    // 登录失败提示由请求拦截器统一展示
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.title {
  font-size: 22px;
  font-weight: 700;
}
.subtitle {
  margin-top: 6px;
  font-size: 13px;
}
.form {
  margin-top: 22px;
}
.form__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 18px;
}
.form__hint {
  font-size: 11.5px;
}
.form__submit {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
}
.switch {
  margin-top: 18px;
  text-align: center;
  font-size: 13px;
}
.switch__link {
  color: var(--nd-cyan);
  font-weight: 600;
}
.switch__link:hover {
  text-decoration: underline;
}
</style>
