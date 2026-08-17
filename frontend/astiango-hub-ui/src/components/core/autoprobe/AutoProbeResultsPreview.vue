<script setup lang="ts">
import {
  type CSSProperties,
  onMounted,
  ref,
  computed,
  watch,
  onBeforeUnmount,
} from 'vue';
import useRequest from '@/services/request';
import { getIconByPageElementType, translate } from '@/utils';
import { debounce } from 'lodash';
import type { Property } from 'csstype';

const props = defineProps<{
  activeId: string;
  activeNavItem?: AutoProbeNavItem;
  viewport?: PageViewPort;
}>();

const t = translate;

const { get } = useRequest();

const previewRef = ref<HTMLDivElement | null>(null);
const previewLoading = ref(false);
const previewResult = ref<PagePreviewResult>();

const screenshotRef = ref<HTMLDivElement | null>(null);
const screenshotScale = ref(1);
const updateOverlayScale = () => {
  const { viewport } = props;
  const rect = screenshotRef.value?.getBoundingClientRect();
  if (!rect) return 1;
  screenshotScale.value = rect.width / (viewport?.width ?? 1280);
};

const displayConfig = ref({
  showLabel: true,
  focusMode: true,
});

let resizeObserver: ResizeObserver | null = null;

onMounted(() => {
  // Initial calculation if reference is already available
  if (screenshotRef.value) {
    setupResizeObserver();
  }

  // Watch for when the reference becomes available
  watch(screenshotRef, newRef => {
    if (newRef) {
      setupResizeObserver();
    }
  });

  const handle = setInterval(() => {
    if (!screenshotRef.value) return;
    screenshotRef.value.addEventListener('resize', updateOverlayScale);
    updateOverlayScale();
    clearInterval(handle);
  }, 10);
  return () => {
    screenshotRef.value?.removeEventListener('resize', updateOverlayScale);
  };
});

const setupResizeObserver = () => {
  // Clean up existing observer if there is one
  if (resizeObserver) {
    resizeObserver.disconnect();
  }

  resizeObserver = new ResizeObserver(() => {
    updateOverlayScale();
  });

  resizeObserver.observe(screenshotRef.value!);
  updateOverlayScale();
};

// Clean up function
onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect();
  }
});

const getPreview = debounce(async () => {
  const { activeId } = props;
  previewLoading.value = true;
  try {
    const res = await get<any, ResponseWithData<PagePreviewResult>>(
      `/ai/autoprobes/${activeId}/preview`
    );
    previewResult.value = res.data;
  } finally {
    previewLoading.value = false;
  }
});
onMounted(getPreview);

const getElementMaskStyle = (el: PageElement): CSSProperties => {
  return {
    position: 'absolute',
    left: el.coordinates.left * screenshotScale.value + 'px',
    top: el.coordinates.top * screenshotScale.value + 'px',
    width: el.coordinates.width * screenshotScale.value + 'px',
    height: el.coordinates.height * screenshotScale.value + 'px',
  };
};

