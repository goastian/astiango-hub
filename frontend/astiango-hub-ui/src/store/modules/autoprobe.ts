import {
  getDefaultStoreActions,
  getDefaultStoreGetters,
  getDefaultStoreMutations,
  getDefaultStoreState,
} from '@/utils/store';
import {
  TAB_NAME_OVERVIEW,
  TAB_NAME_PATTERNS,
  TAB_NAME_TASKS,
} from '@/constants/tab';
import { translate } from '@/utils/i18n';
import useRequest from '@/services/request';
import { getViewPortOptions } from '@/utils';

// i18n
const t = translate;

const { get, post } = useRequest();

const state = {
  ...getDefaultStoreState<AutoProbeV2>('autoprobe'),
  newFormFn: () => {
    return {
      run_on_create: true,
      viewport: getViewPortOptions().find(v => v.value === 'pc-normal')
        ?.viewport,
    };
  },
  tabs: [
    { id: TAB_NAME_OVERVIEW, title: t('common.tabs.overview') },
    { id: TAB_NAME_TASKS, title: t('common.tabs.tasks') },
    { id: TAB_NAME_PATTERNS, title: t('common.tabs.patterns') },
  ],
  pagePattern: undefined,
  pagePatternData: [],
} as AutoProbeStoreState;

const getters = {
  ...getDefaultStoreGetters<AutoProbeV2>(),
} as AutoProbeStoreGetters;

const mutations = {
  ...getDefaultStoreMutations<AutoProbeV2>(),
  setPagePattern(state: AutoProbeStoreState, pagePattern: PagePatternV2) {
    state.pagePattern = pagePattern;
  },
  resetPagePattern(state: AutoProbeStoreState) {
    state.pagePattern = undefined;
  },
  setPagePatternData(
    state: AutoProbeStoreState,
    pagePatternData: PatternDataV2[]
  ) {
    state.pagePatternData = pagePatternData;
  },
  resetPagePatternData(state: AutoProbeStoreState) {
    state.pagePatternData = [];
  },
} as AutoProbeStoreMutations;

const endpoint = '/ai/autoprobes';

const actions = {
  ...getDefaultStoreActions<AutoProbeV2>(endpoint),
  runTask: async (
    _: StoreActionContext<AutoProbeStoreState>,
    { id }: { id: string }
  ) => {
    await post(`${endpoint}/${id}/tasks`);
  },
  cancelTask: async (
    _: StoreActionContext<AutoProbeStoreState>,
    { id }: { id: string }
  ) => {
    await post(`${endpoint}/tasks/${id}/cancel`);
  },
  getPagePattern: async (
    { commit, state }: StoreActionContext<AutoProbeStoreState>,
    { id }: { id: string }
  ) => {
    const res = await get(`${endpoint}/${id}/pattern`);
    commit('setPagePattern', res.data);
    
    // Also update the form data so the component can access it
    if (state.form) {
      commit('setForm', {
        ...state.form,
        page_pattern: res.data
      });
    }
  },
  getPagePatternData: async (
    { commit, state }: StoreActionContext<AutoProbeStoreState>,
    { id }: { id: string }
  ) => {
    const res = await get(`${endpoint}/${id}/pattern/results`);
    commit('setPagePatternData', res.data);
    
    // Transform PatternDataV2[] array into structured page data object
    const structuredData: Record<string, any> = {};
    if (Array.isArray(res.data)) {
      res.data.forEach((patternData: any) => {
        // For now, just use a simple mapping - we might need to enhance this later
        // based on how the pattern hierarchy should map to data
        if (patternData.pattern_id && patternData.data !== undefined) {
          structuredData[patternData.pattern_id] = patternData.data;
        }
      });
    }
    
    // Also update the form data so the component can access it
    if (state.form) {
      commit('setForm', {
        ...state.form,
        page_data: structuredData
      });
    }
  },
} as AutoProbeStoreActions;

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions,
} as AutoProbeStoreModule;
