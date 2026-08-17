import { Store } from 'vuex';
import { getDefaultService } from '@/utils';

const useAutoProbeService = (
  store: Store<RootStoreState>
): Services<AutoProbeV2> => {
  const ns: ListStoreNamespace = 'autoprobe';

  return {
    ...getDefaultService<AutoProbeV2>(ns, store),
  };
};

export default useAutoProbeService;
