<template>
  <div class="text-viewer">
    <div v-if="loading" class="text-viewer__state">
      <el-icon class="spin"><Loading /></el-icon>
      <p>正在读取文本…</p>
    </div>

    <div v-else-if="error" class="text-viewer__state">
      <el-icon class="text-viewer__state-icon"><Document /></el-icon>
      <p>{{ error }}</p>
      <el-button type="primary" size="small" @click="load">重新加载</el-button>
    </div>

    <template v-else>
      <div class="text-viewer__bar">
        <span class="num text-secondary">
          {{ lines.length }} 行 · {{ formatBytes(file?.size) }}
          <template v-if="truncated"> · 仅显示前 2 MB</template>
        </span>
        <div class="text-viewer__tools">
          <el-button text size="small" @click="wrap = !wrap">
            <el-icon><Sort /></el-icon>
            <span>{{ wrap ? '关闭换行' : '自动换行' }}</span>
          </el-button>
          <el-button text size="small" @click="copy">
            <el-icon><DocumentCopy /></el-icon>
            <span>复制</span>
          </el-button>
          <el-button text size="small" @click="download">
            <el-icon><Download /></el-icon>
            <span>下载</span>
          </el-button>
        </div>
      </div>

      <div class="text-viewer__body" :class="{ 'is-wrap': wrap }">
        <div class="text-viewer__code">
          <div v-for="(line, i) in lines" :key="i" class="text-viewer__line">
            <span class="text-viewer__no num">{{ i + 1 }}</span>
            <span class="text-viewer__text">{{ line || ' ' }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { rawUrl, downloadUrl } from '@/api/file'
import { downloadUrl as triggerUrl } from '@/utils/download'
import { formatBytes } from '@/utils/format'

// 文本预览上限 2MB，超出只渲染前 2MB（Range 请求），避免大文件把页面卡死
const MAX_BYTES = 2 * 1024 * 1024

const props = defineProps({
  file: { type: Object, default: null }
})

const loading = ref(true)
const error = ref('')
const content = ref('')
const wrap = ref(false)
const truncated = ref(false)

const lines = computed(() => (content.value ? content.value.replace(/\r\n/g, '\n').split('\n') : []))

// 优先按 UTF-8 严格解码，失败后回退 GBK（Windows 下的中文 txt 常见编码）
function decodeText(buffer) {
  const bytes = new Uint8Array(buffer)
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  } catch {
    try {
      return new TextDecoder('gbk').decode(bytes)
    } catch {
      return new TextDecoder('utf-8').decode(bytes)
    }
  }
}

async function load() {
  content.value = ''
  error.value = ''
  truncated.value = false
  if (!props.file) {
    loading.value = false
    return
  }
  loading.value = true
  const oversize = Number(props.file.size) > MAX_BYTES
  try {
    const res = await fetch(
      rawUrl(props.file.id),
      oversize ? { headers: { Range: `bytes=0-${MAX_BYTES - 1}` } } : undefined
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const buffer = await res.arrayBuffer()
    // 后端在文件缺失等场景下会返回 200 + JSON 业务错误，直接取 message 提示
    if ((res.headers.get('content-type') || '').includes('application/json')) {
      let message = '读取失败，文件可能已不存在'
      try {
        message = JSON.parse(decodeText(buffer)).message || message
      } catch {
        // 保留默认提示
      }
      throw new Error(message)
    }
    content.value = decodeText(buffer)
    truncated.value = oversize
  } catch (err) {
    error.value = err?.message ? `读取失败：${err.message}` : '读取失败'
  } finally {
    loading.value = false
  }
}

async function copy() {
  try {
    await navigator.clipboard.writeText(content.value)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动选择内容')
  }
}

function download() {
  if (props.file) triggerUrl(downloadUrl(props.file.id))
}

watch(
  () => props.file?.id,
  () => load(),
  { immediate: true }
)
</script>

<style scoped>
.text-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.text-viewer__state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--nd-text-secondary);
  font-size: 13px;
}
.text-viewer__state-icon {
  font-size: 46px;
  color: var(--nd-primary);
}
.text-viewer__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  background: var(--pv-chip, rgba(255, 255, 255, 0.03));
  border-bottom: 1px solid var(--pv-border, rgba(255, 255, 255, 0.08));
  font-size: 12px;
}
.text-viewer__bar :deep(.el-button) {
  color: var(--pv-text-secondary, rgba(203, 213, 225, 0.85));
}
.text-viewer__bar :deep(.el-button:hover) {
  color: var(--pv-text, #ffffff);
}
.text-viewer__tools {
  display: flex;
  align-items: center;
  gap: 2px;
}
.text-viewer__body {
  flex: 1;
  overflow: auto;
  padding: 10px 0;
  background: var(--pv-code-bg, rgba(3, 7, 18, 0.35));
}
.text-viewer__code {
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, 'Courier New', monospace;
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--pv-code-text, #cbd5e1);
  min-width: 100%;
  width: max-content;
}
.text-viewer__body.is-wrap .text-viewer__code {
  width: 100%;
}
.text-viewer__line {
  display: flex;
  gap: 12px;
  padding: 0 14px;
}
.text-viewer__line:hover {
  background: var(--pv-chip-hover, rgba(99, 102, 241, 0.12));
}
.text-viewer__no {
  flex: none;
  width: 46px;
  text-align: right;
  color: var(--pv-text-muted, #4b5563);
  user-select: none;
}
.text-viewer__text {
  white-space: pre;
  min-width: 0;
}
.text-viewer__body.is-wrap .text-viewer__text {
  white-space: pre-wrap;
  word-break: break-word;
}
.spin {
  font-size: 26px;
  color: var(--nd-cyan);
  animation: spin 1s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
