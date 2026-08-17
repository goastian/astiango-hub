<script setup lang="ts">
import { computed, onBeforeMount, ref, watch } from 'vue';
import { useStore } from 'vuex';
import { useAutoProbe } from '@/components';
import { getViewPortOptions, translate } from '@/utils';

// i18n
const t = translate;

// store
const store = useStore();
const { form, formRef, formRules, isSelectiveForm, isFormItemDisabled } =
  useAutoProbe(store);

const isCreate = computed(() => !!form.value?._id);
const isNameModified = ref(false);

const viewportOptions = computed<ViewPortSelectOption[]>(() => {
  return getViewPortOptions();
});
const viewportValue = ref<ViewPortValue>('pc-normal');
const onViewportChange = (value: ViewPortValue) => {
  if (!form.value) return;
  const selectedOption = viewportOptions.value.find(
    option => option.value === value
  );
  if (selectedOption) {
    form.value.viewport = selectedOption.viewport;
  }
};
const updateViewPortValue = () => {
  const selectedOption = viewportOptions.value.find(
    op =>
      op.viewport.width === form.value?.viewport?.width &&
      op.viewport.height === form.value?.viewport?.height
  );
  if (selectedOption) {
    viewportValue.value = selectedOption.value!;
  }
};
watch(() => JSON.stringify(form.value?.viewport), updateViewPortValue);
onBeforeMount(updateViewPortValue);

// Auto naming handling
const onNameChange = (_: string) => {
  isNameModified.value = true;
};
const getNameByURL = (url: string) => {
  if (!url) return '';
  const urlObj = new URL(url);
  const pathname = urlObj.pathname.replace(/\/$/, '');
  const segments = pathname.split('/');
  return segments[segments.length - 1] || 'New AutoProbe';
};
const onURLChange = (url: string) => {
  if (!form.value) return;
  if (!isNameModified.value) {
    form.value.name = getNameByURL(url);
  }
};

defineOptions({ name: 'ClAutoProbeForm' });
</script>

<template>
  <cl-form
    v-if="form"
    ref="formRef"
    :model="form"
    :rules="formRules"
    :selective="isSelectiveForm"
  >
    <cl-form-item
      :span="2"
      :offset="2"
      :label="t('components.project.form.name')"
      not-editable
      prop="name"
      required
    >
      <el-input
        v-model="form.name"
        :disabled="isFormItemDisabled('name')"
        :placeholder="t('components.autoprobe.form.name')"
      />
    </cl-form-item>
    <cl-form-item
      :span="4"
      :label="t('components.autoprobe.form.url')"
      prop="url"
      required
    >
      <el-input
        v-model="form.url"
        :disabled="isFormItemDisabled('url')"
        :placeholder="t('components.autoprobe.form.url')"
        @input="onURLChange"
      >
        <template #prefix>
          <cl-icon :icon="['fa', 'at']" />
        </template>
      </el-input>
    </cl-form-item>
    <cl-form-item
      :span="4"
      :label="t('components.autoprobe.form.query')"
      prop="query"
    >
      <el-input
        v-model="form.query"
        :disabled="isFormItemDisabled('query')"
        :placeholder="t('components.autoprobe.form.queryPlaceholder')"
        type="textarea"
      />
    </cl-form-item>
    <cl-form-item
      :span="2"
      :label="t('components.autoprobe.form.viewport')"
      prop="viewport"
    >
      <el-select
        v-model="viewportValue"
        :disabled="isFormItemDisabled('viewport')"
        :placeholder="t('components.autoprobe.form.viewport')"
        @change="onViewportChange"
      >
        <el-option
          v-for="op in viewportOptions"
          :key="op.value"
          :label="op.label"
          :value="op.value"
        />
      </el-select>
      <cl-tag
        v-if="form.viewport"
        size="large"
        :icon="['fa', 'desktop']"
        :label="`${form.viewport?.width}x${form.viewport?.height}`"
      >
        <template #tooltip>
          <div>
            <label>{{ t('components.autoprobe.form.viewportWidth') }}: </label>
            <span
              >{{ form.viewport?.width }}
              {{ t('components.autoprobe.form.viewportPx') }}</span
            >
          </div>
          <div>
            <label>{{ t('components.autoprobe.form.viewportHeight') }}: </label>
            <span
              >{{ form.viewport?.height }}
              {{ t('components.autoprobe.form.viewportPx') }}</span
            >
          </div>
        </template>
      </cl-tag>
    </cl-form-item>
    <cl-form-item
      :span="2"
      :label="t('components.autoprobe.form.runOnCreate')"
      prop="query"
    >
      <cl-switch
        v-model="form.run_on_create"
        :disabled="isFormItemDisabled('run_on_create')"
        :placeholder="t('components.autoprobe.form.runOnCreate')"
      />
    </cl-form-item>
  </cl-form>
</template>
