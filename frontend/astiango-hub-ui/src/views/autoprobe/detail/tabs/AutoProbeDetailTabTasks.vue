<script setup lang="tsx">
import { computed, onBeforeMount, ref } from 'vue';
import { getDefaultPagination, setupAutoUpdate, translate } from '@/utils';
import useRequest from '@/services/request';
import { ClNavLink, ClAutoProbeTaskStatus } from '@/components';
import { useDetail } from '@/layouts';
import { useStore } from 'vuex';
import { ElMessageBox, ElMessage } from 'element-plus';
import {
  ACTION_CANCEL,
  ACTION_DELETE,
  ACTION_RESTART,
  ACTION_VIEW,
  TABLE_COLUMN_NAME_ACTIONS,
} from '@/constants';

const t = translate;

const ns: ListStoreNamespace = 'autoprobe';
const store = useStore();

const { getList, del } = useRequest();

const { activeId } = useDetail<AutoProbe>('autoprobe');

const tasks = ref<AutoProbeTask[]>([]);
const taskTotal = ref(0);

const getTasks = async () => {
  const res = await getList(`/ai/autoprobes/${activeId.value}/tasks`, {
    page: tablePagination.value.page,
    size: tablePagination.value.size,
  });
  tasks.value = res.data || [];
  taskTotal.value = res.total || 0;
};

// Cancel task function
const cancelTask = async (row: AutoProbeTask, force: boolean = false) => {
  if (force) {
    ElMessage.info(t('common.message.info.forceCancel'));
  } else {
    ElMessage.info(t('common.message.info.cancel'));
  }

  try {
    await store.dispatch(`${ns}/cancelTask`, { id: row._id });
    await getTasks(); // Refresh the list
    ElMessage.success(t('common.message.success.cancel'));
  } catch (error) {
    console.error('Failed to cancel task:', error);
    ElMessage.error(t('common.message.error.action'));
  }
};

// Delete task function
const deleteTask = async (row: AutoProbeTask) => {
  try {
    await del(`/ai/autoprobes/tasks/${row._id}`);
    ElMessage.success(t('common.message.success.delete'));
    await getTasks(); // Refresh the list
  } catch (error) {
    console.error('Failed to delete task:', error);
    ElMessage.error(t('common.message.error.action'));
  }
};

const tableColumns = computed<TableColumns<AutoProbeTask>>(() => {
  return [
    {
      key: 'url',
      label: t('views.autoprobe.table.columns.url'),
      icon: ['fa', 'at'],
      width: 'auto',
      minWidth: '400px',
      value: (row: AutoProbeTask) => (
        <ClNavLink path={row.url} label={row.url} external />
      ),
      allowFilterSearch: true,
    },
    {
      key: 'query',
      label: t('views.autoprobe.table.columns.query'),
      icon: ['fa', 'search'],
      width: '200px',
      value: (row: AutoProbeTask) => row.query,
    },
    {
      key: 'status',
      label: t('views.autoprobe.table.columns.status'),
      icon: ['fa', 'info-circle'],
      width: '120px',
      value: (row: AutoProbeTask) => (
        <ClAutoProbeTaskStatus status={row.status} error={row.error} />
      ),
    },
    {
      key: TABLE_COLUMN_NAME_ACTIONS,
      label: t('components.table.columns.actions'),
      icon: ['fa', 'tools'],
      width: '150px',
      fixed: 'right',
      buttons: (row: AutoProbeTask) =>
        (
          [
            {
              tooltip: t('common.actions.view'),
              onClick: async (row: AutoProbeTask) => {
                // View task details implementation
                ElMessage.info('View task details - to be implemented');
              },
              action: ACTION_VIEW,
            },
            {
              tooltip: t('common.actions.restart'),
              contextMenu: true,
              onClick: async (_: AutoProbeTask) => {
                await ElMessageBox.confirm(
                  t('common.messageBox.confirm.restart'),
                  t('common.actions.restart'),
                  {
                    type: 'warning',
                    confirmButtonClass: 'restart-confirm-btn',
                  }
                );
                await store.dispatch(`${ns}/runTask`, {
                  id: activeId.value,
                });
                ElMessage.success(t('common.message.success.restart'));
                await getTasks();
              },
              action: ACTION_RESTART,
            },
            {
              tooltip: t('common.actions.cancel'),
              contextMenu: true,
              onClick: async (row: AutoProbeTask) => {
                await ElMessageBox.confirm(
                  t('common.messageBox.confirm.cancel'),
                  t('common.actions.cancel'),
                  {
                    type: 'warning',
                    confirmButtonClass: 'cancel-confirm-btn',
                  }
                );
                await cancelTask(row, false);
              },
              action: ACTION_CANCEL,
            },
            {
              tooltip: t('common.actions.delete'),
              contextMenu: true,
              onClick: async (row: AutoProbeTask) => {
                await ElMessageBox.confirm(
                  t('common.messageBox.confirm.delete'),
                  t('common.actions.delete'),
                  {
                    type: 'warning',
                    confirmButtonClass: 'delete-confirm-btn',
                  }
                );
                await deleteTask(row);
              },
              action: ACTION_DELETE,
            },
          ] as TableColumnButton<AutoProbeTask>[]
        ).filter(btn => {
          switch (btn.action) {
            case ACTION_CANCEL:
              return row.status === 'pending' || row.status === 'running';
            case ACTION_DELETE:
              return row.status !== 'pending' && row.status !== 'running';
            default:
              return true;
          }
        }),
      disableTransfer: true,
    },
  ];
});

const tableData = computed(() => tasks.value);
const tablePagination = ref<TablePagination>(getDefaultPagination());
const tableTotal = computed(() => taskTotal.value);
const onTablePaginationChange = async (pagination: TablePagination) => {
  tablePagination.value = pagination;
  await getTasks();
};

const onClickRun = async () => {
  await ElMessageBox.confirm(
    t('common.messageBox.confirm.run'),
    t('common.actions.run')
  );
  try {
    await store.dispatch(`${ns}/runTask`, { id: activeId.value });
    await getTasks(); // Refresh the list after running a task
  } catch (error) {
    console.error('Failed to run task:', error);
    ElMessage.error(t('common.message.error.action'));
  }
};

onBeforeMount(getTasks);
setupAutoUpdate(getTasks);

defineOptions({ name: 'ClAutoProbeDetailTabTasks' });
</script>

<template>
  <div class="tasks-container">
    <cl-table
      :columns="tableColumns"
      :data="tableData"
      :page="tablePagination.page"
      :page-size="tablePagination.size"
      :total="tableTotal"
      selectable
      embedded
      @pagination-change="onTablePaginationChange"
    >
      <template #empty>
        <cl-label-button
          :icon="['fa', 'play']"
          :label="t('views.autoprobe.navActions.run.label')"
          @click="onClickRun"
        />
      </template>
    </cl-table>
  </div>
</template>

<style scoped>
.tasks-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}
</style>
