<template>
  <el-dialog
    :model-value="modelValue"
    :title="mode === 'copy' ? '复制到…' : '移动到…'"
    width="520px"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <p class="tip">
      {{ mode === 'copy' ? '复制' : '移动' }}「{{ item?.name || '' }}」，请选择目标目录：
    </p>

    <div class="tree-box">
      <el-tree
        ref="treeRef"
        lazy
        :load="loadNode"
        :props="treeProps"
        node-key="id"
        highlight-current
        :expand-on-click-node="false"
        @node-click="onSelect"
      />
    </div>

    <el-input
      v-if="mode === 'copy'"
      v-model="newName"
      class="name-input"
      placeholder="新名称（留空则自动生成副本名）"
      maxlength="255"
    />

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :disabled="targetId === null" @click="confirm">
        {{ mode === 'copy' ? '复制' : '移动' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import * as fileApi from '@/api/file'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  mode: { type: String, default: 'move' },
  item: { type: Object, default: null }
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const treeRef = ref(null)
const targetId = ref(null)
const newName = ref('')

const treeProps = {
  label: 'name',
  isLeaf: (data) => data.isLeaf === true
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      targetId.value = null
      newName.value = ''
    }
  }
)

async function loadNode(node, resolve) {
  if (node.level === 0) {
    resolve([{ id: 0, name: '我的文件（根目录）', isLeaf: false }])
    return
  }
  try {
    const data = await fileApi.list({ parent_id: node.data.id, sort: 'name', order: 'asc' })
    const dirs = (data?.items || [])
      .filter((it) => Number(it.is_dir) === 1)
      // 不能把目录移动到 / 复制到自身
      .filter((it) => it.id !== props.item?.id)
      .map((it) => ({ id: it.id, name: it.name, isLeaf: false }))
    resolve(dirs)
  } catch {
    resolve([])
  }
}

function onSelect(data) {
  targetId.value = data.id
}

function confirm() {
  if (targetId.value === null) return
  emit('confirm', { targetId: targetId.value, newName: newName.value.trim() })
  emit('update:modelValue', false)
}
</script>

<style scoped>
.tip {
  margin-bottom: 10px;
  font-size: 13px;
  color: var(--nd-text-secondary);
}
.tree-box {
  max-height: 300px;
  overflow: auto;
  padding: 8px;
  border-radius: 12px;
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
}
.name-input {
  margin-top: 12px;
}
</style>
