<script setup lang="ts">
import { useStore } from 'vuex';
import { ElMessage } from 'element-plus';
import { TASK_MODE_SELECTED_NODES } from '@/constants/task';
import { useSchedule, useNode, ClRemoteSelect } from '@/components';
import { priorityOptions, translate } from '@/utils';
import { ref } from 'vue';

const t = translate;

// store
const ns = 'schedule';
const store = useStore();

const {
  form,
  formRef,
  formRules,
  isSelectiveForm,
  isFormItemDisabled,
  modeOptions,
} = useSchedule(store);

// use node
const { activeNodesSorted: activeNodes } = useNode(store);

// on enabled change
const onEnabledChange = async (value: boolean) => {
  if (value) {
    await store.dispatch(`${ns}/enable`, form.value._id);
    ElMessage.success(t('components.schedule.message.success.enable'));
  } else {
    await store.dispatch(`${ns}/disable`, form.value._id);
    ElMessage.success(t('components.schedule.message.success.disable'));
  }
  await store.dispatch(`${ns}/getList`);
};

const spiderRef = ref<typeof ClRemoteSelect>();
const onSpiderChange = (spiderId: string) => {
  if (!spiderId) return;
  const payload = { ...form.value } as Schedule;
  if (!spiderRef.value) return;
  const spider: Spider = spiderRef.value.getSelectedItem();
  if (!spider) return;
  if (spider.cmd) payload.cmd = spider.cmd;
  if (spider.param) payload.param = spider.param;
  if (spider.mode) payload.mode = spider.mode;
  if (spider.node_ids?.length) payload.node_ids = spider.node_ids;
  if (spider.node_tags?.length) payload.node_tags = spider.node_tags;
  store.commit(`${ns}/setForm`, payload);
};

defineOptions({ name: 'ClScheduleForm' });
</script>

<template>
  <cl-form
    v-if="form"
    ref="formRef"
    :model="form"
    :rules="formRules"
    :selective="isSelectiveForm"
    class="schedule-form"
  >
    <!-- Row -->
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.name')"
      prop="name"
      required
    >
      <el-input
        v-model="form.name"
        :disabled="isFormItemDisabled('name')"
        :placeholder="t('components.schedule.form.name')"
      />
    </cl-form-item>
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.spider')"
      prop="spider_id"
      required
    >
      <cl-remote-select
        ref="spiderRef"
        v-model="form.spider_id"
        endpoint="/spiders"
        @change="onSpiderChange"
      />
    </cl-form-item>
    <!-- ./Row -->

    <!-- Row -->
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.cron')"
      prop="cron"
      required
    >
      <el-input
        v-model="form.cron"
        :disabled="isFormItemDisabled('cron')"
        :placeholder="t('components.schedule.form.cron')"
      />
    </cl-form-item>
    <cl-form-item
      :not-editable="isSelectiveForm"
      :span="2"
      :label="t('components.schedule.form.cronInfo')"
    >
      <div class="nav-btn">
        <cl-schedule-cron :cron="form.cron" icon-only />
      </div>
    </cl-form-item>
    <!-- ./Row -->

    <!-- Row -->
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.command')"
      prop="cmd"
    >
      <el-input
        v-model="form.cmd"
        :disabled="isFormItemDisabled('cmd')"
        :placeholder="t('components.schedule.form.command')"
      />
    </cl-form-item>
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.param')"
      prop="param"
    >
      <el-input
        v-model="form.param"
        :disabled="isFormItemDisabled('param')"
        :placeholder="t('components.schedule.form.param')"
      />
    </cl-form-item>
    <!-- ./Row -->

    <!-- Row -->
    <cl-form-item
      :span="2"
      :offset="2"
      :label="t('components.spider.form.priority')"
      prop="priority"
    >
      <el-select
        v-model="form.priority"
        :placeholder="t('components.spider.form.priority')"
        :disabled="isFormItemDisabled('priority')"
        id="priority"
        class-name="priority"
      >
        <el-option
          v-for="op in priorityOptions"
          :key="op.value"
          :label="op.label"
          :value="op.value"
        />
      </el-select>
    </cl-form-item>
    <!-- ./Row -->

    <!-- Row -->
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.defaultMode')"
      prop="mode"
    >
      <el-select v-model="form.mode" :disabled="isFormItemDisabled('mode')">
        <el-option
          v-for="op in modeOptions"
          :key="op.value"
          :label="op.label"
          :value="op.value"
        />
      </el-select>
    </cl-form-item>
    <cl-form-item
      :span="2"
      :label="t('components.schedule.form.enabled')"
      prop="enabled"
      required
    >
      <cl-switch v-model="form.enabled" @change="onEnabledChange" />
    </cl-form-item>
    <!-- ./Row -->

    <cl-form-item
      v-if="[TASK_MODE_SELECTED_NODES].includes(form.mode)"
      :span="4"
      :label="t('components.schedule.form.selectedNodes')"
      required
    >
      <el-select
        v-model="form.node_ids"
        multiple
        :placeholder="t('components.schedule.form.selectedNodes')"
      >
        <el-option
          v-for="n in activeNodes"
          :key="n.key"
          :value="n._id"
          :label="n.name"
        >
          <span style="margin-right: 5px">
            <cl-node-tag :node="n" icon-only />
          </span>
          <span>{{ n.name }}</span>
        </el-option>
      </el-select>
    </cl-form-item>

    <!-- Row -->
    <cl-form-item
      :span="4"
      :label="t('components.schedule.form.description')"
      prop="description"
    >
      <el-input
        v-model="form.description"
        :disabled="isFormItemDisabled('description')"
        :placeholder="t('components.schedule.form.description')"
        type="textarea"
      />
    </cl-form-item>
    <!-- ./Row -->
  </cl-form>
</template>

<style scoped></style>
