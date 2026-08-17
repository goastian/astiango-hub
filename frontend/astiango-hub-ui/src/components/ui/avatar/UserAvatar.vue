<script setup lang="ts">
import { computed } from 'vue';
import { getUserShortName } from '@/utils/user';

const props = withDefaults(
  defineProps<{
    icon?: Icon;
    size?: BasicSize | number;
    shape?: 'circle' | 'square';
    src?: string;
    alt?: string;
    fit?: 'fill' | 'contain' | 'cover' | 'none' | 'scale-down';
    color?: string;
    user?: User;
    tooltip?: string;
  }>(),
  {
    size: 36,
  }
);

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void;
}>();

const slots = defineSlots<{
  default: any;
}>();

const userLabel = computed<string>(() => {
  const { user } = props;
  if (user) {
    return getUserShortName(user);
  }
  return '';
});

defineOptions({ name: 'ClUserAvatar' });
</script>

<template>
  <div class="avatar">
    <el-tooltip :content="tooltip" :disabled="!tooltip">
      <el-avatar
        v-bind="props"
        class="avatar"
        @click="(e: MouseEvent) => emit('click', e)"
      >
        <slot v-if="slots.default" name="default" />
        <template v-else-if="user">
          <span class="label">
            {{ userLabel }}
          </span>
        </template>
        <template v-else></template>
      </el-avatar>
    </el-tooltip>
  </div>
</template>

<style scoped>
.avatar {
  max-height: 50px;

  .el-avatar {
    background-color: var(--el-color-primary-dark-2);
    color: #ffffff;
    cursor: pointer;
    max-height: 50px;

    &:hover {
      background-color: var(--cl-primary-color);
    }

    .label {
      display: flex;
      align-items: center;
      justify-content: center;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      max-width: 100%;
      min-height: 100%;
      line-height: 1.5;
      padding: 0 2px;
    }
  }
}
</style>
