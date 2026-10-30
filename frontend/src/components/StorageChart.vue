<template>
  <div class="storage-chart">
    <div class="storage-chart__ring-wrap">
      <div class="storage-chart__ring" :style="{ background: ringStyle }">
        <div class="storage-chart__hole">
          <p class="storage-chart__used num">{{ formatBytes(used) }}</p>
          <p class="storage-chart__quota text-secondary num">
            共 {{ quota > 0 ? formatBytes(quota) : '不限' }}
          </p>
          <p class="storage-chart__percent num">{{ usedPercent }}%</p>
        </div>
      </div>
    </div>

    <ul class="storage-chart__legend">
      <li v-for="seg in segments" :key="seg.type" class="legend-item">
        <span class="legend-item__dot" :style="{ background: seg.color }" />
        <span class="legend-item__label">{{ seg.label }}</span>
        <span class="legend-item__size num">{{ formatBytes(seg.size) }}</span>
        <span class="legend-item__count text-muted num">{{ seg.count }} 个</span>
      </li>
      <li v-if="!segments.length" class="legend-empty text-muted">暂无文件，快去上传吧</li>
    </ul>

    <div class="storage-chart__summary">
      <div class="summary-item">
        <p class="summary-item__value num">{{ totalCount }}</p>
        <p class="summary-item__label text-secondary">文件数</p>
      </div>
      <div class="summary-item">
        <p class="summary-item__value num">{{ dirCount }}</p>
        <p class="summary-item__label text-secondary">文件夹</p>
      </div>
      <div class="summary-item">
        <p class="summary-item__value num">{{ formatBytes(totalSize) }}</p>
        <p class="summary-item__label text-secondary">占用空间</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatBytes, percent } from '@/utils/format'

const props = defineProps({
  // FileStatsResp: { used, quota, total_size, total_count, dir_count, by_type }
  stats: { type: Object, default: () => ({}) }
})

// 分类配色与中文名，与后端 constants.StatsCategories 对齐
const TYPE_META = {
  image: { label: '图片', color: '#22D3EE' },
  video: { label: '视频', color: '#A78BFA' },
  pdf: { label: '文档', color: '#F87171' },
  audio: { label: '音频', color: '#F472B6' },
  text: { label: '文本', color: '#34D399' },
  other: { label: '其他', color: '#94A3B8' }
}

const used = computed(() => Number(props.stats.used) || 0)
const quota = computed(() => Number(props.stats.quota) || 0)
const totalSize = computed(() => Number(props.stats.total_size) || 0)
const totalCount = computed(() => Number(props.stats.total_count) || 0)
const dirCount = computed(() => Number(props.stats.dir_count) || 0)

const usedPercent = computed(() => {
  if (quota.value <= 0) return 0
  return Math.min(100, Math.round((used.value / quota.value) * 1000) / 10)
})

const segments = computed(() => {
  const list = Array.isArray(props.stats.by_type) ? props.stats.by_type : []
  return list
    .map((it) => {
      const meta = TYPE_META[it.type] || TYPE_META.other
      return {
        type: it.type,
        label: meta.label,
        color: meta.color,
        size: Number(it.size) || 0,
        count: Number(it.count) || 0
      }
    })
    .filter((it) => it.count > 0 || it.size > 0)
})

const ringStyle = computed(() => {
  const base = quota.value > 0 ? quota.value : used.value || 1
  let cursor = 0
  const stops = []
  for (const seg of segments.value) {
    const span = percent(seg.size, base)
    if (span <= 0) continue
    stops.push(`${seg.color} ${cursor}% ${cursor + span}%`)
    cursor += span
  }
  // 剩余配额留出轨道色，避免环形图被填满误导
  if (quota.value > 0) {
    const rest = Math.max(0, 100 - cursor)
    stops.push(`rgba(148, 163, 184, 0.16) ${cursor}% ${cursor + rest}%`)
  }
  if (!stops.length) stops.push('rgba(148, 163, 184, 0.18) 0% 100%')
  return { background: `conic-gradient(${stops.join(',')})` }
})
</script>

<style scoped>
.storage-chart {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.storage-chart__ring-wrap {
  display: flex;
  justify-content: center;
}
.storage-chart__ring {
  width: 190px;
  height: 190px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  position: relative;
  transition: background 320ms ease;
}
.storage-chart__ring::before {
  content: '';
  position: absolute;
  inset: 26px;
  border-radius: 50%;
  background: var(--el-bg-color-overlay, #131a2e);
  box-shadow: inset 0 0 24px rgba(3, 7, 18, 0.35);
}
html.light .storage-chart__ring::before {
  background: #fff;
  box-shadow: inset 0 0 20px rgba(30, 41, 59, 0.08);
}
.storage-chart__hole {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  text-align: center;
}
.storage-chart__used {
  font-size: 19px;
  font-weight: 650;
  letter-spacing: 0.2px;
}
.storage-chart__quota {
  font-size: 11.5px;
}
.storage-chart__percent {
  margin-top: 4px;
  font-size: 11.5px;
  color: var(--nd-cyan);
}
.storage-chart__legend {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}
.legend-item__dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  flex: none;
}
.legend-item__label {
  flex: 1;
  min-width: 0;
}
.legend-item__size {
  font-size: 12.5px;
  color: var(--nd-text);
}
.legend-item__count {
  width: 62px;
  text-align: right;
  font-size: 11.5px;
}
.legend-empty {
  font-size: 12.5px;
  text-align: center;
  padding: 8px 0;
}
.storage-chart__summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  padding-top: 16px;
  border-top: 1px solid var(--nd-border);
}
.summary-item {
  text-align: center;
}
.summary-item__value {
  font-size: 15px;
  font-weight: 600;
}
.summary-item__label {
  font-size: 11.5px;
  margin-top: 3px;
}
</style>
