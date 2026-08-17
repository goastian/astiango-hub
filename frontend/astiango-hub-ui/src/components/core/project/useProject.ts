import { Store } from 'vuex';
import useForm from '@/components/ui/form/useForm';
import useProjectService from '@/services/project/projectService';
import { getDefaultFormComponentData } from '@/utils/form';

// form component data
const formComponentData = getDefaultFormComponentData<Project>();

const useProject = (store: Store<RootStoreState>) => {
  // form rules
  const formRules: FormRules = {};

  return {
    ...useForm<Project>(
      'project',
      store,
      useProjectService(store),
      formComponentData
    ),
    formRules,
  };
};

export default useProject;
