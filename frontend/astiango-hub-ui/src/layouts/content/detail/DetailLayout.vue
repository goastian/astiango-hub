<script setup lang="ts">
import {
  computed,
  onBeforeMount,
  onBeforeUnmount,
  onMounted,
  watch,
} from 'vue';
import { useStore } from 'vuex';
import { useDetail } from '@/layouts';

const props = withDefaults(
  defineProps<{
    storeNamespace: ListStoreNamespace;
    navItemNameKey?: string;
    showBackButton?: boolean;
    showSaveButton?: boolean;
    navItemLabelFn?: (item: NavItem) => string;
  }>(),
  {
    navItemNameKey: 'name',
    showSaveButton: true,
    showBackButton: true,
  }
);

const ns = computed(() => props.storeNamespace);
const store = useStore();

const {
  activeId,
  activeTabName,
  getForm,
  navLoading,
  navItems,
  onNavSelect,
  onNavTabsSelect,
  onBack,
  onSave,
  tabs,
} = useDetail(props.storeNamespace);

const computedTabs = computed<NavItem[]>(() =>
  tabs.value.map((tab: NavItem) => ({ ...tab }))
);

// Fetch the form data when the component is mounted
onBeforeMount(getForm);

// Watch for changes in the activeId and fetch the form data accordingly
watch(
  () => activeId.value,
  async () => {
    if (!activeId.value) return;
    await getForm();
  }
);

// Fetch navigation list before mounting the component
const getNavList = async (query?: string) => {
  await store.dispatch(`${ns.value}/getNavList`, query);
};
onBeforeMount(getNavList);

// reset form before unmount
onBeforeUnmount(() => {
  if (!activeTabName.value) {
    store.commit(`${ns.value}/resetForm`);
    store.commit(`${ns.value}/resetNavList`);
  }
});

defineOptions({ name: 'ClDetailLayout' });
</script>

<template>
  <div class="detail-layout">
    <div v-loading="navLoading" class="content">
      <cl-nav-tabs
        :active-key="activeTabName"
        :items="computedTabs"
        class="nav-tabs"
        @select="onNavTabsSelect"
      >
        <template #extra>
          <div class="nav-select">
            <cl-icon :icon="['fa', 'exchange-alt']" size="small" />
            <el-select
              :model-value="activeId"
              size="small"
              placement="bottom-end"
              filterable
              remote
              remote-show-suffix
              :remote-method="getNavList"
              @change="onNavSelect"
            >
              <el-option
                v-for="item in navItems"
                :key="item.id"
                :label="navItemLabelFn ? navItemLabelFn(item) : item.label"
                :value="item.id"
              />
              <template #label>
                {{
                  navItems.find(item => item.id === activeId)?.label || activeId
                }}
              </template>
            </el-select>
          </div>
        </template>
      </cl-nav-tabs>
      <cl-nav-actions class="nav-actions">
        <cl-nav-action-group-detail-common
          :show-back-button="showBackButton"
          :show-save-button="showSaveButton"
          @back="onBack"
          @save="onSave"
        />
        <slot name="actions" />
      </cl-nav-actions>
      <div class="content-container">
        <router-view />
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail-layout {
  display: flex;
  height: 100%;

  .content {
    flex: 1;
    width: 100%;
    height: 100%;
    background-color: var(--cl-container-white-bg);
    display: flex;
    flex-direction: column;
    overflow: hidden;

    .nav-actions {
      height: 53px;
      flex: 0 0 53px;
    }

    .nav-select {
      width: 180px;
      margin-right: 10px;
      display: flex;
      align-items: center;
      gap: 5px;
      color: var(--cl-info-medium-color);

      .el-select {
        flex: 1;
      }
    }

    .content-container {
      flex: 1;
      height: calc(100% - 41px - 53px);
    }
  }
}
</style>
