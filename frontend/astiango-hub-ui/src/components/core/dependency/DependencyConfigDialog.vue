<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useStore } from 'vuex';
import { translate } from '@/utils';
import { ClForm } from '@/components';

const t = translate;

const ns: ListStoreNamespace = 'dependency';
const store = useStore();
const { dependency: state, node: nodeState } = store.state as RootStoreState;

const visible = computed(() => state.activeDialogKey === 'config');

const form = computed(() => state.config);
const formRef = ref<typeof ClForm>();

const confirmLoading = ref(false);
const onConfirm = async () => {
  await formRef.value?.validate();
  confirmLoading.value = true;
  try {
    await store.dispatch(`${ns}/saveDependencyConfig`);
  } finally {
    confirmLoading.value = false;
    store.commit(`${ns}/hideDialog`);
    setTimeout(() => {
      store.commit(`${ns}/resetConfig`);
    }, 300);
  }
};

const onClose = () => {
  store.commit(`${ns}/hideDialog`);
  setTimeout(() => {
    store.commit(`${ns}/resetConfig`);
  }, 300);
};

watch(visible, async () => {
  if (!visible.value) {
    store.commit(`${ns}/resetInstallForm`);
  }
});

defineOptions({ name: 'ClDependencyConfigDialog' });
</script>

<template>
  <cl-dialog
    :title="t('common.actions.configure')"
    :visible="visible"
    width="800px"
    :confirm-loading="confirmLoading"
    @confirm="onConfirm"
    @close="onClose"
  >
    <cl-form v-if="form" ref="formRef" :model="form" label-width="180px">
      <cl-form-item
        :span="4"
        :label="t('views.env.deps.config.form.name')"
        prop="name"
      >
        <el-input v-model="form.name" disabled />
      </cl-form-item>
      <cl-form-item
        :span="4"
        :label="t('views.env.deps.config.form.execCmd')"
        prop="exec_cmd"
        required
      >
        <el-input
          v-model="form.exec_cmd"
          :placeholder="t('views.env.deps.config.form.execCmd')"
        />
      </cl-form-item>
      <cl-form-item
        :span="4"
        :label="t('views.env.deps.config.form.pkgCmd')"
        prop="pkg_cmd"
        required
      >
        <el-input
          v-model="form.pkg_cmd"
          :placeholder="t('views.env.deps.config.form.pkgCmd')"
        />
      </cl-form-item>
      <cl-form-item
        :span="4"
        :label="t('views.env.deps.config.form.pkgSrcURL')"
        prop="pkg_src_url"
      >
        <el-input
          v-model="form.pkg_src_url"
          :placeholder="t('views.env.deps.config.form.pkgSrcURL')"
        />
      </cl-form-item>
    </cl-form>
    <el-skeleton v-else />
  </cl-dialog>
</template>
