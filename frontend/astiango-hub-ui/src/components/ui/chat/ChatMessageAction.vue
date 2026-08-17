<script setup lang="tsx">
import { computed, ref } from 'vue';
import JsonEditorVue from 'json-editor-vue';

const props = defineProps<{
  action: string;
  actionTarget?: string;
  actionStatus: ChatMessageActionStatus;
  parameters?: Record<string, any>;
  content?: string;
}>();

const isExpanded = ref(false);
const isFullScreen = ref(false);
const isParamsExpanded = ref(true);
const isResponseExpanded = ref(true);

const actionStatusIcon = computed<Icon>(() => {
  switch (props.actionStatus) {
    case 'pending':
      return ['fas', 'circle-notch'];
    case 'success':
      return ['fas', 'check-circle'];
    case 'failed':
      return ['fas', 'times-circle'];
    default:
      return ['fas', 'question-circle'];
  }
});

const parsedContent = computed<
  Record<string, any> | Record<string, any>[] | null
>(() => {
  if (!props.content) return null;
  try {
    return JSON.parse(props.content);
  } catch (e) {
    return null;
  }
});

const hasParameters = computed(() => {
  return props.parameters && Object.keys(props.parameters).length > 0;
});

const isJsonContent = computed(() => {
  return parsedContent.value !== null || hasParameters.value;
});

const hasContent = computed(() => {
  return props.content || hasParameters.value;
});

const actionLabel = computed<string>(() => {
  const { action, actionTarget } = props;
  if (actionTarget) {
    return `${action}: ${actionTarget}`;
  }
  return action;
});

defineOptions({ name: 'ClChatMessageAction' });
</script>

<template>
  <div class="action-wrapper" :class="[actionStatus, { expanded: isExpanded }]">
    <div class="action-header" role="button">
      <span class="action-header-icon">
        <cl-icon
          :icon="actionStatusIcon"
          :spinning="actionStatus === 'pending'"
        />
      </span>
      <span
        class="action-name"
        :class="{ 'flash-text': actionStatus === 'pending' }"
        @click="isExpanded = !isExpanded"
      >
        {{ actionLabel }}
        <span
          v-if="actionStatus === 'pending'"
          class="flash-overlay"
          aria-hidden="true"
        ></span>
      </span>
      <div class="action-header-buttons">
        <cl-icon
          v-if="isJsonContent"
          class="action-button"
          :icon="['fas', 'expand']"
          @click.stop="isFullScreen = true"
        />
        <cl-icon
          v-if="hasContent"
          class="action-button"
          :icon="['fas', isExpanded ? 'chevron-up' : 'chevron-down']"
          @click.stop="isExpanded = !isExpanded"
        />
      </div>
    </div>
    <div
      v-if="hasContent"
      class="action-content"
      :class="{ expanded: isExpanded }"
    >
      <el-scrollbar max-height="500px">
        <!-- Parameters Section -->
        <div v-if="hasParameters" class="content-section">
          <div
            class="section-header"
            @click="isParamsExpanded = !isParamsExpanded"
          >
            <span class="section-title">Parameters</span>
            <cl-icon
              class="action-button"
              :icon="['fas', isParamsExpanded ? 'chevron-up' : 'chevron-down']"
            />
          </div>
          <div v-show="isParamsExpanded" class="json-content">
            <json-editor-vue
              :model-value="parameters"
              expanded-on-start
              read-only
            />
          </div>
        </div>

        <!-- Response Section -->
        <div v-if="content" class="content-section">
          <div
            class="section-header"
            @click="isResponseExpanded = !isResponseExpanded"
          >
            <span class="section-title">Response</span>
            <cl-icon
              class="action-button"
              :icon="[
                'fas',
                isResponseExpanded ? 'chevron-up' : 'chevron-down',
              ]"
            />
          </div>
          <div v-show="isResponseExpanded">
            <template v-if="parsedContent">
              <div class="json-content">
                <json-editor-vue
                  :model-value="parsedContent"
                  expanded-on-start
                  read-only
                />
              </div>
            </template>
            <template v-else>
              <div class="text-content">{{ content }}</div>
            </template>
          </div>
        </div>
      </el-scrollbar>
    </div>
  </div>

  <!-- Full Screen Dialog -->
  <el-dialog
    v-if="isJsonContent"
    v-model="isFullScreen"
    :title="action"
    width="80%"
    fullscreen
    append-to-body
  >
    <div class="fullscreen-content">
      <!-- Parameters Section -->
      <div v-if="hasParameters" class="content-section">
        <div
          class="section-header"
          @click="isParamsExpanded = !isParamsExpanded"
        >
          <span class="section-title">Parameters</span>
          <cl-icon
            class="action-button"
            :icon="['fas', isParamsExpanded ? 'chevron-up' : 'chevron-down']"
          />
        </div>
        <div v-show="isParamsExpanded" class="json-content">
          <json-editor-vue
            :model-value="parameters"
            expanded-on-start
            read-only
          />
        </div>
      </div>

      <!-- Response Section -->
      <div v-if="content" class="content-section">
        <div
          class="section-header"
          @click="isResponseExpanded = !isResponseExpanded"
        >
          <span class="section-title">Response</span>
          <cl-icon
            class="action-button"
            :icon="['fas', isResponseExpanded ? 'chevron-up' : 'chevron-down']"
          />
        </div>
        <div v-show="isResponseExpanded">
          <template v-if="parsedContent">
            <div class="json-content">
              <json-editor-vue
                :model-value="parsedContent"
                expanded-on-start
                read-only
              />
            </div>
          </template>
          <template v-else>
            <div class="text-content">{{ content }}</div>
          </template>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<style scoped>
