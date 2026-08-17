<script setup lang="ts">
import { computed, onBeforeMount, ref, watch } from 'vue';
import { useStore } from 'vuex';
import { ElMessage, ElMessageBox } from 'element-plus';
import { EMPTY_OBJECT_ID, translate } from '@/utils';
import useRequest from '@/services/request';
import { useSpiderDetail } from '@/views';
import { debounce } from 'lodash';
import { DATABASE_STATUS_OFFLINE } from '@/constants/database';
import { getDatabaseName } from '@/utils/database';

const t = translate;

const { get } = useRequest();

// store
const ns = 'spider';
const store = useStore();
const { spider: state } = store.state as RootStoreState;

// display all fields
const displayAllFields = ref<boolean>(state.dataDisplayAllFields);
const onDisplayAllFieldsChange = (val: boolean) => {
  store.commit(`${ns}/setDataDisplayAllFields`, val);
};

const form = computed(() => state.form);

const { activeId } = useSpiderDetail();

const allDatabases = ref<Database[]>([]);
const getAllDatabases = async () => {
  const res = await get<Database[]>('/databases', { size: 100 });
  allDatabases.value = res.data || [];
};
onBeforeMount(getAllDatabases);

const currentDatabase = computed<Database | undefined>(() => {
  return allDatabases.value.find(db => db._id === databaseId.value);
});

const isDatabaseOffline = computed(() => {
  return currentDatabase.value?.status === DATABASE_STATUS_OFFLINE;
});

const isDatabaseTableMissing = computed(() => {
  if (currentDatabase.value?.data_source === 'mongo') {
    return false;
  }
  if (isMultiDatabases.value) {
    return !databaseTableSelectOptions.value.some(
      op =>
        op.value === form.value?.db_name &&
        op.children?.some(c => c.value === form.value?.col_name)
    );
  } else {
    return !databaseTableSelectOptions.value.some(
      op => op.value === form.value?.col_name
    );
  }
});

const databaseMetadata = computed(() => state.databaseMetadata);
const isMultiDatabases = computed<boolean>(() => {
  if (!databaseMetadata.value?.databases?.length) return false;
  return databaseMetadata.value.databases.length > 1;
});
watch(isMultiDatabases, () => {
  if (isMultiDatabases.value) {
    updateMultiDbTableName();
  } else {
    updateTableName();
  }
});

const databaseId = ref<string>(form.value?.database_id || EMPTY_OBJECT_ID);
const onDatabaseChange = async (value: string) => {
  databaseId.value = form.value?.database_id || EMPTY_OBJECT_ID;
  await ElMessageBox.confirm(
    t('components.spider.messageBox.confirm.changeDatabase.message'),
    t('components.spider.messageBox.confirm.changeDatabase.title'),
    {
      type: 'warning',
    }
  );
  store.commit(`${ns}/setForm`, {
    ...form.value,
    database_id: value,
  });
  try {
    await store.dispatch(`${ns}/updateById`, {
      id: activeId.value,
      form: form.value,
    });
    ElMessage.success(t('common.message.success.save'));
  } catch (e: any) {
    ElMessage.error(e.message);
  }
};
watch(
  () => form.value?.database_id,
  value => {
    databaseId.value = value || EMPTY_OBJECT_ID;
  }
);
const getDataSourceByDatabaseId = (id: string): DatabaseDataSource => {
  const db = allDatabases.value.find(db => db._id === id) as Database;
  if (!db?.data_source) return 'mongo';
  return db.data_source;
};

// database table options
const databaseTableSelectOptions = computed<SelectOption[]>(() => {
  return store.getters[`${ns}/databaseTableSelectOptions`];
});

// single database table name
const tableName = ref<string>('');
const updateTableName = () => {
  if (isMultiDatabases.value) return;
  tableName.value = form.value?.col_name || '';
};
watch(() => form.value?.col_name, updateTableName);
onBeforeMount(updateTableName);
const onTableChange = debounce(async (value: string | string[]) => {
  store.commit(`${ns}/setForm`, {
    ...form.value,
    db_name: '',
    col_name: value,
  });
  await updateForm();
});

// multi database table name
const multiDbTableName = ref<string[]>([]);
const updateMultiDbTableName = () => {
  if (!isMultiDatabases.value) return;
  multiDbTableName.value = [
    form.value.db_name || '',
    form.value.col_name || '',
  ];
};
watch(
  () => JSON.stringify([form.value?.db_name, form.value?.col_name]),
  updateMultiDbTableName
);
onBeforeMount(updateMultiDbTableName);
const onMultiDbTableChange = debounce(async (value: string[]) => {
  const dbName = value[0];
  const colName = value[1];
  store.commit(`${ns}/setForm`, {
    ...form.value,
    db_name: dbName,
    col_name: colName,
  });
  await updateForm();
});

