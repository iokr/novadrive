// 文件类型判定：预览组件分发与图标 / 配色

const IMAGE_EXT = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg', 'ico', 'avif']
const VIDEO_EXT = ['mp4', 'webm', 'mov', 'm4v', 'ogv', 'avi', 'mkv']
const AUDIO_EXT = ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a']
const TEXT_EXT = ['txt', 'md', 'json', 'log', 'xml', 'yml', 'yaml', 'csv', 'ini', 'conf', 'js', 'ts', 'css', 'html', 'go', 'py', 'java', 'sh', 'sql']
const PDF_EXT = ['pdf']

export function extOf(file) {
  if (!file) return ''
  if (file.ext) return String(file.ext).toLowerCase()
  const name = file.name || ''
  const idx = name.lastIndexOf('.')
  return idx > 0 ? name.slice(idx + 1).toLowerCase() : ''
}

export function isDir(file) {
  return Number(file?.is_dir) === 1
}

// previewKind 返回预览类型：pdf / video / image / audio / text / other
export function previewKind(file) {
  if (!file || isDir(file)) return 'other'
  const mime = String(file.mime_type || '').toLowerCase()
  const ext = extOf(file)
  if (mime === 'application/pdf' || PDF_EXT.includes(ext)) return 'pdf'
  if (mime.startsWith('video/') || VIDEO_EXT.includes(ext)) return 'video'
  if (mime.startsWith('image/') || IMAGE_EXT.includes(ext)) return 'image'
  if (mime.startsWith('audio/') || AUDIO_EXT.includes(ext)) return 'audio'
  if (mime.startsWith('text/') || mime.includes('json') || TEXT_EXT.includes(ext)) return 'text'
  return 'other'
}

export function canPreview(file) {
  return previewKind(file) !== 'other'
}

export function isImage(file) {
  return previewKind(file) === 'image'
}

// iconOf 返回已全局注册的 Element Plus 图标组件名
export function iconOf(file) {
  if (isDir(file)) return 'Folder'
  switch (previewKind(file)) {
    case 'image':
      return 'Picture'
    case 'video':
      return 'VideoCamera'
    case 'audio':
      return 'Headset'
    case 'pdf':
      return 'Document'
    case 'text':
      return 'Memo'
    default:
      return 'Files'
  }
}

// tintOf 图标主色（文件夹取黄、图片取青、视频取紫，便于视觉区分）
export function tintOf(file) {
  if (isDir(file)) return '#FFB020'
  switch (previewKind(file)) {
    case 'image':
      return '#22D3EE'
    case 'video':
      return '#A78BFA'
    case 'audio':
      return '#F472B6'
    case 'pdf':
      return '#F87171'
    case 'text':
      return '#34D399'
    default:
      return '#94A3B8'
  }
}

// typeLabel 列表「类型」列的文案：文件夹 / MP4 视频 / PDF 文档 …
export function typeLabel(file) {
  if (!file) return '文件'
  if (isDir(file)) return '文件夹'
  const ext = extOf(file).toUpperCase()
  switch (previewKind(file)) {
    case 'video':
      return `${ext || 'VIDEO'} 视频`
    case 'image':
      return `${ext || 'IMG'} 图片`
    case 'audio':
      return `${ext || 'AUDIO'} 音频`
    case 'pdf':
      return 'PDF 文档'
    case 'text':
      return `${ext || 'TXT'} 文本`
    default:
      return ext ? `${ext} 文件` : '文件'
  }
}

// 表格排序字段白名单（与后端一致）
export const SORT_FIELDS = [
  { label: '名称', value: 'name' },
  { label: '大小', value: 'size' },
  { label: '修改时间', value: 'updated_at' },
  { label: '类型', value: 'ext' }
]
