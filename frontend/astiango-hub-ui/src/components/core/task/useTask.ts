import { useRoute } from 'vue-router';
import { computed } from 'vue';
import { Store } from 'vuex';
import { useForm } from '@/components';
import useTaskService from '@/services/task/taskService';
import {
  getDefaultFormComponentData,
  getModeOptions,
  getModeOptionsDict,
  getPriorityLabel,
} from '@/utils';

// form component data
const formComponentData = getDefaultFormComponentData<Task>();

const useTask = (store: Store<RootStoreState>) => {
  // options for default mode
  const modeOptions = getModeOptions();
  const modeOptionsDict = computed(() => getModeOptionsDict());

  // route
  const route = useRoute();

  // task id
  const id = computed(() => route.params.id);

  return {
    ...useForm<Task>('task', store, useTaskService(store), formComponentData),
    id,
    modeOptions,
    modeOptionsDict,
    getPriorityLabel,
  };
};

export default useTask;
