import SparkMD5 from 'spark-md5'

// computeFileMD5 分块读取文件计算 MD5（不一次性载入内存）
// onProgress 回调入参为 0~1 的进度
export function computeFileMD5(file, chunkSize = 5 * 1024 * 1024, onProgress) {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    let offset = 0
    const total = file.size || 1

    reader.onload = (e) => {
      spark.append(e.target.result)
      offset += e.target.result.byteLength
      if (onProgress) onProgress(Math.min(offset / total, 1))
      if (offset < file.size) readNext()
      else resolve(spark.end())
    }
    reader.onerror = () => reject(reader.error || new Error('读取文件失败'))

    function readNext() {
      const blob = file.slice(offset, offset + chunkSize)
      reader.readAsArrayBuffer(blob)
    }

    if (file.size === 0) {
      resolve(spark.end())
      return
    }
    readNext()
  })
}
