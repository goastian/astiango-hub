<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { ElTree, ElInput, ElScrollbar } from 'element-plus';
import type { FilterNodeMethodFunction } from 'element-plus/es/components/tree/src/tree.type';
import { debounce } from 'lodash';
import { translate } from '@/utils';

// i18n
const t = translate;

// props and emits
const props = defineProps<{
  activeNavItemId?: string;
  treeItems?: AutoProbeNavItem[];
  defaultExpandedKeys?: string[];
}>();

const emit = defineEmits<{
  (e: 'node-select', item: AutoProbeNavItem): void;
}>();

const treeRef = ref<InstanceType<typeof ElTree>>();
const searchKeyword = ref('');
const showSearch = ref(false);
const defaultExpandedKeys = ref<string[]>(props.defaultExpandedKeys || []);

// Context menu functionality
const activeContextMenuNavItem = ref<AutoProbeNavItem>();
const contextMenuVisibleMap = ref<Record<string, boolean>>({});
const isContextMenuVisible = (id: string) => {
  if (!contextMenuItems.value?.length) return false;
  if (!activeContextMenuNavItem.value) return false;
  if (activeContextMenuNavItem.value?.id !== id) return false;
  return contextMenuVisibleMap.value[id] || false;
};
const onActionsClick = (item: AutoProbeNavItem) => {
  activeContextMenuNavItem.value = item;
  contextMenuVisibleMap.value[item.id] = true;
};
const onContextMenuHide = (id: string) => {
  activeContextMenuNavItem.value = undefined;
  contextMenuVisibleMap.value[id] = false;
};
const contextMenuItems = computed(() => {
  if (!activeContextMenuNavItem.value) return [];
  const { id, type } = activeContextMenuNavItem.value;
  if (!contextMenuVisibleMap.value[id]) return [];
  switch (type) {
    case 'field':
      return [
        {
          title: t('common.actions.view'),
          icon: ['fa', 'eye'],
          action: () =>
            selectNode(activeContextMenuNavItem.value as AutoProbeNavItem),
        },
      ];
    case 'list':
      return [
        {
          title: t('common.actions.view'),
          icon: ['fa', 'eye'],
          action: () =>
            selectNode(activeContextMenuNavItem.value as AutoProbeNavItem),
        },
      ];
    default:
      return [];
  }
});

// Search functionality
const onSearchFilter: FilterNodeMethodFunction = (value, data) => {
  if (!value) return true;
  return data.label.toLowerCase().includes(value.toLowerCase());
};

const debouncedFilter = debounce(() => {
  treeRef.value?.filter(searchKeyword.value);
}, 300);

watch(searchKeyword, debouncedFilter);

// Node selection and expansion tracking
const onNodeClick = async (data: AutoProbeNavItem) => {
  await selectNode(data);
};

const selectNode = async (data: AutoProbeNavItem) => {
  const { id } = data;
  emit('node-select', data);

  // Highlight current node
  setTimeout(() => {
    treeRef.value?.setCurrentKey(id);
  }, 0);
};

const onNodeExpand = (data: AutoProbeNavItem) => {
  defaultExpandedKeys.value.push(data.id);
};

const onNodeCollapse = (data: AutoProbeNavItem) => {
  const idx = defaultExpandedKeys.value.findIndex(id => id === data.id);
  defaultExpandedKeys.value.splice(idx, 1);
};

const getNode = (id: string) => {
  return treeRef.value?.getNode(id)?.data as AutoProbeNavItem | undefined;
};

const onContextMenuClick = (event: MouseEvent, data: AutoProbeNavItem) => {
  event.stopPropagation();
  activeContextMenuNavItem.value = data;
  contextMenuVisibleMap.value[data.id] = true;
};

const onSearchClick = () => {
  showSearch.value = !showSearch.value;
};

// Update current selection when prop changes
watch(
  () => props.activeNavItemId,
  newId => {
    if (newId) {
      setTimeout(() => {
        treeRef.value?.setCurrentKey(newId);
      }, 0);
    }
  }
);

// Sidebar resizing
const widthKey = ref('autoprobe.sidebar.width');
const sidebarRef = ref<HTMLElement | null>(null);

