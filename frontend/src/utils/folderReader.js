// 文件夹上传：拖拽目录递归读取 / <input webkitdirectory> 结果归一
// 统一产出 [{ file, relativePath }]，relativePath 形如 "photos/2024/a.jpg"

// filesFromInput 处理 <input type="file" webkitdirectory multiple> 的 FileList
export function filesFromInput(fileList) {
  const result = []
  for (const file of Array.from(fileList || [])) {
    const relativePath = file.webkitRelativePath || file.relativePath || file.name
    result.push({ file, relativePath: normalize(relativePath) })
  }
  return result
}

// filesFromDataTransfer 处理拖拽：目录通过 webkitGetAsEntry 递归展开
export async function filesFromDataTransfer(dataTransfer) {
  const items = dataTransfer?.items
  if (!items || !items.length || typeof items[0].webkitGetAsEntry !== 'function') {
    // 退化：无法识别目录，按普通文件处理
    return Array.from(dataTransfer?.files || []).map((file) => ({ file, relativePath: file.name }))
  }
  const entries = []
  for (const item of Array.from(items)) {
    if (item.kind !== 'file') continue
    const entry = item.webkitGetAsEntry()
    if (entry) entries.push(entry)
  }
  const result = []
  for (const entry of entries) {
    if (entry.isDirectory) await walkDirectory(entry, '', result)
    else {
      const file = await readFileEntry(entry)
      if (file) result.push({ file, relativePath: file.name })
    }
  }
  return result
}

// hasDirectory 判断拖拽内容是否包含目录（用于提示「上传文件夹」）
export function hasDirectory(dataTransfer) {
  const items = dataTransfer?.items
  if (!items || !items.length) return false
  for (const item of Array.from(items)) {
    const entry = typeof item.webkitGetAsEntry === 'function' ? item.webkitGetAsEntry() : null
    if (entry && entry.isDirectory) return true
  }
  return false
}

async function walkDirectory(dirEntry, prefix, out) {
  const base = prefix ? `${prefix}/${dirEntry.name}` : dirEntry.name
  const reader = dirEntry.createReader()
  const entries = await readAllEntries(reader)
  for (const entry of entries) {
    if (entry.isDirectory) {
      await walkDirectory(entry, base, out)
      continue
    }
    const file = await readFileEntry(entry)
    if (file) out.push({ file, relativePath: `${base}/${file.name}` })
  }
}

// readAllEntries 逐批读取（readEntries 单次最多返回 100 条）
function readAllEntries(reader) {
  return new Promise((resolve, reject) => {
    const all = []
    const next = () => {
      reader.readEntries((batch) => {
        if (!batch.length) {
          resolve(all)
          return
        }
        all.push(...batch)
        next()
      }, reject)
    }
    next()
  })
}

function readFileEntry(fileEntry) {
  return new Promise((resolve) => {
    fileEntry.file(
      (file) => resolve(file),
      () => resolve(null)
    )
  })
}

function normalize(p) {
  return String(p || '')
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
}
