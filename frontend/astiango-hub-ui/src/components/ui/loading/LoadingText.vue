<script setup lang="ts">
import { ref, onUnmounted, onMounted } from 'vue';

const props = defineProps<{
  text?: string;
  interval?: number;
}>();

const loadingDots = ref('');
let dotInterval: any = null;

const startLoadingAnimation = () => {
  let count = 0;
  dotInterval = setInterval(() => {
    count = (count + 1) % 4;
    loadingDots.value = '.'.repeat(count);
  }, props.interval || 500);
};

const stopLoadingAnimation = () => {
  if (dotInterval) {
    clearInterval(dotInterval);
    dotInterval = null;
  }
};

onMounted(() => {
  startLoadingAnimation();
});

onUnmounted(() => {
  stopLoadingAnimation();
});
defineOptions({ name: 'ClLoadingText' });
</script>

<template>
  <span class="loading-text">
    {{ text }}{{ loadingDots }}
  </span>
</template>

<style scoped>
.loading-text {
  display: inline-block;
}
</style>