// update form (save spider)
const updateForm = async () => {
  try {
    await store.dispatch(`${ns}/updateById`, {
      id: activeId.value,
      form: form.value,
    });
    ElMessage.success(t('common.message.success.save'));
  } catch (e: any) {
    ElMessage.error(e.message);
  }
};

defineOptions({ name: 'ClSpiderDetailActionsData' });
</script>

<template>
  <cl-nav-action-group v-if="form" class="spider-detail-actions-data">
    <cl-nav-action-fa-icon :icon="['fa', 'table']" />
    <cl-nav-action-item>
      <el-select
        class="database"
        :class="isDatabaseOffline ? 'offline' : ''"
        v-model="databaseId"
        @change="onDatabaseChange"
      >
        <template #label="{ label }">
          <div>
            <cl-database-data-source
              :data-source="
                getDataSourceByDatabaseId(form.database_id as string)
              "
              icon-only
            />
            <span style="margin: 5px">{{ label }}</span>
            <cl-icon
              v-if="form.database_id === EMPTY_OBJECT_ID"
              color="var(--cl-warning-color)"
              :icon="['fa', 'star']"
            />
          </div>
        </template>
        <el-option
          v-for="db in allDatabases"
          :key="db._id"
          :label="getDatabaseName(db)"
          :value="db._id"
        >
          <div>
            <cl-database-data-source
              :data-source="getDataSourceByDatabaseId(db._id!)"
              icon-only
            />
            <span style="margin: 5px">{{ getDatabaseName(db) }}</span>
            <cl-icon
              v-if="db._id === EMPTY_OBJECT_ID"
              color="var(--cl-warning-color)"
              :icon="['fa', 'star']"
            />
          </div>
        </el-option>
      </el-select>
    </cl-nav-action-item>
    <cl-nav-action-item>
      <template v-if="isMultiDatabases">
        <el-cascader
          class="table"
          :class="isDatabaseTableMissing ? 'missing' : ''"
          v-model="multiDbTableName"
          :options="databaseTableSelectOptions"
          filterable
          :placeholder="t('components.spider.actions.data.placeholder.table')"
          :disabled="isDatabaseOffline"
          @change="onMultiDbTableChange"
        >
          <template #label="{ label }">
            <div>
              <cl-icon :icon="['fa', 'table']" />
              <span style="margin-left: 5px">{{ label }}</span>
            </div>
          </template>
        </el-cascader>
      </template>
      <template v-else>
        <el-select
          class="table"
          :class="isDatabaseTableMissing ? 'missing' : ''"
          v-model="tableName"
          filterable
          :placeholder="t('components.spider.actions.data.placeholder.table')"
          :disabled="isDatabaseOffline"
          @change="onTableChange"
        >
          <template #label="{ label }">
            <div>
              <cl-icon :icon="['fa', 'table']" />
              <span style="margin-left: 5px">{{ label }}</span>
            </div>
          </template>
          <el-option
            v-for="(op, $index) in databaseTableSelectOptions"
            :key="$index"
            :label="op.label"
            :value="op.value"
          />
        </el-select>
      </template>
    </cl-nav-action-item>
    <cl-nav-action-item>
      <el-tooltip
        :content="t('components.spider.actions.data.tooltip.displayAllFields')"
      >
        <div class="display-all-fields">
          <cl-switch
            v-model="displayAllFields"
            :active-icon="['fa', 'eye']"
            :inactive-icon="['fa', 'eye']"
            inline-prompt
            @change="onDisplayAllFieldsChange"
          />
        </div>
      </el-tooltip>
    </cl-nav-action-item>
    <cl-nav-action-item>
      <cl-export-button
        :target="form.col_name"
        :db-id="currentDatabase?._id"
        :tooltip="t('common.actions.exportData')"
      />
    </cl-nav-action-item>
  </cl-nav-action-group>
</template>

<style scoped>
.spider-detail-actions-data {
  &:deep(.el-cascader),
  &:deep(.el-select) {
    width: 150px;

    &.database {
      width: 160px;

      &.offline {
        &:deep(.el-select__wrapper) {
          box-shadow: 0 0 0 1px var(--cl-danger-color) inset;
        }
      }
    }

    &.table {
      &.missing {
        &:deep(.el-select__wrapper) {
          box-shadow: 0 0 0 1px var(--cl-danger-color) inset;
        }
      }
    }
  }

  &:deep(.el-cascader) {
    width: 200px;
  }
}
</style>
