<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { ElCard, ElAlert, ElButton, ElLoading } from 'element-plus';
import { ClIcon, ClTag } from '@/components';
import { useDatabaseOrmService } from '@/services/database/databaseService';
import { translate } from '@/utils';
import { getStore } from '@/store';
import useDatabaseDetail from '@/views/database/detail/useDatabaseDetail';

// i18n
const t = translate;

// store
const store = getStore();

const { activeId } = useDatabaseDetail();

// ORM service
const {
  getOrmCompatibility,
  getOrmStatus,
  setOrmStatus,
  isOrmSupported
} = useDatabaseOrmService();

// reactive state
const loading = ref(false);
const compatibility = ref<DatabaseOrmCompatibility | null>(null);
const ormStatus = ref<DatabaseOrmStatus | null>(null);
const database = computed(() => store.getters['database/form']);

// methods
const loadOrmInfo = async () => {
  if (!activeId.value) return;

  loading.value = true;
  try {
    const [compatibilityRes, statusRes] = await Promise.all([
      getOrmCompatibility(activeId.value),
      getOrmStatus(activeId.value)
    ]);
    compatibility.value = compatibilityRes;
    ormStatus.value = statusRes;

    // Debug logging to understand the issue
    console.log('Database ORM Compatibility Debug:', {
      databaseId: activeId.value,
      compatibility: compatibilityRes,
      status: statusRes,
      database: database.value,
      dataSource: database.value?.data_source
    });
  } catch (error) {
    console.error('Failed to load ORM info:', error);
  } finally {
    loading.value = false;
  }
};

const handleToggleOrm = async () => {
  if (!activeId.value || !ormStatus.value) return;

  loading.value = true;
  try {
    const newValue = !ormStatus.value.enabled;
    await setOrmStatus(activeId.value, newValue);

    // Update local state
    ormStatus.value.enabled = newValue;

    // Update form in store
    store.commit('database/setForm', {
      ...database.value,
      use_orm: newValue,
    });

    // Show success message
    // You might want to add a success notification here

  } catch (error) {
    console.error('Failed to toggle ORM:', error);
    // You might want to add an error notification here
  } finally {
    loading.value = false;
  }
};

// lifecycle
onMounted(() => {
  loadOrmInfo();
});

defineOptions({ name: 'ClDatabaseOrmSettings' });
</script>

<template>
  <div class="database-orm-settings">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <cl-icon icon="fa-bolt" />
          <span>{{ t('components.database.form.ormMode') }}</span>
        </div>
      </template>

      <div v-if="compatibility?.should_show_toggle" class="orm-settings-content">
        <div class="orm-status-section">
          <div class="status-row">
            <span class="label">{{ t('components.database.form.status') }}:</span>
            <cl-tag
              v-if="ormStatus?.enabled"
              type="success"
              :label="t('components.database.orm.enabled')"
              :icon="['fa', 'bolt']"
            />
            <cl-tag
              v-else
              type="warning"
              :label="t('components.database.orm.disabled')"
              :icon="['fa', 'wrench']"
            />
          </div>

          <div class="action-row">
            <el-button
              :type="ormStatus?.enabled ? 'warning' : 'success'"
              :icon="ormStatus?.enabled ? 'fa-wrench' : 'fa-bolt'"
              @click="handleToggleOrm"
              :loading="loading"
            >
              {{ ormStatus?.enabled
                  ? t('components.database.orm.switchToLegacy')
                  : t('components.database.orm.switchToOrm')
              }}
            </el-button>
          </div>
        </div>

        <div v-if="ormStatus?.enabled" class="orm-benefits">
          <h4>{{ t('components.database.orm.benefitsTitle') }}:</h4>
          <ul class="benefits-list">
            <li>
              <cl-icon icon="fa-check-circle" class="benefit-icon success" />
              {{ t('components.database.orm.benefit1') }}
            </li>
            <li>
              <cl-icon icon="fa-check-circle" class="benefit-icon success" />
              {{ t('components.database.orm.benefit2') }}
            </li>
            <li>
              <cl-icon icon="fa-check-circle" class="benefit-icon success" />
              {{ t('components.database.orm.benefit3') }}
            </li>
            <li>
              <cl-icon icon="fa-check-circle" class="benefit-icon success" />
              {{ t('components.database.orm.benefit4') }}
            </li>
          </ul>
        </div>

        <div class="migration-info">
          <el-alert
            type="info"
            :closable="false"
            show-icon
          >
            <template #title>
              {{ t('components.database.orm.migrationTitle') }}
            </template>
            <p>{{ t('components.database.orm.migrationMessage') }}</p>
          </el-alert>
        </div>
      </div>

      <div v-else class="orm-not-supported">
        <el-alert
          type="info"
          :closable="false"
          show-icon
        >
          <template #title>
            {{ t('components.database.orm.notSupportedTitle') }}
          </template>
          <p>
            {{ t('components.database.orm.notSupportedMessage') }}
          </p>
          <div v-if="database?.data_source" class="debug-info">
            <p><strong>Current database type:</strong> {{ database.data_source }}</p>
            <p><strong>Supported types:</strong> MySQL, PostgreSQL, SQL Server</p>
          </div>
        </el-alert>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.database-orm-settings {
  margin: 16px 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.orm-settings-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.orm-status-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.status-row,
.action-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.label {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.orm-benefits h4 {
  margin: 0 0 12px 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.benefits-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.benefits-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.benefit-icon.success {
  color: var(--cl-success-color);
}

.migration-info,
.orm-not-supported {
  margin-top: 16px;
}

.debug-info {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.debug-info p {
  margin: 4px 0;
}
</style>
