import { useDetail } from '@/layouts';

const useScheduleDetail = () => {
  const ns: ListStoreNamespace = 'schedule';

  return {
    ...useDetail<Schedule>(ns),
  };
};

export default useScheduleDetail;
