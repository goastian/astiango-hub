import { useStore } from 'vuex';
import { useDetail } from '@/layouts';
import useFileService from '@/services/utils/file';

const useSpiderDetail = () => {
  const ns: ListStoreNamespace = 'spider';
  const store = useStore();

  return {
    ...useDetail<Spider>('spider'),
    ...useFileService(ns, store),
  };
};

export default useSpiderDetail;
