<template>
  <div class="nd-page profile">
    <div class="nd-page__header">
      <div>
        <p class="nd-page__title">个人中心</p>
        <p class="nd-page__subtitle">管理账号资料、密码与存储空间</p>
      </div>
    </div>

    <div class="profile__grid">
      <aside class="user-card glass-card">
        <div class="user-card__avatar">
          <img v-if="user.info?.avatar" :src="user.info.avatar" alt="avatar" />
          <span v-else>{{ user.avatarText }}</span>
        </div>
        <p class="user-card__name">{{ user.displayName }}</p>
        <p class="user-card__account text-secondary">@{{ user.info?.username }}</p>
        <el-tag :type="user.isAdmin ? 'warning' : 'info'" size="small" effect="light" class="user-card__tag">
          {{ user.isAdmin ? '管理员' : '普通用户' }}
        </el-tag>

        <dl class="user-card__meta">
          <div>
            <dt>邮箱</dt>
            <dd class="ellipsis">{{ user.info?.email || '未填写' }}</dd>
          </div>
          <div>
            <dt>注册时间</dt>
            <dd>{{ formatDate(user.info?.created_at) }}</dd>
          </div>
          <div>
            <dt>最近登录</dt>
            <dd>{{ user.info?.last_login_at ? formatTime(user.info.last_login_at) : '—' }}</dd>
          </div>
        </dl>

        <div class="user-card__space">
          <div class="user-card__space-head">
            <span class="text-secondary">存储空间</span>
            <span class="num">{{ formatBytes(user.space.used) }} / {{ quotaText }}</span>
          </div>
          <div class="user-card__bar">
            <div class="user-card__bar-fill" :style="{ width: `${user.usedPercent}%` }" />
          </div>
          <p class="user-card__space-tip text-muted num">
            已使用 {{ user.usedPercent }}% · 共 {{ user.space.file_count }} 个文件
          </p>
        </div>

        <el-button class="user-card__logout" plain @click="logout">
          <el-icon><SwitchButton /></el-icon>
          <span>退出登录</span>
        </el-button>
      </aside>

      <section class="profile__main glass-card">
        <el-tabs v-model="tab">
          <el-tab-pane label="空间分析" name="storage">
            <div class="pane">
              <StorageChart :stats="stats" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="资料设置" name="profile">
            <el-form :model="profileForm" label-width="86px" class="pane form">
              <el-form-item label="用户名">
                <el-input :model-value="user.info?.username" disabled />
              </el-form-item>
              <el-form-item label="邮箱">
                <el-input :model-value="user.info?.email" disabled />
              </el-form-item>
              <el-form-item label="昵称">
                <el-input v-model="profileForm.nickname" maxlength="32" show-word-limit placeholder="展示在侧边栏的名字" />
              </el-form-item>
              <el-form-item label="头像地址">
                <el-input v-model="profileForm.avatar" placeholder="留空则使用昵称首字母" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="savingProfile" @click="saveProfile">保存修改</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="安全设置" name="security">
            <el-form
              ref="pwdFormRef"
              :model="pwdForm"
              :rules="pwdRules"
              label-width="86px"
              class="pane form"
            >
              <el-form-item label="当前密码" prop="old_password">
                <el-input v-model="pwdForm.old_password" type="password" show-password autocomplete="current-password" />
              </el-form-item>
              <el-form-item label="新密码" prop="new_password">
                <el-input v-model="pwdForm.new_password" type="password" show-password autocomplete="new-password" />
              </el-form-item>
              <el-form-item label="确认新密码" prop="confirm">
                <el-input v-model="pwdForm.confirm" type="password" show-password autocomplete="new-password" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="savingPwd" @click="savePassword">更新密码</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="关于" name="about">
            <div class="pane about">
              <p class="about__name">NovaDrive · 星云盘 <span class="text-muted">v2.1</span></p>
              <p class="text-secondary">
                基于 Go + Gin + MySQL + Vue 3 构建的私有云盘，支持分片上传、秒传、在线预览、链接分享与管理后台。
              </p>
              <ul class="about__list text-secondary">
                <li>· 分片阈值 10 MB，分片大小 5 MB，并发 3</li>
                <li>· 支持图片 / 视频 / 音频 / PDF / 文本在线预览</li>
                <li>· 前端构建产物通过 go:embed 内嵌，单二进制即可部署</li>
              </ul>
            </div>
          </el-tab-pane>
        </el-tabs>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import StorageChart from '@/components/StorageChart.vue'
