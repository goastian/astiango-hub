<script setup lang="ts">
import { useStore } from 'vuex';
import { useRouter } from 'vue-router';
import {
  hasNotificationSettingChannelWarningMissingMailConfigFields,
  translate,
} from '@/utils';
import { useNotificationChannel, useNotificationSetting } from '@/components';
import { computed, onBeforeMount, onBeforeUnmount, ref, watch } from 'vue';
import { useNotificationSettingDetail } from '@/views';

const t = translate;

const store = useStore();
const { notificationChannel: notificationChannelState } =
  store.state as RootStoreState;

const router = useRouter();

const { form } = useNotificationSetting(store);

const { activeId } = useNotificationSettingDetail();

const allChannels = computed<NotificationChannel[]>(
  () => notificationChannelState.allChannels
);
const allChannelsDict = computed<Map<string, NotificationChannel>>(() => {
  return new Map(allChannels.value.map(channel => [channel._id!, channel]));
});
onBeforeMount(() => store.dispatch('notificationChannel/getAllChannels'));
onBeforeUnmount(() => store.commit('notificationChannel/resetAllChannels'));

const selectAll = ref(false);
const selectIntermediate = ref(false);
const updateSelectAll = () => {
  if (!allChannels.value?.length) return;

  // check if all options are selected
  selectAll.value = allChannels.value.every(channel =>
    form.value.channel_ids?.includes(channel._id!)
  );

  // check if some options are selected
  if (!selectAll.value) {
    selectIntermediate.value = allChannels.value.some(channel =>
      form.value.channel_ids?.includes(channel._id!)
    );
  } else {
    selectIntermediate.value = false;
  }
};
watch(allChannels, updateSelectAll);
watch(() => form.value.channel_ids, updateSelectAll);
watch(activeId, updateSelectAll);
const onSelectAll = () => {
  if (selectAll.value) {
    form.value.channel_ids = allChannels.value.map(channel => channel._id!);
  } else {
    form.value.channel_ids = [];
  }
  selectIntermediate.value = false;
};

const onChannelNavigate = async (channelId: string) => {
  await router.push(`/notifications/channels/${channelId}`);
};

const hasWarningMissingMailConfigFields = computed(() => {
  return hasNotificationSettingChannelWarningMissingMailConfigFields(
    form.value,
    allChannelsDict.value
  );
});

const hasWarningEmptyChannel = computed(() => {
  return !form.value.channel_ids?.length;
});

defineOptions({ name: 'ClNotificationSettingDetailTabChannels' });
</script>

<template>
  <div class="notification-setting-detail-tab-channels">
    <cl-form>
      <cl-form-item :span="4" :label="t('common.actions.selectAll')">
        <el-checkbox
          v-model="selectAll"
          :indeterminate="selectIntermediate"
          @change="onSelectAll"
        />
      </cl-form-item>
      <cl-form-item
        :span="4"
        :label="t('components.notification.channel.label')"
      >
        <el-checkbox-group v-model="form.channel_ids">
          <el-space spacer="10px" wrap>
            <div
              v-for="channel in allChannels"
              :key="channel._id!"
              style="display: flex; align-items: center"
            >
              <el-checkbox :label="channel.name" :value="channel._id">
                {{ channel.name }}
              </el-checkbox>
              <cl-icon
                :icon="['fa', 'external-link-alt']"
                @click="onChannelNavigate(channel._id!)"
              />
            </div>
          </el-space>
        </el-checkbox-group>
      </cl-form-item>

      <cl-form-item :span="4">
        <el-alert
          v-if="hasWarningMissingMailConfigFields"
          type="warning"
          :closable="false"
          show-icon
        >
          <div style="line-height: 24px">
            {{
              t(
                'views.notification.settings.warnings.missingMailConfigFields.content'
              )
            }}
          </div>
          <cl-nav-link
            :icon="['fa', 'external-link-alt']"
            :path="`/notifications/settings/${activeId}/mail`"
            :label="
              t(
                'views.notification.settings.warnings.missingMailConfigFields.action'
              )
            "
          />
        </el-alert>
        <el-alert
          v-else-if="hasWarningEmptyChannel"
          type="warning"
          :closable="false"
          show-icon
        >
          <div style="line-height: 24px">
            {{ t('views.notification.settings.warnings.emptyChannel.content') }}
          </div>
        </el-alert>
        <el-alert v-else type="success" :closable="false" show-icon>
          {{ t('views.notification.settings.warnings.noWarning.content') }}
        </el-alert>
      </cl-form-item>
    </cl-form>
  </div>
</template>

<style scoped>
.notification-setting-detail-tab-channels {
  margin: 20px;

  &:deep(.icon) {
    color: var(--cl-info-color);
    margin-left: 5px;
    cursor: pointer;
    height: 14px;
    width: 14px;

    &:hover {
      opacity: 0.8;
    }
  }

  &:deep(.is-checked + .icon) {
    color: var(--cl-primary-color);
  }

  &:deep(.el-alert) {
    width: 100%;

    .icon {
      color: var(--cl-primary-color);
    }
  }
}
</style>
