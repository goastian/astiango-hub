<script setup lang="ts">
import { computed, onBeforeMount, onBeforeUnmount, ref, watch } from 'vue';
import { useStore } from 'vuex';
import {
  alertTemplates,
  allTemplates,
  getIconByAction,
  translate,
} from '@/utils';
import useNotificationSetting from '@/components/core/notification/setting/useNotificationSetting';
import { ElMessage } from 'element-plus';
import { ACTION_ADD } from '@/constants';

defineProps<{
  readonly?: boolean;
}>();

// i18n
const t = translate;

// store
const ns: ListStoreNamespace = 'notificationSetting';
const store = useStore();
const {
  notificationAlert: notificationAlertState,
  notificationChannel: notificationChannelState,
} = store.state as RootStoreState;

const { form, formRef, isSelectiveForm, activeDialogKey } =
  useNotificationSetting(store);

const onTemplateChange = () => {
  const { template_key } = form.value;
  const template = allTemplates.find(t => t.key === template_key);
  if (!template) return;
  const { name, description, title, template_markdown, template_rich_text } =
    template;
  store.commit(`${ns}/setForm`, {
    ...form.value,
    ...template,
    name: t(name as string),
    description: t(description as string),
    title: t(title as string),
    template_markdown,
    template_rich_text,
  });

  // handle alert template
  if (template.key.startsWith('alert')) {
    onCreateAlertClick();
  }
};

const createChannelVisible = ref(false);
const channelFormRef = ref();
const allChannels = computed<NotificationChannel[]>(
  () => notificationChannelState.allChannels
);
const updateChannelIds = async () => {
  if (activeDialogKey.value === 'create') {
    // get all channels
    await store.dispatch('notificationChannel/getAllChannels');

    // enable all channels
    store.commit(`${ns}/setForm`, {
      ...form.value,
      channel_ids: allChannels.value.map(channel => channel._id),
    });
  } else if (!activeDialogKey.value) {
    store.commit('notificationChannel/resetAllChannels');
  }
};
watch(activeDialogKey, updateChannelIds);
onBeforeMount(updateChannelIds);
onBeforeUnmount(() => store.commit('notificationChannel/resetAllChannels'));
const onCreateChannelConfirm = async () => {
  // validate channel form
  await channelFormRef.value?.validateForm();

  // create channel
  const { data: newChannel } = await store.dispatch(
    'notificationChannel/create',
    notificationChannelState.form
  );
  ElMessage.success(t('views.notification.message.success.create.channel'));

  // get all channels again
  await store.dispatch('notificationChannel/getAllChannels');

  // set channel ids
  store.commit(`${ns}/setForm`, {
    ...form.value,
    channel_ids: [...(form.value.channel_ids || []), newChannel._id],
  });

  // close channel form create dialog
  createChannelVisible.value = false;
};

const createAlertVisible = ref(false);
const alertFormRef = ref();
const onCreateAlertClick = () => {
  // find existing alert
  let alertForm = notificationAlertState.allAlerts.find(
    a => a.name === form.value.name
  );

  // create new alert given a template key
  if (!alertForm) {
    if (form.value.template_key) {
      // find alert template
      alertForm = alertTemplates.find(
        t => t.key === form.value.template_key
      ) as NotificationAlert;

      // handle alert template
      if (alertForm) {
        alertForm = {
          ...alertForm,
          name: t(alertForm.name as string),
          description: t(alertForm.description as string),
          enabled: true,
          template_key: form.value.template_key,
        };
      }
    }

    // create new alert form if template not found
    if (!alertForm) alertForm = notificationAlertState.newFormFn();

    // set alert form
    store.commit('notificationAlert/setForm', { ...alertForm });

    // open alert form create dialog
    createAlertVisible.value = true;
  } else {
    // set alert id if alert form exists
    store.commit(`${ns}/setForm`, {
      ...form.value,
      alert_id: alertForm._id,
    });
  }
};
const onCreateAlertConfirm = async () => {
  // validate alert form
  await alertFormRef.value?.validateForm();

  // create alert
  const { data: newAlert } = await store.dispatch(
    'notificationAlert/create',
    notificationAlertState.form
  );
  ElMessage.success(t('views.notification.message.success.create.alert'));

  // set alert id
  store.commit(`${ns}/setForm`, {
    ...form.value,
    alert_id: newAlert._id,
  });

  // close alert form create dialog
  createAlertVisible.value = false;
};

const formDisabled = computed<boolean>(() => {
  if (activeDialogKey.value !== 'create') {
    return false;
  }
  return !form.value.use_custom_setting;
});

const formRequired = computed<boolean>(() => {
  if (activeDialogKey.value !== 'create') {
    return true;
  }
  return !!form.value.use_custom_setting;
});

