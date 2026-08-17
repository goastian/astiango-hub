<script setup lang="ts">
import { computed } from 'vue';
import { translate } from '@/utils';

const props = defineProps<{
  autoprobe: AutoProbe;
  clickable?: boolean;
}>();

const emit = defineEmits<{
  (e: 'click'): void;
}>();

// i18n
const t = translate;

/**
 * Recursively counts all lists in a given pattern (including nested lists)
 */
const countAllLists = (
  pattern: ItemPattern | PagePattern | undefined
): number => {
  if (!pattern || (!pattern.lists && !pattern.fields)) return 0;

  // Count top-level lists
  let count = pattern.lists?.length || 0;

  // Count nested lists in each list's item pattern
  if (pattern.lists) {
    for (const list of pattern.lists) {
      count += countAllLists(list.item_pattern);
    }
  }

  return count;
};

/**
 * Recursively counts all fields in a given pattern (including fields in nested lists)
 */
const countAllFields = (
  pattern: ItemPattern | PagePattern | undefined
): number => {
  if (!pattern || (!pattern.lists && !pattern.fields)) return 0;

  // Count top-level fields
  let count = pattern.fields?.length || 0;

  // Count fields in nested lists
  if (pattern.lists) {
    for (const list of pattern.lists) {
      count += countAllFields(list.item_pattern);
    }
  }

  return count;
};

// Computed statistics
const hasPattern = computed(() => {
  return !!props.autoprobe?.page_pattern;
});

const totalFields = computed(() => {
  return countAllFields(props.autoprobe?.page_pattern);
});

const totalLists = computed(() => {
  return countAllLists(props.autoprobe?.page_pattern);
});

const hasPagination = computed(() => {
  return !!props.autoprobe?.page_pattern?.pagination;
});

const paginationType = computed(() => {
  if (!hasPagination.value) return null;
  return props.autoprobe?.page_pattern?.pagination?.type;
});

defineOptions({ name: 'ClAutoProbePatternStats' });
</script>

<template>
  <div v-if="hasPattern" class="autoprobe-stats">
    <cl-tag
      :icon="['fa', 'list']"
      :label="totalFields.toString()"
      :tooltip="t('components.autoprobe.stats.totalFields')"
      type="primary"
      :clickable="clickable"
      @click="emit('click')"
    />
    <cl-tag
      :icon="['fa', 'table']"
      :label="totalLists.toString()"
      :tooltip="t('components.autoprobe.stats.totalLists')"
      type="success"
      :clickable="clickable"
      @click="emit('click')"
    />
    <cl-tag
      v-if="hasPagination"
      :icon="['fa', 'copy']"
      :label="paginationType"
      :tooltip="t('components.autoprobe.stats.paginationType')"
      type="warning"
      :clickable="clickable"
      @click="emit('click')"
    />
    <cl-tag
      v-else
      :icon="['fa', 'copy']"
      label="none"
      :tooltip="t('components.autoprobe.stats.noPagination')"
      type="info"
      :clickable="clickable"
      @click="emit('click')"
    />
  </div>
  <div v-else class="autoprobe-stats-empty">
    <cl-tag
      :icon="['fa', 'question-circle']"
      :label="t('common.status.unknown')"
      type="info"
      :clickable="clickable"
      @click="emit('click')"
    />
  </div>
</template>

<style scoped>
.autoprobe-stats {
  display: flex;
  gap: 10px;
}

.autoprobe-stats-empty {
  display: flex;
}
</style>
