import { onBeforeUnmount } from 'vue';
import { useStore } from 'vuex';
import { useDetail } from '@/layouts';

const useTaskDetail = () => {
  // store
  const ns = 'task';
  const store = useStore();

  // dispose
  onBeforeUnmount(() => {
    store.commit(`${ns}/resetLogContent`);
    store.commit(`${ns}/resetLogPagination`);
    store.commit(`${ns}/resetLogTotal`);
    store.commit(`${ns}/disableLogAutoUpdate`);
  });

  return {
    ...useDetail<Task>('task'),
  };
};

export default useTaskDetail;
