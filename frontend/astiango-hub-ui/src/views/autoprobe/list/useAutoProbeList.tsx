import { useStore } from 'vuex';
import { useList } from '@/layouts';
import { computed } from 'vue';
import {
  ACTION_ADD,
  ACTION_DELETE,
  ACTION_FILTER,
  ACTION_FILTER_SEARCH,
  ACTION_RUN,
  ACTION_VIEW,
  FILTER_OP_CONTAINS,
  TABLE_COLUMN_NAME_ACTIONS,
} from '@/constants';
import {
  getIconByAction,
  onListFilterChangeByKey,
  setupAutoUpdate,
  translate,
} from '@/utils';
import {
  ClNavLink,
  ClAutoProbeTaskStatus,
  ClAutoProbePatternStats,
} from '@/components';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';

const t = translate;

const useAutoProbeList = () => {
  const router = useRouter();

  const ns: ListStoreNamespace = 'autoprobe';
  const store = useStore();
  const { commit } = store;

  const { actionFunctions } = useList<AutoProbeV2>(ns, store);
  const { getList, deleteByIdConfirm } = actionFunctions;

  // nav actions
  const navActions = computed<ListActionGroup[]>(() => [
    {
      name: 'common',
      children: [
        {
          action: ACTION_ADD,
          id: 'add-btn',
          className: 'add-btn',
          buttonType: 'label',
          label: t('views.autoprobe.navActions.new.label'),
          tooltip: t('views.autoprobe.navActions.new.tooltip'),
          icon: getIconByAction(ACTION_ADD),
          onClick: () => {
            commit(`${ns}/showDialog`, 'create');
          },
        },
      ],
    },
    {
      action: ACTION_FILTER,
      name: 'filter',
      children: [
        {
          action: ACTION_FILTER_SEARCH,
          id: 'filter-search',
          className: 'search',
          placeholder: t(
            'views.autoprobe.navActions.filter.search.placeholder'
          ),
          onChange: onListFilterChangeByKey(
            store,
            ns,
            'name',
            FILTER_OP_CONTAINS
          ),
        },
      ],
    },
  ]);

  // table columns
  const tableColumns = computed<TableColumns<AutoProbeV2>>(
    () =>
      [
        {
          className: 'name',
          key: 'name',
          label: t('views.autoprobe.table.columns.name'),
          icon: ['fa', 'font'],
          width: '150',
          value: (row: AutoProbeV2) => (
            <ClNavLink path={`/autoprobes/${row._id}`} label={row.name} />
          ),
          hasSort: true,
          hasFilter: true,
          allowFilterSearch: true,
        },
        {
          key: 'url',
          label: t('views.autoprobe.table.columns.url'),
          icon: ['fa', 'at'],
          width: 'auto',
          minWidth: '200',
          value: (row: AutoProbeV2) => (
            <ClNavLink path={row.url} label={row.url} external />
          ),
          hasFilter: true,
          allowFilterSearch: true,
        },
        {
          key: 'last_task',
          label: t('views.autoprobe.table.columns.lastTask'),
          icon: ['fa', 'heartbeat'],
          width: '120',
          value: (row: AutoProbeV2) => {
            const status = row.last_task_status;
            const error = row.last_task_error;
            if (!status) return;
            return (
              <ClAutoProbeTaskStatus
                status={status}
                error={error}
                clickable
                onClick={() => router.push(`/autoprobes/${row._id}/tasks`)}
              />
            );
          },
          hasFilter: true,
          allowFilterSearch: true,
        },
        // {
        //   key: 'patterns',
        //   label: t('views.autoprobe.table.columns.patterns'),
        //   icon: ['fa', 'network-wired'],
        //   width: '200',
        //   value: (row: AutoProbe) => {
        //     return (
        //       <ClAutoProbePatternStats
        //         autoprobe={row}
        //         clickable
        //         onClick={() => router.push(`/autoprobes/${row._id}/patterns`)}
        //       />
        //     );
        //   },
        //   hasFilter: true,
        //   allowFilterSearch: true,
        // },
        {
          key: TABLE_COLUMN_NAME_ACTIONS,
          label: t('components.table.columns.actions'),
          fixed: 'right',
          width: '150',
          buttons: [
            {
              tooltip: t('common.actions.view'),
              onClick: async row => {
                await router.push(`/autoprobes/${row._id}`);
              },
              action: ACTION_VIEW,
            },
            {
              tooltip: t('common.actions.run'),
              onClick: async row => {
                await ElMessageBox.confirm(
                  t('common.messageBox.confirm.run'),
                  t('common.actions.restart'),
                  {
                    type: 'warning',
                    confirmButtonClass: 'confirm-btn',
                  }
                );
                try {
                  await store.dispatch(`${ns}/runTask`, { id: row._id });
                  ElMessage.success(t('common.message.success.run'));
                } catch (e) {
                  ElMessage.error((e as Error).message);
                }
              },
              action: ACTION_RUN,
              contextMenu: true,
            },
            {
              tooltip: t('common.actions.delete'),
              onClick: deleteByIdConfirm,
              action: ACTION_DELETE,
              contextMenu: true,
            },
          ],
          disableTransfer: true,
        },
      ] as TableColumns<AutoProbeV2>
  );

  const rowKey = (row: AutoProbeV2) => {
    return JSON.stringify([
      row._id,
      row.url,
      row.last_task_status,
      row.page_pattern,
    ]);
  };

  setupAutoUpdate(getList);

  return {
    ...useList<AutoProbeV2>(ns, store),
    navActions,
    tableColumns,
    rowKey,
  };
};

export default useAutoProbeList;
