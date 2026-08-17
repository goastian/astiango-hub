import {
  getDefaultStoreActions,
  getDefaultStoreGetters,
  getDefaultStoreMutations,
  getDefaultStoreState,
} from '@/utils/store';
import { TAB_NAME_OVERVIEW } from '@/constants';
import { translate } from '@/utils/i18n';
import useRequest from '@/services/request';

const t = translate;

const { get } = useRequest();

const state = {
  ...getDefaultStoreState<NotificationAlert>('notificationAlert'),
  newFormFn: () => ({
    name: '',
    enabled: true,
    has_metric_target: false,
    operator: 'ge',
    lasting_seconds: 60 * 5,
    level: 'warning',
  }),
  tabs: [{ id: TAB_NAME_OVERVIEW, title: t('common.tabs.overview') }],
  allAlerts: [],
} as NotificationAlertStoreState;

const getters = {
  ...getDefaultStoreGetters<NotificationAlert>(),
} as NotificationAlertStoreGetters;

const mutations = {
  ...getDefaultStoreMutations<NotificationAlert>(),
  setAllAlerts: (
    state: NotificationAlertStoreState,
    allAlerts: NotificationAlert[]
  ) => {
    state.allAlerts = allAlerts;
  },
  resetAllAlerts: (state: NotificationAlertStoreState) => {
    state.allAlerts = [];
  },
} as NotificationAlertStoreMutations;

const actions = {
  ...getDefaultStoreActions<NotificationAlert>('/notifications/alerts'),
  getAllAlerts: async ({ commit }: StoreActionContext) => {
    const res = await get<NotificationAlert[]>('/notifications/alerts', {
      size: 10000,
    });
    commit('setAllAlerts', res.data || []);
  },
} as NotificationAlertStoreActions;

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions,
} as NotificationAlertStoreModule;
