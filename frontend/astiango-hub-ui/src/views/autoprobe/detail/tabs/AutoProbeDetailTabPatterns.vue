<script setup lang="ts">
import { computed, onBeforeMount, ref, watch } from 'vue';
import { useStore } from 'vuex';
import { cloneDeep, debounce } from 'lodash';
import { getIconByExtractType, getIconByItemType, translate } from '@/utils';
import { useAutoProbeDetail } from '@/views';

// i18n
const t = translate;

// store
const ns: ListStoreNamespace = 'autoprobe';
const store = useStore();
const { autoprobe: state } = store.state as RootStoreState;

const { activeId } = useAutoProbeDetail();

// form data
const form = computed<AutoProbeV2>(() => state.form);
const pagePattern = computed(() => form.value?.page_pattern as PagePatternV2);
const pageData = computed<PageData>(() => form.value?.page_data || {});
const pageNavItemId = 'page';

// results data and fields based on active item
const resultsDataFields = computed<AutoProbeResults>(() => {
  const rootDataFields: AutoProbeResults = {
    data: pageData.value,
    fields: computedTreeItems.value[0]?.children?.filter(
      item => item.type !== 'pagination'
    ),
  };

  if (!activeNavItem.value || !pageData.value) {
    return rootDataFields;
  }

  const item = activeNavItem.value;

  if (item.level === 0) {
    return rootDataFields;
  } else if (item.level === 1) {
    if (item.type === 'list') {
      // For V2 patterns, use pattern ID to get the data
      const pattern = item.rule as PatternV2;
      const patternId = pattern._id || pattern.name;
      return {
        data: pageData.value[patternId],
        fields: item.children,
      } as AutoProbeResults;
    }
    return {
      ...rootDataFields,
      activeField: item,
    };
  } else {
    let currentItem = item;
    while (currentItem.parent) {
      const parent = currentItem.parent;
      if (parent.level === 1 && parent.type === 'list') {
        // For V2 patterns, use pattern ID to get the data
        const parentPattern = parent.rule as PatternV2;
        const parentPatternId = parentPattern._id || parentPattern.name;
        return {
          data: pageData.value[parentPatternId],
          fields: parent.children,
          activeField: currentItem,
        } as AutoProbeResults;
      }
      currentItem = currentItem.parent;
    }
    return rootDataFields;
  }
});
const resultsData = computed(() => resultsDataFields.value.data);
const resultsFields = computed(() => resultsDataFields.value.fields);
const resultsActiveField = computed(() => resultsDataFields.value.activeField);

const normalizeItem = (item: AutoProbeNavItem) => {
  const label = item.label ?? `${item.name} (${item.children?.length || 0})`;
  let icon: Icon;
  if (item.type === 'field') {
    // For V2 patterns, extraction_type is directly on the pattern object
    const pattern = item.rule as PatternV2;
    icon = getIconByExtractType(pattern.extraction_type);
  } else {
    icon = getIconByItemType(item.type);
  }
  return {
    ...item,
    label,
    icon,
  } as AutoProbeNavItem;
};

// Helper function to recursively process V2 patterns
const processPatternV2 = (
  pattern: PatternV2,
  parent?: AutoProbeNavItem,
  level: number = 1
): AutoProbeNavItem => {
  const navItem: AutoProbeNavItem = {
    id: pattern._id || pattern.name,
    name: pattern.name,
    type: pattern.type as AutoProbeItemType,
    rule: pattern as any, // PatternV2 structure is compatible, just cast for type compatibility
    children: [],
    parent,
    level,
  };

  // Recursively process child patterns
  if (pattern.children && pattern.children.length > 0) {
    pattern.children.forEach((childPattern: PatternV2) => {
      navItem.children!.push(processPatternV2(childPattern, navItem, level + 1));
    });
  }

  return normalizeItem(navItem);
};

