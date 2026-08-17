<script setup lang="ts">
import { computed } from 'vue';
import {
  translate,
  getIconBySelectorType,
  getIconByExtractType,
} from '@/utils';

const props = defineProps<{
  selectorType: SelectorType;
  selector: string;
  extractType?: ExtractType;
  attribute?: string;
}>();

const emit = defineEmits<{
  (e: 'click'): void;
}>();

const t = translate;

const selectorIcon = computed<Icon>(() => {
  const { selectorType } = props;
  return getIconBySelectorType(selectorType);
});

const extractIcon = computed<Icon>(() => {
  const { extractType } = props;
  return getIconByExtractType(extractType);
});

defineOptions({ name: 'ClAutoProbeSelector' });
</script>

<template>
  <div class="selector">
    <cl-tag
      :icon="selectorIcon"
      :label="selector"
      :tooltip="
        t(`components.autoprobe.pagePattern.selectorTypes.${selectorType}`)
      "
      clickable
    >
      <template #tooltip>
        <div>
          <label>
            {{ t('components.autoprobe.pagePattern.selectorType') }}:
          </label>
          <span>
            {{
              t(
                `components.autoprobe.pagePattern.selectorTypes.${selectorType}`
              )
            }}
          </span>
        </div>
        <div>
          <label>{{ t('components.autoprobe.pagePattern.selector') }}: </label>
          <span>{{ selector }}</span>
        </div>
      </template>
    </cl-tag>
    <template v-if="extractType">
      <span class="divider">
        <cl-icon :icon="['fa', 'angle-right']" />
      </span>
      <cl-tag
        :icon="extractIcon"
        :label="attribute"
        clickable
        @click="emit('click')"
      >
        <template #tooltip>
          <div>
            <label>
              {{ t('components.autoprobe.pagePattern.extractionType') }}:
            </label>
            <span>
              {{
                t(
                  `components.autoprobe.pagePattern.extractionTypes.${extractType}`
                )
              }}
            </span>
          </div>
          <div v-if="extractType === 'attribute'">
            <label>
              {{ t('components.autoprobe.pagePattern.attribute') }}:
            </label>
            <span>{{ attribute }}</span>
          </div>
        </template>
      </cl-tag>
    </template>
  </div>
</template>

<style scoped>
.selector {
  display: flex;
  align-items: center;
  gap: 5px;

  .divider {
    color: var(--el-text-color-secondary);
    font-size: 10px;
  }
}
</style>
