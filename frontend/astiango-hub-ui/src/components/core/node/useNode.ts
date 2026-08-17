import { Store } from 'vuex';
import useForm from '@/components/ui/form/useForm';
import useNodeService from '@/services/node/nodeService';
import { getDefaultFormComponentData } from '@/utils/form';
import { computed } from 'vue';

type Node = CNode;

// form component data
const formComponentData = getDefaultFormComponentData<Node>();

const useNode = (store: Store<RootStoreState>) => {
  // store
  const ns = 'node';
  const { node: state } = store.state;

  // form rules
  const formRules: FormRules = {};

  const allNodesSorted = computed(() => {
    return state.allNodes.sort((a, b) => {
      if (a.is_master) return -1;
      if (b.is_master) return 1;
      return a.name!.localeCompare(b.name!);
    });
  });

  const activeNodesSorted = computed(() => {
    return allNodesSorted.value.filter(node => node.active);
  });

  return {
    ...useForm<Node>(ns, store, useNodeService(store), formComponentData),
    formRules,
    allNodesSorted,
    activeNodesSorted,
  };
};

export default useNode;
