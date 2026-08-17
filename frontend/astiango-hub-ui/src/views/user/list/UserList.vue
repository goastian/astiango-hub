<script setup lang="ts">
import useUserList from './useUserList';

defineProps<{
  noActions: boolean;
  embedded: boolean;
}>();

const {
  navActions,
  tableColumns,
  tableData,
  tableTotal,
  tablePagination,
  actionFunctions,
  selectableFunction,
} = useUserList();

defineOptions({ name: 'ClUserList' });
</script>

<template>
  <cl-list-layout
    class="user-list"
    :action-functions="actionFunctions"
    :nav-actions="navActions"
    :table-pagination="tablePagination"
    :table-columns="tableColumns"
    :table-data="tableData"
    :table-total="tableTotal"
    :no-actions="noActions"
    :embedded="embedded"
    :selectable-function="selectableFunction"
  >
    <template #extra>
      <!-- Dialogs (handled by store) -->
      <cl-create-edit-user-dialog />
      <!-- ./Dialogs -->
    </template>
  </cl-list-layout>
</template>
