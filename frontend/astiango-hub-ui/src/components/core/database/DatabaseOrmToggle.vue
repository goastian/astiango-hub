<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ElSwitch, ElTooltip } from 'element-plus';
import { ClIcon, ClTag } from '@/components';
import { useDatabaseOrmService } from '@/services/database/databaseService';
import { translate } from '@/utils';

interface Props {
  dataSource?: DatabaseDataSource;
  modelValue?: boolean;
  disabled?: boolean;
  showTooltip?: boolean;
  showLabel?: boolean;
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  disabled: false,
  showTooltip: true,
  showLabel: true,
});

const emit = defineEmits<Emits>();

// i18n
const t = translate;

// ORM service
const { isOrmSupported } = useDatabaseOrmService();

// computed
const isSupported = computed(() => isOrmSupported(props.dataSource));
const internalValue = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
});

// Don't show the toggle if ORM is not supported for this data source
const shouldShow = computed(() => isSupported.value);

defineOptions({ name: 'ClDatabaseOrmToggle' });
</script>

<template>
  <div v-if="shouldShow" class="database-orm-toggle">
    <div class="toggle-container">
      <div v-if="showLabel" class="label-container">
        <span class="form-label">{{ t('components.database.form.ormMode') }}</span>
        <el-tooltip
          v-if="showTooltip"
          :content="t('components.database.form.ormModeTooltip')"
          placement="top"
        >
          <cl-icon icon="fa-info-circle" class="info-icon" />
        </el-tooltip>
      </div>
      
      <div class="toggle-content">
        <el-switch
          v-model="internalValue"
          :disabled="disabled"
          :active-text="t('components.database.orm.enabled')"
          :inactive-text="t('components.database.orm.disabled')"
          size="default"
        />
        
        <div class="status-badge">
          <cl-tag
            v-if="internalValue"
            type="success"
            :label="t('components.database.orm.modern')"
            :icon="['fa', 'bolt']"
          />
          <cl-tag
            v-else
            type="warning"
            :label="t('components.database.orm.legacy')"
            :icon="['fa', 'wrench']"
          />
        </div>
      </div>
      
      <div class="help-text">
        <span v-if="internalValue" class="help-text-enabled">
          {{ t('components.database.orm.helpTextEnabled') }}
        </span>
        <span v-else class="help-text-disabled">
          {{ t('components.database.orm.helpTextDisabled') }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.database-orm-toggle {
  margin: 16px 0;
}

.toggle-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.label-container {
  display: flex;
  align-items: center;
  gap: 6px;
}

.form-label {
  font-weight: 500;
  color: var(--cl-text-color-primary);
}

.info-icon {
  color: var(--cl-info-color);
  font-size: 14px;
  cursor: help;
}

.toggle-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-badge {
  display: flex;
  align-items: center;
}

.help-text {
  font-size: 12px;
  line-height: 1.4;
}

.help-text-enabled {
  color: var(--cl-success-color);
}

.help-text-disabled {
  color: var(--cl-warning-color);
}
</style>