.action-wrapper {
  padding: 6px 12px;
  margin: 0;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-size: 0.9em;
  cursor: pointer;
  transition: background-color 0.2s;
}

.action-wrapper:hover {
  background: var(--el-fill-color);
}

.action-header {
  display: flex;
  align-items: center;
  gap: 8px;
  user-select: none;
}

.action-header-icon {
  display: flex;
  align-items: center;
}

.pending .action-header-icon {
  color: var(--el-color-warning);
}

.success .action-header-icon {
  color: var(--el-color-success);
}

.failed .action-header-icon {
  color: var(--el-color-danger);
}

.action-name {
  flex: 1;
  position: relative;
  overflow: hidden;
  cursor: pointer;
}

.flash-text {
  position: relative;
  color: var(--el-text-color-regular);
}

.flash-overlay {
  position: absolute;
  top: 0;
  left: -100%;
  width: 50%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.6) 50%,
    transparent 100%
  );
  animation: flash-animation 1.5s infinite linear;
  pointer-events: none;
}

@keyframes flash-animation {
  0% {
    left: -50%;
  }
  100% {
    left: 150%;
  }
}

.action-header-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-button {
  opacity: 0.6;
  cursor: pointer;
  transition: opacity 0.2s;
}

.action-button:hover {
  opacity: 1;
}

.action-content {
  max-height: 0;
  overflow: hidden;
  font-size: 0.9em;
  opacity: 0.8;
  transition: max-height 0.2s ease-out;
}

.action-content.expanded {
  max-height: 500px;
  margin-top: 4px;
  transition: max-height 0.3s ease-in;
}

.content-section {
  margin-bottom: 4px;
}

.content-section:last-child {
  margin-bottom: 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px 0 0;
  margin-bottom: 0;
  cursor: pointer;
}

.section-title {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.json-content {
  padding: 0 10px 0 0;
}

.text-content {
  padding: 8px;
  white-space: pre-wrap;
  word-break: break-word;
}

.action-content :deep(.jse-main) {
  background: transparent;
  border: none;
  font-size: 0.9em;
  padding: 0;
}

.action-content :deep(.jse-main .jse-value) {
  color: var(--el-text-color-regular);
  word-break: break-word;
  white-space: pre-wrap;
}

.action-content :deep(.jse-main .jse-key) {
  color: var(--el-color-primary);
}

.action-content :deep(.jse-main .jse-value pre) {
  max-width: 100%;
  overflow-x: auto;
}

.fullscreen-content {
  height: calc(100vh - 80px);
  overflow-y: auto;
  padding: 16px;
}

.fullscreen-content .content-section {
  background: var(--el-bg-color);
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 16px;
}
</style>
<style>
.el-dialog.is-fullscreen .jse-main {
  background: var(--el-bg-color);
  padding: 16px;
  border-radius: 4px;
}
</style>
