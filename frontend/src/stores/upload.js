import { defineStore } from 'pinia'
import pLimit from 'p-limit'
import * as uploadApi from '@/api/upload'
import { computeFileMD5 } from '@/utils/md5'

// 上传参数：小文件直传，大文件分片；阈值与分片大小与后端配置保持一致
const CHUNK_SIZE = 5 * 1024 * 1024
const THRESHOLD = 10 * 1024 * 1024
const CONCURRENCY = 3
const MAX_TASK_PARALLEL = 2
const MAX_RETRY = 3
const BATCH_MAX_FILES = 50
const BATCH_MAX_BYTES = 64 * 1024 * 1024

let taskSeq = 0

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

const isActive = (task) => ['hashing', 'uploading', 'merging'].includes(task.status)

export const useUploadStore = defineStore('upload', {
  state: () => ({
    tasks: [],
    panelOpen: false,
    concurrency: CONCURRENCY,
    chunkSize: CHUNK_SIZE,
    threshold: THRESHOLD,
    // 每次有任务完成自增，供列表页监听后刷新
    revision: 0
  }),
  getters: {
    activeCount: (s) => s.tasks.filter(isActive).length,
    pendingCount: (s) => s.tasks.filter((t) => t.status === 'waiting').length,
    doneCount: (s) => s.tasks.filter((t) => t.status === 'done').length,
    failedCount: (s) => s.tasks.filter((t) => t.status === 'error').length,
    hasTasks: (s) => s.tasks.length > 0,
    overallProgress: (s) => {
      if (!s.tasks.length) return 0
      const total = s.tasks.reduce((sum, t) => sum + (t.size || 1), 0)
      const loaded = s.tasks.reduce((sum, t) => sum + (t.size || 1) * (t.progress || 0), 0)
      return total > 0 ? Math.min(100, Math.round((loaded / total) * 1000) / 10) : 0
    }
  },
  actions: {
    // addFiles 接收 [{ file, relativePath }]，自动区分普通文件与文件夹
    addFiles(items, parentId = 0) {
      const plain = []
      const folder = []
      for (const item of items || []) {
        if (!item?.file) continue
        if ((item.relativePath || '').includes('/')) folder.push(item)
        else plain.push(item)
      }
      for (const item of plain) {
        this.tasks.push(this.createFileTask(item, parentId))
      }
      if (folder.length) {
        this.tasks.push(this.createBatchTask(folder, parentId))
      }
      if (plain.length || folder.length) {
        this.panelOpen = true
        this.pump()
      }
    },
    createFileTask(item, parentId) {
      taskSeq += 1
      return {
        id: `t${Date.now()}-${taskSeq}`,
        kind: 'file',
        file: item.file,
        name: item.file.name,
        size: item.file.size,
        relativePath: item.relativePath || item.file.name,
        parentId,
        status: 'waiting',
        progress: 0,
        uploadedBytes: 0,
        speed: 0,
        doneChunks: 0,
        totalChunks: 0,
        uploadedSet: [],
        hash: '',
        uploadId: '',
        error: '',
        instant: false,
        canceled: false,
        result: null,
        createdAt: Date.now()
      }
    },
    createBatchTask(items, parentId) {
      taskSeq += 1
      const root = (items[0].relativePath || '').split('/')[0]
      const size = items.reduce((sum, it) => sum + it.file.size, 0)
      return {
        id: `t${Date.now()}-${taskSeq}`,
        kind: 'batch',
        items,
        name: root || '文件夹',
        size,
        parentId,
        status: 'waiting',
        progress: 0,
        uploadedBytes: 0,
        speed: 0,
        doneChunks: 0,
        totalChunks: 0,
        error: '',
        canceled: false,
        failedItems: [],
        totalFiles: items.length,
        doneFiles: 0,
        createdAt: Date.now()
      }
    },
    // pump 控制同时进行的任务数，避免多个大文件同时占满带宽
    pump() {
      const running = this.tasks.filter(isActive).length
      let slots = Math.max(0, MAX_TASK_PARALLEL - running)
      for (const task of this.tasks) {
        if (slots <= 0) break
        if (task.status !== 'waiting') continue
        slots -= 1
        this.run(task)
      }
    },
    async run(task) {
      try {
        if (task.kind === 'batch') await this.runBatch(task)
        else await this.runFile(task)
      } catch (err) {
        if (task.canceled) {
          task.status = 'canceled'
        } else if (task.status !== 'paused') {
          task.status = 'error'
          task.error = err?.message || '上传失败'
        }
      } finally {
        if (task.status === 'done') {
          task.progress = 1
          this.revision += 1
        }
        this.pump()
      }
    },
    async runFile(task) {
      task.status = 'hashing'
      task.error = ''
      if (task.size <= this.threshold) {
        await this.uploadSimple(task)
        task.status = 'done'
        return
      }
      task.hash = await computeFileMD5(task.file, this.chunkSize, (p) => {
        task.progress = p * 0.1
      })
      if (task.status === 'paused' || task.canceled) return

      const total = Math.max(1, Math.ceil(task.size / this.chunkSize))
      task.totalChunks = total
      task.status = 'uploading'

      const init = await uploadApi.init({
        parent_id: task.parentId,
        file_name: task.file.name,
        file_size: task.size,
        hash_md5: task.hash,
        chunk_size: this.chunkSize,
        total_chunks: total
      })
      if (task.canceled) return
      if (init.instant) {
        // 秒传：服务端已复用物理文件
        task.instant = true
        task.uploadId = ''
        task.result = init.file
        task.uploadedBytes = task.size
        task.doneChunks = total
        task.status = 'done'
        return
      }

      task.uploadId = init.upload_id
      task.uploadedSet = Array.isArray(init.uploaded_chunks) ? [...init.uploaded_chunks] : []
      task.doneChunks = task.uploadedSet.length
      if (task.status === 'paused') return

      await this.dispatchChunks(task)
      if (task.status === 'paused' || task.canceled) return

      task.status = 'merging'
      task.progress = Math.max(task.progress, 0.96)
      const done = await uploadApi.complete(task.uploadId, task.hash)
      task.result = done?.file || null
      task.uploadedBytes = task.size
      task.status = 'done'
    },
    async uploadSimple(task) {
      task.status = 'uploading'
      const res = await uploadApi.simple(task.file, task.parentId, task.relativePath, (p) => {
        task.progress = p
      })
      task.result = res
      task.uploadedBytes = task.size
    },
    async dispatchChunks(task) {
      const total = task.totalChunks
      const uploaded = new Set(task.uploadedSet || [])
      const queue = []
      for (let i = 0; i < total; i += 1) {
        if (!uploaded.has(i)) queue.push(i)
      }
      if (!queue.length) {
        task.progress = Math.max(task.progress, 0.95)
        return
      }

      const baseBytes = task.doneChunks * this.chunkSize
      const startedAt = Date.now()
      let cursor = 0
      const limit = pLimit(Math.max(1, this.concurrency))

      const worker = async () => {
        for (;;) {
          if (task.canceled || task.status === 'paused') return
          const index = queue[cursor]
          cursor += 1
          if (index === undefined) return

          const start = index * this.chunkSize
          const blob = task.file.slice(start, Math.min(start + this.chunkSize, task.size))
          await this.uploadChunkWithRetry(task, index, blob)

          task.doneChunks += 1
          task.uploadedSet.push(index)
          task.uploadedBytes = Math.min(task.size, baseBytes + (task.doneChunks - uploaded.size) * this.chunkSize)
          task.progress = 0.1 + (task.doneChunks / total) * 0.85
          const elapsed = (Date.now() - startedAt) / 1000
          if (elapsed > 0.4) {
            const sent = task.uploadedBytes - baseBytes
            task.speed = sent > 0 ? sent / elapsed : 0
          }
        }
      }

      const workers = Array.from({ length: Math.min(Math.max(1, this.concurrency), queue.length) }, worker)
      await Promise.all(workers)
    },
    // uploadChunkWithRetry 单个分片失败按指数退避重试 3 次
    async uploadChunkWithRetry(task, index, blob) {
      for (let attempt = 0; ; attempt += 1) {
        if (task.canceled || task.status === 'paused') return
        try {
          await uploadApi.chunk(task.uploadId, index, blob)
          return
        } catch (err) {
          if (attempt >= MAX_RETRY) throw err
          await sleep(400 * 2 ** attempt)
        }
      }
    },
    async runBatch(task) {
      task.status = 'uploading'
      const groups = this.groupBatch(task.items)
      const totalBytes = task.size || 1
      let doneBytes = 0
      const failed = []

      for (const group of groups) {
        if (task.canceled) {
          task.status = 'canceled'
          return
        }
        if (task.status === 'paused') return
        const bytes = group.reduce((sum, it) => sum + it.file.size, 0)
        const res = await uploadApi.batch(group, task.parentId, (p) => {
          task.progress = Math.min(0.99, (doneBytes + bytes * p) / totalBytes)
        })
        doneBytes += bytes
        task.uploadedBytes = doneBytes
        task.progress = Math.min(1, doneBytes / totalBytes)
        task.doneFiles += res?.success_count || 0
        if (res?.failed?.length) failed.push(...res.failed)
      }

      task.failedItems = failed
      if (failed.length) {
        task.error = `${failed.length} 个文件上传失败`
        task.status = failed.length === task.items.length ? 'error' : 'done'
      } else {
        task.status = 'done'
      }
    },
    // groupBatch 按文件数与总字节拆分批次，避免单次请求过大
    groupBatch(items) {
      const groups = []
      let current = []
      let bytes = 0
      for (const item of items) {
        const size = item.file.size
        if (current.length >= BATCH_MAX_FILES || (bytes + size > BATCH_MAX_BYTES && current.length > 0)) {
          groups.push(current)
          current = []
          bytes = 0
        }
        current.push(item)
        bytes += size
      }
      if (current.length) groups.push(current)
      return groups
    },
    pause(taskId) {
      const task = this.tasks.find((t) => t.id === taskId)
      if (!task) return
      if (['waiting', 'hashing', 'uploading', 'merging'].includes(task.status)) {
        task.status = 'paused'
      }
    },
    async resume(taskId) {
      const task = this.tasks.find((t) => t.id === taskId)
      if (!task || task.status !== 'paused') return
      task.status = 'waiting'
      if (task.kind === 'batch' || !task.uploadId) {
        this.pump()
        return
      }
      // 分片任务先向服务端同步已上传列表，实现断点续传
      try {
        const st = await uploadApi.status(task.uploadId)
        task.uploadedSet = Array.isArray(st.uploaded_chunks) ? [...st.uploaded_chunks] : []
        task.doneChunks = task.uploadedSet.length
        task.totalChunks = st.total_chunks || task.totalChunks
      } catch {
        task.uploadedSet = task.uploadedSet || []
      }
      this.run(task)
    },
    retry(taskId) {
      const task = this.tasks.find((t) => t.id === taskId)
      if (!task) return
      task.canceled = false
      task.error = ''
      task.progress = 0
      task.uploadedBytes = 0
      task.speed = 0
      task.doneChunks = 0
      task.uploadedSet = []
      task.instant = false
      task.failedItems = []
      task.doneFiles = 0
      task.status = 'waiting'
      this.pump()
    },
    async remove(taskId) {
      const index = this.tasks.findIndex((t) => t.id === taskId)
      if (index < 0) return
      const [task] = this.tasks.splice(index, 1)
      task.canceled = true
      if (task.uploadId && isActive(task)) {
        try {
          await uploadApi.cancel(task.uploadId)
        } catch {
          // 取消会话失败忽略：GC 会清理过期分片
        }
      }
    },
    clearFinished() {
      this.tasks = this.tasks.filter((t) => !['done', 'canceled'].includes(t.status))
      if (!this.tasks.length) this.panelOpen = false
    },
    cancelAll() {
      for (const task of [...this.tasks]) {
        if (isActive(task) || task.status === 'waiting' || task.status === 'paused') {
          this.remove(task.id)
        }
      }
    }
  }
})
