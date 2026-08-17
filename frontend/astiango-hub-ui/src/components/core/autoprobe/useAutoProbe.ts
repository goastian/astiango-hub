import { computed } from 'vue';
import { Store } from 'vuex';
import { useForm } from '@/components';
import useAutoProbeService from '@/services/autoprobe/autoprobeService';
import { getDefaultFormComponentData } from '@/utils/form';

// form component data
const formComponentData = getDefaultFormComponentData<AutoProbeV2>();

const useAutoProbe = (store: Store<RootStoreState>) => {
  // store
  const ns = 'autoprobe';

  // form rules
  const formRules: FormRules = {
    url: {
      pattern: /^https?:\/\/.+/,
      message: 'URL is not valid (must start with http:// or https://)',
    },
  };

  return {
    ...useForm<AutoProbeV2>(
      'autoprobe',
      store,
      useAutoProbeService(store),
      formComponentData
    ),
    formRules,
  };
};

export default useAutoProbe;
