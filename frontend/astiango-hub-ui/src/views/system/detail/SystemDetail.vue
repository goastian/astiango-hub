<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  getIconByGeneralConcept,
  getIconByRouteConcept,
  translate,
} from '@/utils';
import { useStore } from 'vuex';
import { ElMessage } from 'element-plus';

const t = translate;

const router = useRouter();

const route = useRoute();

const ns = 'system';
const store = useStore();

const activeItemKey = computed(() => {
  return route.path.split('/').pop() || '';
});

const formRef = ref();

const onSave = async () => {
  await formRef.value?.validate();
  await store.dispatch(`${ns}/saveCustomize`);
  ElMessage.success(t('common.message.success.save'));
  await formRef.value?.save?.();
};

const menuItems = computed<NavItem[]>(() => [
  {
    id: 'customize',
    icon: getIconByGeneralConcept('customize'),
    label: t('views.system.menuItems.customize'),
  },
  // {
  //   id: 'models',
  //   icon: getIconByRouteConcept('models'),
  //   label: t('views.system.menuItems.models'),
  // },
  {
    id: 'dependency',
    icon: getIconByRouteConcept('dependency'),
    label: t('views.system.menuItems.dependency'),
  },
  {
    id: 'environment',
    icon: getIconByRouteConcept('environment'),
    label: t('views.system.menuItems.environment'),
  },
]);

defineOptions({ name: 'ClSystemDetail' });
</script>

<template>
  <div class="system-detail">
    <el-menu
      :default-active="activeItemKey"
      @select="(value: string) => router.push(`/system/${value}`)"
    >
      <el-menu-item v-for="item in menuItems" :key="item.id" :index="item.id">
        <cl-icon :icon="item.icon" />
        {{ item.label }}
      </el-menu-item>
    </el-menu>
    <div class="system-detail-content">
      <router-view />
    </div>
  </div>
</template>

<style scoped>
.system-detail {
  background-color: #ffffff;
  height: calc(100vh - var(--cl-header-height) - var(--cl-tabs-view-height));
  display: flex;
  overflow: hidden;

  &:deep(.form) {
    display: block;
    width: 100%;
  }

  &:deep(.el-menu) {
    flex: 0 0 180px;
    width: 180px;

    .el-menu-item {
      &:hover {
        background-color: inherit !important;
        color: var(--cl-primary-color);
      }

      &:deep(.icon) {
        width: 24px;
      }
    }
  }

  .system-detail-content {
    flex: 1;
    width: calc(100% - 180px);
  }
}
</style>
