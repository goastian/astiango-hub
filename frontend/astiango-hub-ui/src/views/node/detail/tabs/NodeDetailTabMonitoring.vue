<script setup lang="ts">
import { ref } from 'vue';
import { getAllMetricGroups } from '@/utils/metric';
import { loadLocalStorage, saveLocalStorage } from '@/utils/storage';

const timeRangeKey = 'node.monitoring.timeRange';
const metricGroupsKey = 'node.monitoring.metricGroups';

const defaultTimeRange = ref<string>(loadLocalStorage(timeRangeKey) || '1h');

const defaultMetricGroups = ref<string[]>(
  loadLocalStorage(metricGroupsKey) || [
    'cpu_usage_percent',
    'used_memory_percent',
    'used_disk_percent',
    'disk_io_bytes_rate',
    'network_io_bytes_rate',
  ]
);

const onTimeRangeChange = (timeRange: string) => {
  saveLocalStorage(timeRangeKey, timeRange);
};

const onMetricGroupsChange = (metricGroups: string[]) => {
  saveLocalStorage(metricGroupsKey, metricGroups);
};

defineOptions({ name: 'ClNodeDetailTabMonitoring' });
</script>

<template>
  <cl-metric-monitoring-detail
    ns="node"
    api-prefix="/nodes"
    :default-metric-groups="defaultMetricGroups"
    :default-time-range="defaultTimeRange"
    :all-metric-groups-fn="getAllMetricGroups"
    @time-range-change="onTimeRangeChange"
    @metric-groups-change="onMetricGroupsChange"
  />
</template>
