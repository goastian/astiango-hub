import { useDetail } from '@/layouts/content';

const useAutoProbeDetail = () => {
  return {
    ...useDetail<AutoProbe>('autoprobe'),
  };
};

export default useAutoProbeDetail;
