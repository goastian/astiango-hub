<script setup lang="ts">
import { useTaskDetail } from '@/views';
import { formatTimeAgo, isPro } from '@/utils';

const { activeTabName } = useTaskDetail();

const navItemLabelFn = (item: NavItem<Task>) => {
  if (!item.data) return item.label;
  const spiderName = item.data.spider?.name;
  const createdAt = formatTimeAgo(item.data.created_at!, 'mini-minute-now');
  return `${spiderName} - ${createdAt}`;
};

defineOptions({ name: 'ClTaskDetail' });
</script>

<template>
  <cl-detail-layout
    store-namespace="task"
    :nav-item-label-fn="navItemLabelFn"
  >
    <template #actions>
      <cl-task-detail-actions-common />
      <cl-task-detail-actions-logs v-if="activeTabName === 'logs'" />
      <template v-if="isPro()">
        <cl-task-detail-actions-data v-if="activeTabName === 'data'" />
      </template>
    </template>
  </cl-detail-layout>
</template>