defineExpose({
  getNode,
});

defineOptions({ name: 'ClAutoProbePagePatternsSidebar' });
</script>

<template>
  <div ref="sidebarRef" class="sidebar">
    <cl-resize-handle :target-ref="sidebarRef" :size-key="widthKey" />
    <div class="sidebar-actions">
      <cl-icon
        :class="showSearch ? 'selected' : ''"
        :icon="['fa', 'search']"
        @click="onSearchClick"
      />
    </div>
    <div v-if="showSearch" class="sidebar-search">
      <el-input
        v-model="searchKeyword"
        :placeholder="t('common.actions.search')"
        clearable
        @clear="
          () => {
            searchKeyword = '';
            showSearch = false;
          }
        "
      />
    </div>
    <el-scrollbar>
      <el-tree
        ref="treeRef"
        node-key="id"
        :data="treeItems"
        :filter-node-method="onSearchFilter"
        :expand-on-click-node="false"
        :default-expanded-keys="defaultExpandedKeys"
        highlight-current
        @node-click="onNodeClick"
        @node-contextmenu="onContextMenuClick"
        @node-expand="onNodeExpand"
        @node-collapse="onNodeCollapse"
      >
        <template #default="{ data }">
          <cl-context-menu
            :visible="isContextMenuVisible(data.id)"
            :style="{ flex: 1, paddingRight: '5px' }"
          >
            <template #reference>
              <div class="node-wrapper" :title="data.label">
                <span class="icon-wrapper">
                  <cl-icon
                    v-if="data.loading"
                    :icon="['fa', 'spinner']"
                    spinning
                  />
                  <cl-icon v-else :icon="data.icon || ['fa', 'folder']" />
                </span>
                <span class="label">
                  {{ data.label }}
                </span>
              </div>
              <div class="actions">
                <cl-icon
                  class="more"
                  :icon="['fa', 'ellipsis']"
                  @click.stop="onActionsClick(data)"
                />
              </div>
            </template>
            <cl-context-menu-list
              v-if="isContextMenuVisible(data.id)"
              :items="contextMenuItems"
              @hide="onContextMenuHide(data.id)"
            />
          </cl-context-menu>
        </template>
      </el-tree>
    </el-scrollbar>
  </div>
</template>

<style scoped>
.sidebar {
  flex: 0 0 240px;
  height: 100%;
  overflow: hidden;
  border-right: 1px solid var(--el-border-color);
  display: flex;
  flex-direction: column;
  position: relative;

  .sidebar-actions {
    height: 41px;
    flex: 0 0 41px;
    padding: 5px;
    display: flex;
    align-items: center;
    gap: 5px;
    color: var(--cl-primary-color);
    border-bottom: 1px solid var(--el-border-color);

    & > * {
      display: flex;
      align-items: center;
    }

    &:deep(.icon) {
      cursor: pointer;
      padding: 6px;
      font-size: 14px;
      width: 14px;
      height: 14px;
      border-radius: 50%;
    }

    &:deep(.icon.selected),
    &:deep(.icon:hover) {
      background-color: var(--cl-primary-plain-color);
    }
  }

  .sidebar-search {
    height: 38px;
    flex: 0 0 38px;
    border-bottom: 1px solid var(--el-border-color);

    &:deep(.el-input .el-input__wrapper) {
      box-shadow: none;
      border: none;
    }
  }

  .el-tree {
    min-width: fit-content;

    &:deep(.el-tree-node__content:hover .actions .icon) {
      display: flex !important;
    }

    &:deep(.el-tree-node__content) {
      width: 100%;
      position: relative;

      .node-wrapper {
        display: flex;
        align-items: center;
        position: relative;
        width: 100%;

        .icon-wrapper {
          width: 20px;
          display: flex;
        }

        .label {
          flex: 0 0 auto;
        }
      }

      .actions {
        display: flex;
        gap: 5px;
        position: absolute;
        top: 0;
        right: 5px;
        height: 100%;
        align-items: center;

        &:deep(.icon.more) {
          display: none;
        }
      }
    }
  }
}
</style>
