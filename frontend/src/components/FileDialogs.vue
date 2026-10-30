<template>
  <template v-if="actions">
    <PreviewModal
      :model-value="state.previewOpen"
      :file="state.previewFile"
      :siblings="state.previewSiblings"
      @update:model-value="state.previewOpen = $event"
      @change="state.previewFile = $event"
      @download="actions.download"
    />

    <ShareDialog
      :model-value="state.shareOpen"
      :file="state.shareFile"
      @update:model-value="state.shareOpen = $event"
    />

    <MoveDialog
      :model-value="state.moveOpen"
      :mode="state.moveMode"
      :item="state.moveItem"
      @update:model-value="state.moveOpen = $event"
      @confirm="onMoveConfirm"
    />

    <el-dialog
      :model-value="state.renameOpen"
      title="重命名"
      width="420px"
      append-to-body
      @update:model-value="state.renameOpen = $event"
      @opened="focusRename"
    >
      <el-input
        ref="renameInputRef"
        v-model="state.renameValue"
        maxlength="255"
        show-word-limit
        placeholder="请输入新名称"
        @keyup.enter="onRenameConfirm"
      />
      <template #footer>
        <el-button @click="state.renameOpen = false">取消</el-button>
        <el-button type="primary" :loading="renaming" @click="onRenameConfirm">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      :model-value="state.detailsOpen"
      title="文件详情"
      width="420px"
      append-to-body
      @update:model-value="state.detailsOpen = $event"
    >
      <div v-if="state.detailsItem" class="detail">
        <div class="detail__head">
          <FileAvatar :item="state.detailsItem" size="lg" />
          <p class="detail__name">{{ state.detailsItem.name }}</p>
        </div>
        <dl class="detail__list">
          <div class="detail__row">
            <dt>类型</dt>
            <dd>{{ typeLabel(state.detailsItem) }}</dd>
          </div>
          <div class="detail__row">
            <dt>大小</dt>
            <dd class="num">{{ isDir(state.detailsItem) ? '—' : formatBytes(state.detailsItem.size) }}</dd>
          </div>
          <div class="detail__row">
            <dt>修改时间</dt>
            <dd class="num">{{ formatTime(state.detailsItem.updated_at) }}</dd>
          </div>
          <div class="detail__row">
            <dt>创建时间</dt>
            <dd class="num">{{ formatTime(state.detailsItem.created_at) }}</dd>
          </div>
          <div class="detail__row">
            <dt>收藏</dt>
            <dd>{{ Number(state.detailsItem.is_favorite) === 1 ? '已收藏' : '未收藏' }}</dd>
          </div>
        </dl>
      </div>
      <template #footer>
        <el-button @click="state.detailsOpen = false">关闭</el-button>
      </template>
    </el-dialog>
  </template>
</template>

<script setup>
import { computed, ref } from 'vue'
import PreviewModal from './PreviewModal.vue'
import ShareDialog from './ShareDialog.vue'
import MoveDialog from './MoveDialog.vue'
import FileAvatar from './FileAvatar.vue'
import { formatBytes, formatTime } from '@/utils/format'
import { isDir, typeLabel } from '@/utils/fileType'

const props = defineProps({
  // useFileActions() 的返回值
  actions: { type: Object, default: null }
})

const renaming = ref(false)
const renameInputRef = ref(null)

const state = computed(() => props.actions?.state || {})

function focusRename() {
  renameInputRef.value?.focus()
}

async function onMoveConfirm(payload) {
  await props.actions.confirmMove(payload)
}

async function onRenameConfirm() {
  if (!state.value.renameValue?.trim()) return
  renaming.value = true
  try {
    await props.actions.confirmRename()
  } finally {
    renaming.value = false
  }
}
</script>
<style scoped>
.detail__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--nd-border);
}
.detail__name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  word-break: break-all;
}
.detail__list {
  margin: 12px 0 0;
}
.detail__row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 8px 0;
  font-size: 13px;
}
.detail__row dt {
  width: 68px;
  flex: none;
  color: var(--nd-text-secondary);
}
.detail__row dd {
  flex: 1;
  min-width: 0;
  margin: 0;
  color: var(--nd-text);
  word-break: break-all;
}
</style>