// items
const activeNavItem = ref<AutoProbeNavItem>();
const detailNavItem = computed<AutoProbeNavItem | undefined>(() => {
  if (!activeNavItem.value?.type) return;
  switch (activeNavItem.value.type) {
    case 'page_pattern':
    case 'list':
      return activeNavItem.value;
    case 'field':
    case 'pagination':
      return activeNavItem.value.parent;
  }
});
const computedTreeItems = computed<AutoProbeNavItem[]>(() => {
  if (!pagePattern.value) return [];

  const rootItem: AutoProbeNavItem = {
    id: pageNavItemId,
    name: pagePattern.value.name,
    type: 'page_pattern',
    children: [],
    level: 0,
  };

  // Process V2 pattern children
  if (pagePattern.value.children && pagePattern.value.children.length > 0) {
    pagePattern.value.children.forEach(pattern => {
      rootItem.children!.push(processPatternV2(pattern, rootItem, 1));
    });
  }

  return [normalizeItem(rootItem)];
});
const treeItems = ref<AutoProbeNavItem[]>([]);
watch(
  () => state.form,
  () => {
    treeItems.value = cloneDeep(computedTreeItems.value);
    if (!activeNavItem.value) {
      activeNavItem.value = treeItems.value[0];
    }
  },
  { immediate: true }
);

watch(activeId, () => {
  // Reset active item when the page changes
  activeNavItem.value = undefined;
});

// ref
const sidebarRef = ref();
const detailContainerRef = ref<HTMLElement | null>(null);

const onNodeSelect = (item: AutoProbeNavItem) => {
  activeNavItem.value = item;
};

const onItemRowClick = (id: string) => {
  const item = sidebarRef.value?.getNode(id);
  if (!item) return;
  activeNavItem.value = sidebarRef.value?.getNode(id);
};

// Handle results container resize
const onSizeChange = (size: number) => {
  if (!detailContainerRef.value) return;
  detailContainerRef.value.style.flex = `0 0 calc(100% - ${size}px)`;
  detailContainerRef.value.style.height = `calc(100% - ${size}px)`;
};

const getData = debounce(async () => {
  await Promise.all([
    store.dispatch(`${ns}/getPagePattern`, { id: activeId.value }),
    store.dispatch(`${ns}/getPagePatternData`, { id: activeId.value }),
  ]);
});
watch(activeId, getData);
onBeforeMount(getData);

defineOptions({ name: 'ClAutoProbeDetailTabPatterns' });
</script>

<template>
  <div class="autoprobe-detail-tab-patterns">
    <cl-auto-probe-page-patterns-sidebar
      ref="sidebarRef"
      :active-nav-item-id="activeNavItem?.id"
      :tree-items="treeItems"
      :default-expanded-keys="[pageNavItemId]"
      @node-select="onNodeSelect"
    />
    <div class="content">
      <div ref="detailContainerRef" class="detail-container">
        <template v-if="detailNavItem">
          <cl-auto-probe-item-detail
            :item="detailNavItem"
            :active-id="activeNavItem?.id"
            @row-click="onItemRowClick"
          />
        </template>
        <div v-else class="placeholder">
          {{ t('components.autoprobe.patterns.selectItem') }}
        </div>
      </div>

      <cl-auto-probe-results-container
        v-if="detailNavItem"
        :data="resultsData"
        :fields="resultsFields"
        :active-field-name="resultsActiveField?.name"
        :active-nav-item="activeNavItem"
        :url="form.url"
        :viewport="form.viewport"
        :active-id="activeId"
        @size-change="onSizeChange"
      />
    </div>
  </div>
</template>

<style scoped>
.autoprobe-detail-tab-patterns {
  height: 100%;
  display: flex;

  .content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;

    .detail-container {
      flex: 1;
      overflow: auto;
    }

    .placeholder {
      display: flex;
      justify-content: center;
      align-items: center;
      height: 100%;
      color: var(--el-text-color-secondary);
      font-style: italic;
    }
  }
}
</style>
