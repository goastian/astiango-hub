<script setup lang="ts">
import { computed, onBeforeMount, ref, watch } from 'vue';
import useRequest from '@/services/request';
import { Placement } from '@popperjs/core';
import { FILTER_OP_CONTAINS } from '@/constants';

const props = withDefaults(
  defineProps<{
    modelValue?: string;
    placeholder?: string;
    disabled?: boolean;
    size?: BasicSize;
    placement?: Placement;
    filterable?: boolean;
    clearable?: boolean;
    remoteShowSuffix?: boolean;
    endpoint: string;
    labelKey?: string;
    valueKey?: string;
    limit?: number;
    emptyOption?: SelectOption;
  }>(),
  {
    filterable: true,
    remoteShowSuffix: true,
    labelKey: 'name',
    valueKey: '_id',
    limit: 100,
  }
);

const emit = defineEmits<{
  (e: 'change', value: string): void;
  (e: 'select', value: string): void;
  (e: 'clear'): void;
  (e: 'update:model-value', value: string): void;
}>();

const { get } = useRequest();

const internalValue = ref<string | undefined>(props.modelValue);
watch(
  () => props.modelValue,
  () => {
    internalValue.value = props.modelValue;
  }
);
watch(internalValue, () =>
  emit('update:model-value', internalValue.value || '')
);

const loading = ref(false);
const list = ref<any[]>([]);
const remoteMethod = async (query?: string) => {
  const { endpoint, labelKey, limit } = props;
  try {
    loading.value = true;
    let filter: string | undefined = undefined;
    if (query) {
      filter = JSON.stringify([
        {
          key: labelKey,
          op: FILTER_OP_CONTAINS,
          value: query,
        } as FilterConditionData,
      ]);
    }
    const sort = labelKey;
    const res = await get(endpoint, { filter, size: limit, sort });
    list.value = res.data || [];
  } catch (e) {
    console.error(e);
  } finally {
    loading.value = false;
  }
};
const selectOptions = computed<SelectOption[]>(() => {
  const { emptyOption, labelKey, valueKey } = props;
  const options: SelectOption[] = list.value.map(row => ({
    label: row[labelKey],
    value: row[valueKey],
  }));
  if (emptyOption) {
    const { label, value } = emptyOption;
    options.unshift({
      label,
      value,
    });
  }
  return options;
});
onBeforeMount(remoteMethod);

const selectedItem = computed(() => {
  const { valueKey } = props;
  return list.value.find(item => item[valueKey] === internalValue.value);
});

const getSelectedItem = () => {
  return selectedItem.value;
};

defineExpose({
  getSelectedItem,
});

defineOptions({ name: 'ClRemoteSelect' });
</script>

<template>
  <el-select
    v-model="internalValue"
    :key="JSON.stringify(selectOptions)"
    :size="size"
    :placeholder="placeholder"
    :filterable="filterable"
    :disabled="disabled"
    :clearable="clearable"
    :placement="placement"
    remote
    :remote-method="remoteMethod"
    :remote-show-suffix="remoteShowSuffix"
    @change="(value: any) => emit('change', value)"
    @clear="() => emit('clear')"
  >
    <el-option
      v-for="(op, index) in selectOptions"
      :key="index"
      :label="op.label"
      :value="op.value"
    />
    <template #label>
      <slot name="label" />
    </template>
  </el-select>
</template>
