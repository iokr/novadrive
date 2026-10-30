<template>
  <div class="table-wrap">
    <el-table
      v-loading="loading"
      :data="items"
      style="width: 100%"
      :row-class-name="rowClass"
      :default-sort="{ prop: sort, order: order === 'asc' ? 'ascending' : 'descending' }"
      @sort-change="onSortChange"
      @row-click="(row) => $emit('open', row)"
      @row-contextmenu="onRowMenu"
    >
      <el-table-column v-if="selectable" width="48" align="center">
        <template #header>
          <el-checkbox
            :model-value="allSelected"
            :indeterminate="selected.length > 0 && !allSelected"
            @change="$emit('toggle-select-all')"
          />
        </template>
        <template #default="{ row }">
          <el-checkbox
            :model-value="selected.includes(row.id)"
            @change="$emit('toggle-select', row.id)"
            @click.stop
          />
        </template>
      </el-table-column>

      <el-table-column label="文件名" prop="name" sortable="custom" min-width="280">
        <template #default="{ row }">
          <div class="cell-name">
            <FileAvatar :item="row" size="md" />
            <span class="cell-name__text ellipsis" :title="row.name">{{ row.name }}</span>
            <el-icon v-if="Number(row.is_favorite) === 1" class="cell-name__star">
              <StarFilled />
            </el-icon>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="大小" prop="size" sortable="custom" width="112">
        <template #default="{ row }">
          <span class="cell-meta num">{{ isDir(row) ? '—' : formatBytes(row.size) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="类型" width="132">
        <template #default="{ row }">
          <span class="cell-meta">{{ typeLabel(row) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="修改时间" prop="updated_at" sortable="custom" width="168">
        <template #default="{ row }">
          <span class="cell-meta num">{{ formatTime(row.updated_at) }}</span>
        </template>
      </el-table-column>

      <el-table-column v-if="mode !== 'recycle'" width="132" align="right">
        <template #default="{ row }">
          <div class="row-actions" @click.stop @dblclick.stop>
            <button
              type="button"
              class="icon-btn"
              :title="Number(row.is_favorite) === 1 ? '取消收藏' : '收藏'"
              @click="$emit('toggle-favorite', row)"
            >
              <el-icon :class="{ 'is-star': Number(row.is_favorite) === 1 }">
                <StarFilled v-if="Number(row.is_favorite) === 1" />
                <Star v-else />
              </el-icon>
            </button>
            <button type="button" class="icon-btn" title="下载" @click="$emit('action', 'download', row)">
              <el-icon><Download /></el-icon>
            </button>
            <button
              type="button"
              class="icon-btn"
              title="更多"
              @click="openMenuAtButton($event, row)"
            >
              <el-icon><MoreFilled /></el-icon>
            </button>
          </div>
        </template>
      </el-table-column>

      <el-table-column v-else label="操作" width="170" align="right">
        <template #default="{ row }">
          <div class="row-actions is-static">
            <el-button size="small" text type="primary" @click.stop="$emit('action', 'restore', row)">
              还原
            </el-button>
            <el-button size="small" text type="danger" @click.stop="$emit('action', 'forceDelete', row)">
              彻底删除
            </el-button>
          </div>
        </template>
      </el-table-column>

      <template #empty>
        <div class="table-empty">
          <el-icon><Files /></el-icon>
          <p>{{ emptyText }}</p>
        </div>
      </template>
    </el-table>

    <FileActionMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :item="menu.item"
      :mode="mode"
      @action="(key, item) => $emit('action', key, item)"
      @close="closeMenu"
    />
  </div>
</template>

<script setup>
import { computed, reactive } from 'vue'
import FileActionMenu from './FileActionMenu.vue'
import FileAvatar from './FileAvatar.vue'
import { formatBytes, formatTime } from '@/utils/format'
import { isDir, typeLabel } from '@/utils/fileType'

const props = defineProps({
  items: { type: Array, default: () => [] },
  selected: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  mode: { type: String, default: 'default' },
  selectable: { type: Boolean, default: true },
  sort: { type: String, default: 'name' },
  order: { type: String, default: 'asc' },
  emptyText: { type: String, default: '暂无文件' }
})

const emit = defineEmits([
  'open',
  'action',
  'toggle-select',
  'toggle-select-all',
  'toggle-favorite',
  'sort-change'
])

const menu = reactive({ visible: false, x: 0, y: 0, item: null })

const allSelected = computed(
  () => props.items.length > 0 && props.selected.length === props.items.length
)

function rowClass({ row }) {
  return props.selected.includes(row.id) ? 'is-selected-row' : ''
}

function onSortChange({ prop, order }) {
  if (!prop || !order) {
    emit('sort-change', 'name', 'asc')
    return
  }
  emit('sort-change', prop, order === 'ascending' ? 'asc' : 'desc')
}

function openMenuAtButton(event, row) {
  const rect = event.currentTarget.getBoundingClientRect()
  menu.visible = true
  menu.x = rect.left
  menu.y = rect.bottom + 6
  menu.item = row
}

function onRowMenu(row, _column, event) {
  event.preventDefault()
  menu.visible = true
  menu.x = event.clientX
  menu.y = event.clientY
  menu.item = row
}

function closeMenu() {
  menu.visible = false
  menu.item = null
}
</script>

<style scoped>
.table-wrap {
  border-radius: 10px;
  border: 1px solid var(--nd-border);
  background: var(--nd-surface);
  overflow: hidden;
}

/* 行高 / 表头 / 分隔线：贴近网盘列表的疏松排版 */
:deep(.el-table) {
  font-size: 13.5px;
  color: var(--nd-text);
}
:deep(.el-table th.el-table__cell) {
  height: 42px;
  padding: 0;
  background: var(--nd-surface-sunken);
  border-bottom: 1px solid var(--nd-border);
}
:deep(.el-table td.el-table__cell) {
  padding: 11px 0;
  border-bottom: 1px solid var(--nd-border);
}
:deep(.el-table__row) {
  cursor: pointer;
}
:deep(.el-table__row:last-child td.el-table__cell) {
  border-bottom: none;
}
:deep(.el-table__empty-block) {
  min-height: 240px;
}

.cell-name {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.cell-name__text {
  font-size: 13.5px;
  color: var(--nd-text);
}
.cell-name__star {
  color: #f7ba1e;
  font-size: 12px;
  flex: none;
}
.cell-meta {
  color: var(--nd-text-secondary);
  font-size: 12.5px;
}

.row-actions {
  display: flex;
  gap: 2px;
  justify-content: flex-end;
  opacity: 0;
  transition: opacity 160ms ease;
}
.row-actions.is-static {
  opacity: 1;
}
/* 行内操作按钮只在真正支持 hover 的设备上变色，避免触摸端点按时闪一下 */
@media (hover: hover) and (pointer: fine) {
  :deep(.el-table__row:hover) .row-actions {
    opacity: 1;
  }
  .icon-btn:hover {
    background: var(--nd-primary-soft);
    color: var(--nd-primary);
  }
}
:deep(.is-selected-row) .row-actions {
  opacity: 1;
}
.icon-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: var(--nd-text-secondary);
  display: grid;
  place-items: center;
  cursor: pointer;
  font-size: 14px;
  transition: background 160ms ease, color 160ms ease;
}
.is-star {
  color: #f7ba1e;
}

:deep(.is-selected-row) {
  background: var(--nd-primary-soft) !important;
}

.table-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--nd-text-muted);
  font-size: 13px;
}
.table-empty .el-icon {
  font-size: 34px;
  opacity: 0.6;
}

@media (max-width: 767px) {
  .row-actions {
    opacity: 1;
  }
}
</style>
