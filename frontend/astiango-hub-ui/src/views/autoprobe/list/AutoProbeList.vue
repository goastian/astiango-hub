<script setup lang="ts">
import { useAutoProbeList } from '@/views';

const {
  navActions,
  tableColumns,
  tableData,
  tableTotal,
  tablePagination,
  actionFunctions,
  rowKey,
} = useAutoProbeList();

defineOptions({ name: 'ClAutoProbeList' });
</script>

<template>
  <cl-list-layout
    class="autoprobe-list"
    :action-functions="actionFunctions"
    :nav-actions="navActions"
    :table-pagination="tablePagination"
    :table-columns="tableColumns"
    :table-data="tableData"
    :table-total="tableTotal"
    :row-key="rowKey"
  >
    <template #extra>
      <!-- Dialogs (handled by store) -->
      <cl-create-edit-auto-probe-dialog />
      <!-- ./Dialogs -->
    </template>
  </cl-list-layout>
</template>
