<script setup lang="ts">
import { useStore } from 'vuex';
import { useAutoProbe } from '@/components';

// store
const store = useStore();

const {
  activeDialogKey,
  createEditDialogVisible,
  actionFunctions,
  formList,
  confirmDisabled,
  confirmLoading,
} = useAutoProbe(store);
defineOptions({ name: 'ClCreateEditAutoProbeDialog' });
</script>

<template>
  <cl-create-edit-dialog
    :type="activeDialogKey"
    :visible="createEditDialogVisible"
    :action-functions="actionFunctions"
    :batch-form-data="formList"
    :confirm-disabled="confirmDisabled"
    :confirm-loading="confirmLoading"
  >
    <template #default>
      <cl-auto-probe-form />
    </template>
  </cl-create-edit-dialog>
</template>