defineOptions({ name: 'ClNotificationSettingForm' });
</script>

<template>
  <cl-form v-if="form" ref="formRef" :model="form" :selective="isSelectiveForm">
    <template v-if="activeDialogKey === 'create'">
      <cl-form-item
        :span="2"
        :label="t('views.notification.settings.templates.label')"
        prop="template_key"
        :required="!form.use_custom_setting"
      >
        <el-select
          v-model="form.template_key"
          @change="onTemplateChange"
          clearable
        >
          <el-option
            v-for="op in allTemplates"
            :key="op.key"
            :value="op.key"
            :label="
              t(`components.notification.setting.templates.${op.key}.label`)
            "
          />
        </el-select>
      </cl-form-item>
      <cl-form-item :span="2" no-label>
        <el-checkbox v-model="form.use_custom_setting">
          {{ t('views.notification.settings.form.useCustomSetting.label') }}
        </el-checkbox>
        <cl-tip
          :tooltip="
            t('views.notification.settings.form.useCustomSetting.tooltip')
          "
        />
      </cl-form-item>
    </template>

    <cl-form-item
      :span="2"
      :label="t('views.notification.settings.form.name')"
      prop="name"
      :required="formRequired"
    >
      <el-input
        v-model="form.name"
        :placeholder="t('views.notification.settings.form.name')"
        :disabled="formDisabled"
      />
    </cl-form-item>
    <cl-form-item
      :span="2"
      :label="t('views.notification.settings.form.enabled')"
      prop="enabled"
    >
      <cl-switch v-model="form.enabled" />
    </cl-form-item>

    <cl-form-item
      :span="4"
      :label="t('views.notification.settings.form.description')"
      prop="description"
    >
      <el-input
        v-model="form.description"
        type="textarea"
        :placeholder="t('views.notification.settings.form.description')"
        :disabled="formDisabled"
      />
    </cl-form-item>

    <cl-form-item
      v-if="activeDialogKey === 'create'"
      :span="2"
      :offset="form.trigger === 'alert' ? 0 : 2"
      :label="t('views.notification.settings.form.trigger')"
      prop="trigger"
      :required="formRequired"
    >
      <cl-notification-setting-trigger-select
        v-model="form.trigger"
        :disabled="formDisabled"
      />
    </cl-form-item>
    <cl-form-item
      v-if="form.trigger === 'alert'"
      :span="2"
      :label="t('views.notification.settings.form.alert')"
      prop="alert_id"
      required
    >
      <cl-remote-select
        v-model="form.alert_id"
        endpoint="/notifications/alerts"
      />
      <cl-fa-icon-button
        type="default"
        size="default"
        :icon="getIconByAction(ACTION_ADD)"
        :tooltip="t('views.notification.settings.actions.createAlert')"
        @click="onCreateAlertClick"
      />
    </cl-form-item>

    <cl-form-item
      v-if="activeDialogKey === 'create'"
      :span="4"
      :label="t('views.notification.settings.form.channels')"
      prop="channel_ids"
      :required="formRequired"
    >
      <el-checkbox-group v-model="form.channel_ids">
        <el-space spacer="10px" wrap>
          <el-checkbox
            v-for="channel in allChannels"
            :key="channel._id"
            :value="channel._id"
          >
            {{ channel.name }}
          </el-checkbox>
        </el-space>
      </el-checkbox-group>
      <cl-fa-icon-button
        type="default"
        size="small"
        :icon="getIconByAction(ACTION_ADD)"
        :tooltip="t('views.notification.channels.navActions.new.tooltip')"
        @click="createChannelVisible = true"
      />
    </cl-form-item>
  </cl-form>

  <el-drawer
    v-model="createChannelVisible"
    :title="t('views.notification.settings.actions.createChannel')"
    size="960px"
  >
    <cl-notification-channel-form ref="channelFormRef" />
    <template #footer>
      <el-button plain @click="createChannelVisible = false">
        {{ t('common.actions.cancel') }}
      </el-button>
      <el-button type="primary" @click="onCreateChannelConfirm">
        {{ t('common.actions.confirm') }}
      </el-button>
    </template>
  </el-drawer>

  <el-drawer
    v-model="createAlertVisible"
    :title="t('views.notification.settings.actions.createAlert')"
    size="960px"
  >
    <cl-notification-alert-form ref="alertFormRef" />
    <template #footer>
      <el-button plain @click="createAlertVisible = false">
        {{ t('common.actions.cancel') }}
      </el-button>
      <el-button type="primary" @click="onCreateAlertConfirm">
        {{ t('common.actions.confirm') }}
      </el-button>
    </template>
  </el-drawer>
</template>

<style scoped>
.alert-wrapper,
.alert-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
