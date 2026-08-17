<script setup lang="ts">
import { computed } from 'vue';
import { translate } from '@/utils';
import { TagProps } from '@/components/ui/tag/types';

const props = defineProps<{
  status: AutoProbeTaskStatus;
  size?: BasicSize;
  error?: string;
  clickable?: boolean;
}>();

const emit = defineEmits<{
  (e: 'click'): void;
}>();

// i18n
const t = translate;

const data = computed<TagProps>(() => {
  const { status, error } = props;
  switch (status) {
    case 'pending':
      return {
        label: t('components.autoprobe.task.status.label.pending'),
        tooltip: t('components.autoprobe.task.status.tooltip.pending'),
        type: 'primary',
        icon: ['fa', 'hourglass-start'],
        spinning: true,
      };
    case 'running':
      return {
        label: t('components.autoprobe.task.status.label.running'),
        tooltip: t('components.autoprobe.task.status.tooltip.running'),
        type: 'warning',
        icon: ['fa', 'spinner'],
        spinning: true,
      };
    case 'completed':
      return {
        label: t('components.autoprobe.task.status.label.completed'),
        tooltip: t('components.autoprobe.task.status.tooltip.completed'),
        type: 'success',
        icon: ['fa', 'check'],
      };
    case 'failed':
      return {
        label: t('components.autoprobe.task.status.label.failed'),
        tooltip: `${t('components.autoprobe.task.status.tooltip.failed')}<br><span style="color: 'var(--cl-red)">${error}</span>`,
        type: 'danger',
        icon: ['fa', 'times'],
      };
    case 'cancelled':
      return {
        label: t('components.autoprobe.task.status.label.cancelled'),
        tooltip: t('components.autoprobe.task.status.tooltip.cancelled'),
        type: 'info',
        icon: ['fa', 'ban'],
      };
    default:
      return {
        label: t('components.autoprobe.task.status.label.unknown'),
        tooltip: t('components.autoprobe.task.status.tooltip.unknown'),
        type: 'info',
        icon: ['fa', 'question'],
      };
  }
});
defineOptions({ name: 'ClAutoProbeTaskStatus' });
</script>

<template>
  <cl-tag
    class-name="autoprobe-task-status"
    :key="data"
    :icon="data.icon"
    :label="data.label"
    :spinning="data.spinning"
    :type="data.type"
    :size="size"
    :clickable="clickable"
    @click="emit('click')"
  >
    <template #tooltip>
      <div v-html="data.tooltip" />
    </template>
  </cl-tag>
</template>

<style scoped>

</style>