const pageElements = computed<PageElement[]>(() => {
  const { activeNavItem } = props;
  const { focusMode } = displayConfig.value;
  if (!previewResult.value?.page_elements) {
    return [];
  }
  const fieldElements = previewResult.value.page_elements.filter(
    el => el.type === 'field'
  );
  const listElements = previewResult.value.page_elements.filter(
    el => el.type === 'list'
  );

  // If focus mode is not enabled, return all elements
  if (!focusMode) {
    const allListItemElements = listElements.flatMap(el => el.children || []);
    const allListFieldElements = allListItemElements.flatMap(
      el => el.children || []
    );
    return [
      ...fieldElements,
      ...listElements,
      ...allListItemElements,
      ...allListFieldElements,
    ];
  }

  // If focus mode is enabled, filter elements based on the active navigation item
  if (!activeNavItem) {
    return [];
  }
  switch (activeNavItem.type) {
    case 'page_pattern':
      return [...fieldElements, ...listElements];
    case 'list':
      const listElement = listElements.find(
        el => el.name === activeNavItem.name
      );
      if (!listElement) {
        return [];
      }
      const listItemElements = listElements.flatMap(el => el.children || []);
      const listFieldElements = listItemElements.flatMap(
        el => el.children || []
      );
      return [
        { ...listElement, active: true },
        ...listItemElements,
        ...listFieldElements,
      ];
    case 'field':
      // Non-list field
      if (activeNavItem.parent?.type === 'page_pattern') {
        const fieldElement = fieldElements.find(
          el => el.name === activeNavItem.name
        );
        if (!fieldElement) {
          return [];
        }
        return [{ ...fieldElement, active: true }];
      }
      // List item field
      if (activeNavItem.parent?.type === 'list') {
        const listElement = listElements.find(
          el => el.name === activeNavItem.parent!.name
        );
        if (!listElement) {
          return [];
        }
        const listItemElements = listElements.flatMap(el => el.children || []);
        const listFieldElements = listItemElements
          .flatMap(el => el.children || [])
          .filter(el => el.name === activeNavItem.name)
          .map(el => ({ ...el, active: true }));
        return [...listItemElements, ...listFieldElements];
      }
      return [];
    case 'pagination':
      const paginationElement = previewResult.value.page_elements.find(
        el => el.type === 'pagination' && el.name === activeNavItem.name
      );
      if (!paginationElement) {
        return [];
      }
      return [{ ...paginationElement, active: true }];
    default:
      return [];
  }
});

const viewportDisplay = computed(() => {
  const { viewport } = props;
  if (!viewport) return '';
  return `${viewport.width}x${viewport.height}`;
});

defineExpose({
  updateOverlayScale,
});

defineOptions({ name: 'ClAutoProbeResultsPreview' });
</script>

<template>
  <div ref="previewRef" class="preview">
    <div class="preview-control">
      <cl-tag :icon="['fa', 'desktop']" :label="viewportDisplay" />
      <el-checkbox
        v-model="displayConfig.showLabel"
        :label="t('components.autoprobe.pagePattern.displayConfig.showLabel')"
      />
      <el-checkbox
        v-model="displayConfig.focusMode"
        :label="t('components.autoprobe.pagePattern.displayConfig.focusMode')"
      />
    </div>
    <div v-loading="previewLoading" class="preview-container">
      <div v-if="previewResult" class="preview-overlay">
        <img
          ref="screenshotRef"
          class="screenshot"
          :src="previewResult.screenshot_base64"
        />
        <div
          v-for="(el, index) in pageElements"
          :key="index"
          class="element-mask"
          :class="el.active ? 'active' : ''"
          :style="getElementMaskStyle(el)"
        >
          <el-badge
            :type="el.active ? 'danger' : 'primary'"
            :hidden="!displayConfig.showLabel"
            :badge-style="{
              opacity: el.active ? 1 : 0.5,
            }"
          >
            <template #content>
              <span style="margin-right: 5px">
                <cl-icon :icon="getIconByPageElementType(el.type)" />
              </span>
              {{ el.name }}
            </template>
          </el-badge>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.preview {
  overflow: hidden;
  height: calc(100% - 41px);

  .preview-control {
    height: 40px;
    border-bottom: 1px solid var(--el-border-color);
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 8px;

    .el-checkbox {
      margin: 0;
    }
  }

  .preview-container {
    position: relative;
    width: 100%;
    height: calc(100% - 40px);
    overflow: auto;
    scrollbar-width: none;

    .preview-overlay {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      z-index: 1;

      img.screenshot {
        width: fit-content;
        max-width: 100%;
      }

      .element-mask {
        border: 1px solid var(--el-color-primary-light-5);
        border-radius: 4px;
        z-index: 1;
        cursor: pointer;

        &:hover {
          background: rgba(64, 156, 255, 0.2);
        }

        &.active {
          z-index: 2;
          pointer-events: none;
          border: 3px solid var(--el-color-danger);
        }
      }
    }
  }
}
</style>
