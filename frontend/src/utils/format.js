// 体积 / 时间格式化工具

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

// formatBytes 人类可读体积，保留 1 位小数（B 取整）
export function formatBytes(bytes, digits = 1) {
  const n = Number(bytes) || 0
  if (n < 1024) return `${n} B`
  let value = n
  let unit = 0
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(digits)} ${UNITS[unit]}`
}

// formatSpeed 传输速度
export function formatSpeed(bytesPerSecond) {
  if (!bytesPerSecond) return '—'
  return `${formatBytes(bytesPerSecond, 1)}/s`
}

function pad(n) {
  return String(n).padStart(2, '0')
}

// formatDate 2025-01-01
export function formatDate(input) {
  const d = toDate(input)
  if (!d) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// formatTime 2025-01-01 10:00
export function formatTime(input) {
  const d = toDate(input)
  if (!d) return '—'
  return `${formatDate(d)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// formatDateTime 2025-01-01 10:00:00
export function formatDateTime(input) {
  const d = toDate(input)
  if (!d) return '—'
  return `${formatTime(d)}:${pad(d.getSeconds())}`
}

// relativeTime 刚刚 / 5 分钟前 / 3 天前 / 日期
export function relativeTime(input) {
  const d = toDate(input)
  if (!d) return '—'
  const diff = Date.now() - d.getTime()
  if (diff < 0) return formatTime(d)
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < minute) return '刚刚'
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`
  return formatDate(d)
}

// remainingTime 距过期的可读时长，用于分享有效期
export function remainingTime(input) {
  const d = toDate(input)
  if (!d) return '永久有效'
  const diff = d.getTime() - Date.now()
  if (diff <= 0) return '已过期'
  const day = 24 * 60 * 60 * 1000
  if (diff >= day) return `${Math.ceil(diff / day)} 天后过期`
  return `${Math.max(1, Math.ceil(diff / (60 * 60 * 1000)))} 小时后过期`
}

// percent 计算百分比（0-100，保留 1 位）
export function percent(part, total) {
  const p = Number(part) || 0
  const t = Number(total) || 0
  if (t <= 0) return 0
  return Math.min(100, Math.round((p / t) * 1000) / 10)
}

export function toDate(input) {
  if (!input) return null
  if (input instanceof Date) return input
  // 后端返回 RFC3339，浏览器可直接解析
  const d = new Date(input)
  return Number.isNaN(d.getTime()) ? null : d
}
