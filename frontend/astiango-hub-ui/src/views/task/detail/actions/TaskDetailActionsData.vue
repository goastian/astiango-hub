<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useStore } from 'vuex';
import useSpider from '@/components/core/spider/useSpider';
import useRequest from '@/services/request';
import { FILTER_OP_EQUAL } from '@/constants';
import { translate } from '@/utils';

const { get } = useRequest();

// i18n
const t = translate;

// store
const ns = 'task';
const store = useStore();
const { task: state } = store.state as RootStoreState;

// spider
const spider = computed(() => state.form.spider);

// spider collection name
const colName = ref<string>();
watch(
  () => spider.value,
  async () => {
    if (!spider.value) return;
    const res = await get(`/spiders/${spider.value._id}`);
    colName.value = (res.data as Spider)?.col_name;
  }
);

// display all fields
const displayAllFields = ref<boolean>(state.dataDisplayAllFields);
const onDisplayAllFieldsChange = (val: boolean) => {
  store.commit(`${ns}/setDataDisplayAllFields`, val);
};

defineOptions({ name: 'ClTaskDetailActionsData' });
</script>

<template>
  <cl-nav-action-group class="task-detail-actions-data">
    <cl-nav-action-fa-icon
      :icon="['fa', 'database']"
      :tooltip="t('components.task.actions.data.tooltip.dataActions')"
    />
    <cl-nav-action-item>
      <el-tooltip
        :content="t('components.task.actions.data.tooltip.displayAllFields')"
      >
        <div class="display-all-fields">
          <cl-switch
            :active-icon="['fa', 'eye']"
            :inactive-icon="['fa', 'eye']"
            inline-prompt
            v-model="displayAllFields"
            @change="onDisplayAllFieldsChange"
          />
        </div>
      </el-tooltip>
    </cl-nav-action-item>
  </cl-nav-action-group>
</template>

<style scoped>
.task-detail-actions-data:deep(.display-all-fields) {
  margin-right: 10px;
}
</style>
