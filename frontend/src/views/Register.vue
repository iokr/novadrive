<template>
  <AuthPanel>
    <h2 class="title">创建账号</h2>
    <p class="subtitle text-secondary">几秒钟即可拥有自己的私有云盘</p>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      class="form"
      @submit.prevent="submit"
    >
      <el-form-item label="用户名" prop="username">
        <el-input
          v-model="form.username"
          size="large"
          placeholder="3-32 位字母、数字或下划线"
          autocomplete="username"
          :prefix-icon="User"
        />
      </el-form-item>

      <el-form-item label="邮箱" prop="email">
        <el-input
          v-model="form.email"
          size="large"
          placeholder="用于找回账号与通知"
          autocomplete="email"
          :prefix-icon="Message"
        />
      </el-form-item>

      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          size="large"
          show-password
          placeholder="至少 6 位"
          autocomplete="new-password"
          :prefix-icon="Lock"
        />
      </el-form-item>

      <el-form-item label="确认密码" prop="confirm">
        <el-input
          v-model="form.confirm"
          type="password"
          size="large"
          show-password
          placeholder="再次输入密码"
          autocomplete="new-password"
          :prefix-icon="Lock"
          @keyup.enter="submit"
        />
      </el-form-item>

      <el-button type="primary" size="large" class="form__submit" :loading="loading" @click="submit">
        注册并登录
      </el-button>
    </el-form>

    <p class="switch text-secondary">
      已有账号？
      <router-link to="/login" class="switch__link">直接登录</router-link>
    </p>
  </AuthPanel>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Message } from '@element-plus/icons-vue'
import AuthPanel from '@/components/AuthPanel.vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const user = useUserStore()

const formRef = ref(null)
const loading = ref(false)
const form = reactive({ username: '', email: '', password: '', confirm: '' })

function validateConfirm(_rule, value, callback) {
  if (!value) callback(new Error('请再次输入密码'))
  else if (value !== form.password) callback(new Error('两次输入的密码不一致'))
  else callback()
}

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 32, message: '长度需在 3-32 位之间', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: '仅支持字母、数字与下划线', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 64, message: '密码长度需在 6-64 位之间', trigger: 'blur' }
  ],
  confirm: [{ required: true, validator: validateConfirm, trigger: 'blur' }]
}

async function submit() {
  if (loading.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await user.register({
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password
    })
    // 注册成功直接登录，省去再输一次密码
    await user.login({ username: form.username.trim(), password: form.password })
    ElMessage.success('注册成功，欢迎加入 NovaDrive')
    router.replace('/')
  } catch {
    // 错误提示由请求拦截器统一展示
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
.form__submit {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  margin-top: 4px;
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