import { useUserStore } from '@/stores/user'
import { useUploadStore } from '@/stores/upload'
import { formatBytes, formatDate, formatTime } from '@/utils/format'
import * as userApi from '@/api/user'

const router = useRouter()
const user = useUserStore()
const upload = useUploadStore()

const tab = ref('storage')
const stats = ref({})
const savingProfile = ref(false)
const savingPwd = ref(false)
const pwdFormRef = ref(null)

const profileForm = reactive({ nickname: '', avatar: '' })
const pwdForm = reactive({ old_password: '', new_password: '', confirm: '' })

const quotaText = computed(() =>
  Number(user.space.quota) > 0 ? formatBytes(user.space.quota) : '不限'
)

function validateConfirm(_rule, value, callback) {
  if (!value) callback(new Error('请再次输入新密码'))
  else if (value !== pwdForm.new_password) callback(new Error('两次输入的密码不一致'))
  else callback()
}

const pwdRules = {
  old_password: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 64, message: '密码长度需在 6-64 位之间', trigger: 'blur' }
  ],
  confirm: [{ required: true, validator: validateConfirm, trigger: 'blur' }]
}

async function loadStats() {
  try {
    stats.value = await userApi.overview()
  } catch {
    stats.value = {}
  }
}

async function saveProfile() {
  savingProfile.value = true
  try {
    await user.updateProfile({
      nickname: profileForm.nickname.trim(),
      avatar: profileForm.avatar.trim()
    })
    ElMessage.success('资料已更新')
  } finally {
    savingProfile.value = false
  }
}

async function savePassword() {
  const valid = await pwdFormRef.value?.validate().catch(() => false)
  if (!valid) return
  savingPwd.value = true
  try {
    await user.changePassword({
      old_password: pwdForm.old_password,
      new_password: pwdForm.new_password
    })
    ElMessage.success('密码已更新，下次登录请使用新密码')
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdForm.confirm = ''
    pwdFormRef.value?.clearValidate()
  } finally {
    savingPwd.value = false
  }
}

async function logout() {
  try {
    await ElMessageBox.confirm('确定退出当前账号？', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  upload.cancelAll()
  user.logout()
  router.replace('/login')
}

onMounted(async () => {
  await user.fetchProfile(true)
  profileForm.nickname = user.info?.nickname || ''
  profileForm.avatar = user.info?.avatar || ''
  await user.fetchSpace()
  await loadStats()
})
</script>

<style scoped>
.profile__grid {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}
.user-card {
  padding: 24px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  text-align: center;
}
.user-card__avatar {
  width: 74px;
  height: 74px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--nd-gradient);
  color: #fff;
  font-size: 28px;
  font-weight: 650;
  overflow: hidden;
  box-shadow: 0 12px 30px rgba(99, 102, 241, 0.35);
}
.user-card__avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.user-card__name {
  margin-top: 8px;
  font-size: 17px;
  font-weight: 650;
}
.user-card__account {
  font-size: 12.5px;
}
.user-card__tag {
  margin-top: 4px;
}
.user-card__meta {
  width: 100%;
  margin: 16px 0 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  text-align: left;
}
.user-card__meta div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 12.5px;
}
.user-card__meta dt {
  color: var(--nd-text-secondary);
  flex: none;
}
.user-card__meta dd {
  margin: 0;
  min-width: 0;
  color: var(--nd-text);
}
.user-card__space {
  width: 100%;
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--nd-border);
  text-align: left;
}
.user-card__space-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  gap: 8px;
}
.user-card__bar {
  height: 7px;
  margin: 8px 0 6px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.22);
  overflow: hidden;
}
.user-card__bar-fill {
  height: 100%;
  border-radius: 999px;
  background: var(--nd-gradient);
  transition: width 240ms ease;
}
.user-card__space-tip {
  font-size: 11px;
}
.user-card__logout {
  width: 100%;
  margin-top: 18px;
}
.profile__main {
  padding: 8px 20px 20px;
  min-height: 420px;
}
.pane {
  padding: 12px 4px 4px;
}
.form {
  max-width: 480px;
}
.about {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 13px;
  line-height: 1.8;
}
.about__name {
  font-size: 15px;
  font-weight: 650;
}
.about__list {
  margin: 4px 0 0;
  padding: 0;
  list-style: none;
  font-size: 12.5px;
}
@media (max-width: 1023px) {
  .profile__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
