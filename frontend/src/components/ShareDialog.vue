<template>
  <el-dialog
    :model-value="modelValue"
    title="创建分享链接"
    width="520px"
    append-to-body
    class="share-dialog"
    @update:model-value="$emit('update:modelValue', $event)"
    @closed="reset"
  >
    <!-- 第一步：配置分享参数 -->
    <template v-if="!result">
      <div class="share-file">
        <el-icon class="share-file__icon" :style="{ color: tintOf(file) }">
          <component :is="iconOf(file)" />
        </el-icon>
        <div class="share-file__meta">
          <p class="ellipsis">{{ file?.name }}</p>
          <p class="text-secondary num">
            {{ isDir(file) ? '文件夹' : formatBytes(file?.size) }} · 分享后任何人都可访问链接
          </p>
        </div>
      </div>

      <el-form label-position="top" class="share-form">
        <el-form-item label="提取码">
          <div class="share-form__row">
            <el-input
              v-model="password"
              placeholder="留空表示不需要提取码"
              maxlength="16"
              clearable
              :disabled="autoPassword"
            />
            <el-checkbox v-model="autoPassword">自动生成</el-checkbox>
          </div>
        </el-form-item>

        <el-form-item label="有效期">
          <el-radio-group v-model="expireDays">
            <el-radio-button v-for="opt in expireOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="下载次数限制">
          <el-input-number v-model="maxDownloads" :min="0" :max="9999" :step="10" />
          <span class="share-form__hint text-muted">0 表示不限制</span>
        </el-form-item>
      </el-form>
    </template>

    <!-- 第二步：展示结果 -->
    <template v-else>
      <div class="share-result">
        <div class="share-result__qr">
          <QrcodeVue ref="qrRef" :value="resultLink" :size="150" level="M" />
          <button type="button" class="share-result__qr-save" @click="downloadQr">下载二维码</button>
        </div>
        <div class="share-result__info">
          <p class="share-result__label">分享链接</p>
          <div class="share-result__link">
            <el-input :model-value="resultLink" readonly />
            <el-button type="primary" plain @click="copy(resultLink)">复制</el-button>
          </div>
          <p v-if="result.has_password" class="share-result__row">
            <span class="text-secondary">提取码：</span>
            <b class="num">{{ result.password || password }}</b>
            <el-button link type="primary" size="small" @click="copy(result.password || password)">复制</el-button>
          </p>
          <p class="share-result__row text-secondary">
            {{ result.expire_at ? `${formatTime(result.expire_at)} 过期` : '永久有效' }}
          </p>
        </div>
      </div>
    </template>

    <template #footer>
      <template v-if="!result">
        <el-button @click="$emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">创建分享</el-button>
      </template>
      <template v-else>
        <el-button @click="$emit('update:modelValue', false)">关闭</el-button>
        <el-button type="primary" @click="copy(resultLink)">复制链接</el-button>
      </template>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import QrcodeVue from 'qrcode.vue'
import * as shareApi from '@/api/share'
import { formatBytes, formatTime } from '@/utils/format'
import { iconOf, isDir, tintOf } from '@/utils/fileType'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  file: { type: Object, default: null }
})

const emit = defineEmits(['update:modelValue', 'created'])

const expireOptions = [
  { label: '永久', value: 0 },
  { label: '1 天', value: 1 },
  { label: '7 天', value: 7 },
  { label: '30 天', value: 30 }
]

const password = ref('')
const autoPassword = ref(true)
const expireDays = ref(7)
const maxDownloads = ref(0)
const submitting = ref(false)
const result = ref(null)
const qrRef = ref(null)

const resultLink = computed(() => {
  const url = result.value?.url || ''
  if (!url) return ''
  if (/^https?:\/\//i.test(url)) return url
  return `${window.location.origin}${url.startsWith('/') ? '' : '/'}${url}`
})

function randomPassword() {
  const chars = 'abcdefghjkmnpqrstuvwxyz23456789'
  let out = ''
  for (let i = 0; i < 4; i += 1) out += chars[Math.floor(Math.random() * chars.length)]
  return out
}

async function submit() {
  if (!props.file) return
  submitting.value = true
  try {
    const data = await shareApi.create({
      file_id: props.file.id,
      password: autoPassword.value ? randomPassword() : password.value.trim(),
      expire_days: expireDays.value,
      max_downloads: maxDownloads.value
    })
    result.value = data
    emit('created', data)
  } finally {
    submitting.value = false
  }
}

async function copy(text) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择')
  }
}

// qrcode.vue 以 canvas 渲染，直接导出 PNG 方便保存到相册
function downloadQr() {
  const el = qrRef.value?.$el || qrRef.value
  if (!el || typeof el.toDataURL !== 'function') {
    ElMessage.warning('二维码尚未生成')
    return
  }
  const a = document.createElement('a')
  a.href = el.toDataURL('image/png')
  a.download = `novadrive-share-${result.value?.token || 'qr'}.png`
  document.body.appendChild(a)
  a.click()
  a.remove()
}

function reset() {
  result.value = null
  password.value = ''
  autoPassword.value = true
  expireDays.value = 7
  maxDownloads.value = 0
}
</script>

<style scoped>
.share-file {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
}
.share-file__icon {
  font-size: 26px;
  flex: none;
}
.share-file__meta {
  min-width: 0;
  font-size: 13px;
}
.share-file__meta .text-secondary {
  font-size: 12px;
  margin-top: 2px;
}
.share-form {
  margin-top: 16px;
}
.share-form__row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}
.share-form__hint {
  margin-left: 10px;
  font-size: 12px;
}
.share-result {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}
.share-result__qr {
  flex: none;
  padding: 12px;
  border-radius: 14px;
  background: #fff;
  line-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}
.share-result__qr-save {
  border: none;
  background: transparent;
  color: #4f52d6;
  font-size: 12px;
  cursor: pointer;
  line-height: 1.6;
  padding: 0;
}
.share-result__qr-save:hover {
  text-decoration: underline;
}
.share-result__info {
  flex: 1;
  min-width: 0;
}
.share-result__label {
  font-size: 12px;
  color: var(--nd-text-secondary);
  margin-bottom: 6px;
}
.share-result__link {
  display: flex;
  gap: 8px;
}
.share-result__row {
  margin-top: 8px;
  font-size: 12.5px;
  display: flex;
  align-items: center;
  gap: 4px;
}
@media (max-width: 767px) {
  .share-result {
    flex-direction: column;
    align-items: center;
  }
  .share-result__info {
    width: 100%;
  }
}
</style>
