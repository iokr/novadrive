// 下载工具：Blob 落盘 / Content-Disposition 文件名解析

// triggerDownload 以 <a download> 方式保存 Blob
export function triggerDownload(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename || 'download'
  document.body.appendChild(a)
  a.click()
  a.remove()
  // 交给浏览器接管后释放
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// downloadUrl 直接跳转下载（走浏览器原生进度，适合大文件 / zip 流）
export function downloadUrl(url) {
  const a = document.createElement('a')
  a.href = url
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

// filenameFromDisposition 解析响应头中的文件名，优先 RFC 5987 编码
export function filenameFromDisposition(header, fallback = 'download') {
  if (!header) return fallback
  const star = /filename\*=UTF-8''([^;]+)/i.exec(header)
  if (star && star[1]) {
    try {
      return decodeURIComponent(star[1])
    } catch {
      return star[1]
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(header)
  return plain && plain[1] ? plain[1] : fallback
}
