<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import markdownit from 'markdown-it';
import hljs from 'highlight.js';
import { computed, ref, watch } from 'vue';
import 'highlight.js/styles/github.css';
import ClChatMessageAction from '@/components/ui/chat/ChatMessageAction.vue';
import { Document } from '@element-plus/icons-vue';

const { t } = useI18n();

const props = defineProps<{
  message: ChatMessage;
}>();

const md = markdownit({
  html: true,
  linkify: true,
  typographer: true,
  highlight: function (str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        // Add data-language attribute to pre tag
        return (
          '<pre data-language="' +
          lang +
          '"><code class="hljs language-' +
          lang +
          '">' +
          hljs.highlight(str, { language: lang }).value +
          '</code></pre>'
        );
      } catch (__) {}
    }
    // For unknown languages, still add the language if provided
    return lang
      ? '<pre data-language="' + lang + '"><code>' + str + '</code></pre>'
      : '';
  },
});

// Format timestamp
const formatTime = (date: Date | undefined): string => {
  return date
    ? date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : '';
};

// Safe markdown rendering with sanitization
const renderMarkdown = (content: string): string => {
  if (!content) return '';
  return md.render(content);
};

const typing = ref(true);

// Stop the typing animation when streaming ends
watch(
  () => props.message.isStreaming,
  isStreaming => {
    typing.value = isStreaming === true;
  },
  { immediate: true }
);

const filteredContents = computed<ChatMessageContent[]>(() => {
  const { message } = props;
  return message.contents?.filter(content => !content.hidden) || [];
});

// Compute token usage display
const hasTokenUsage = computed(() => {
  const { message } = props;
  return message.usage &&
    (message.usage.total_tokens ||
     message.usage.prompt_tokens ||
     message.usage.completion_tokens);
});

// Format token count with comma separators
const formatTokenCount = (count?: number): string => {
  return count ? count.toLocaleString() : '0';
};

defineOptions({ name: 'ClChatMessage' });
</script>

<template>
  <div :class="['message-container', message.role]">
    <div class="message-content">
      <template v-if="message.content">
        <div v-html="renderMarkdown(message.content)"></div>
      </template>

      <!-- Iterate through content items in order -->
      <div v-else class="content-items">
        <template v-for="(content, index) in filteredContents" :key="index">
          <!-- Action content -->
          <cl-chat-message-action
            v-if="content.type === 'action'"
            :action="content.action!"
            :action-target="content.action_target"
            :action-status="content.action_status!"
            :parameters="content.parameters"
            :content="content.content"
          />

          <!-- Text content -->
          <div v-else-if="content.type === 'text'" class="text-content">
            <template v-if="content.isStreaming">
              <div v-html="renderMarkdown(content.content || '')"></div>
              <span class="typing-indicator" v-if="typing">|</span>
            </template>
            <template v-else>
              <div v-html="renderMarkdown(content.content || '')"></div>
            </template>
          </div>
        </template>
      </div>
    </div>

    <div class="message-footer">
      <!-- Show 'Generating...' for streaming messages -->
      <template v-if="message.isStreaming">
        <cl-loading-text
          class="typing-text"
          :text="t('components.ai.chatbot.generating')"
        />
      </template>
      <template v-else>
        <span class="message-time">
          {{ formatTime(message.timestamp) }}
        </span>
      </template>

      <!-- Token usage display as icon -->
      <el-popover
        v-if="hasTokenUsage && !message.isStreaming && message.role === 'assistant'"
        placement="top"
        trigger="hover"
        :width="220"
        popper-class="token-usage-popover"
      >
        <template #reference>
          <div class="token-usage-icon" :title="formatTokenCount(message.usage?.total_tokens) + ' ' + t('components.ai.chatbot.tokens')">
            <cl-icon :icon="['fa', 'calculator']" />
          </div>
        </template>
        <div class="token-usage-details">
          <div class="token-usage-row">
            <span>{{ t('components.ai.chatbot.promptTokens') }}:</span>
            <span>{{ formatTokenCount(message.usage?.prompt_tokens) }}</span>
          </div>
          <div class="token-usage-row">
            <span>{{ t('components.ai.chatbot.completionTokens') }}:</span>
            <span>{{ formatTokenCount(message.usage?.completion_tokens) }}</span>
          </div>
          <div class="token-usage-row total">
            <span>{{ t('components.ai.chatbot.totalTokens') }}:</span>
            <span>{{ formatTokenCount(message.usage?.total_tokens) }}</span>
          </div>
        </div>
      </el-popover>
    </div>
  </div>
</template>

<style scoped>
.message-container {
  font-size: 14px;
  width: calc(100% - 24px);
  margin: 0 12px;
  padding: 12px;
  position: relative;

  &:hover {
    .token-usage-icon {
      visibility: visible;
    }
  }
}

.message-container:first-child.user {
  margin-top: 12px;
}

.message-container.user {
  border-radius: 12px;
  background-color: var(--el-color-primary-dark-2);
}

.message-container.system {
  background-color: transparent;
}

.message-content {
  word-break: break-word;
  line-height: 1.5;
  max-height: 100%;
  overflow-y: auto;

  /* Firefox scrollbar styles */
  scrollbar-width: thin;
  scrollbar-color: var(--el-border-color-darker) var(--el-fill-color-lighter);

  /* Webkit scrollbar styles */

  &::-webkit-scrollbar {
    width: 7px;
    background-color: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background-color: var(--el-border-color-darker);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-track {
    background-color: var(--el-fill-color-lighter);
    border-radius: 3px;
  }
}

.content-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.text-content {
  width: 100%;
}

/* Add styles for markdown elements */
.message-content :deep(pre) {
  background-color: var(--el-fill-color-light);
  padding: 24px 12px 12px 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 12px 0;
  position: relative;
}

.message-content :deep(pre code) {
  /* Firefox scrollbar styles */
  scrollbar-width: thin;
  scrollbar-color: var(--el-border-color-darker) var(--el-fill-color-lighter);

  /* Webkit scrollbar styles */

  &::-webkit-scrollbar {
    height: 7px;
    background-color: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background-color: var(--el-border-color-darker);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-track {
    background-color: var(--el-fill-color-lighter);
    border-radius: 3px;
  }
}

/* Add language display */
.message-content :deep(pre[data-language]::before) {
  content: attr(data-language);
  position: absolute;
  top: 0;
  left: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  padding: 2px 8px;
  border-bottom-left-radius: 4px;
  border-bottom-right-radius: 4px;
  text-transform: lowercase;
}

.message-content :deep(pre code) {
  font-family: 'Menlo', 'Monaco', 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  display: block;
  padding: 0;
  background: none;
}

.message-content :deep(code:not(pre code)) {
  background-color: var(--el-fill-color-light);
  padding: 2px 4px;
  border-radius: 4px;
  font-size: 0.9em;
}

.message-content :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}

.message-content :deep(blockquote) {
  border-left: 4px solid var(--el-color-info-light-5);
  padding-left: 12px;
  margin-left: 0;
  color: var(--el-text-color-secondary);
}

.message-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin-bottom: 16px;
}

.message-content :deep(th),
.message-content :deep(td) {
  border: 1px solid var(--el-border-color);
  padding: 8px;
  text-align: left;
}

.message-content :deep(th) {
  background-color: var(--el-color-info-light-9);
}

.message-content :deep(p) {
  margin: 0;
  padding: 0;
}

.message-content :deep(p:not(:first-child)) {
  margin-top: 6px;
}

.message-content :deep(h1) {
  margin: 0;
  padding: 12px 0;
}

.message-content :deep(h2) {
  margin: 0;
  padding: 12px 0;
}

.message-content :deep(h3) {
  margin: 0;
  padding: 12px 0;
}

.message-content :deep(h4) {
  margin: 0;
  padding: 12px 0;
}

.message-content :deep(h5) {
  margin: 0;
  padding: 12px 0;
}

.message-content :deep(h6) {
  margin: 0;
  padding: 12px 0;
}

.message-content :deep(ul) {
  margin: 0;
  padding-inline-start: 24px;
}

.message-content :deep(ol) {
  margin: 0;
  padding-inline-start: 24px;
}

.message-content :deep(li) {
  margin: 3px 0;
}

.message-footer {
  height: 16px;
  font-size: 10px;
  opacity: 0.7;
  margin-top: 6px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.message-container.user .message-content {
  color: var(--el-color-white);
}

.message-container.user .message-footer {
  color: var(--el-color-white);
}

.message-container.system .message-content {
  color: var(--el-text-color-regular);
}

.message-container.system .message-footer {
  color: var(--el-text-color-regular);
}

.typing-indicator {
  display: inline-block;
  animation: blink 1s infinite;
  margin-left: 2px;
}

.typing-text {
  display: inline-block;
  color: var(--el-color-primary);
}

.token-usage-icon {
  display: flex;
  align-items: center;
  font-size: 12px;
  cursor: pointer;
  gap: 4px;
  color: var(--el-color-info);
  visibility: hidden;

  &:hover {
    color: var(--el-color-primary);
  }
}

.token-count {
  font-size: 10px;
}

.token-usage-details {
  font-size: 12px;
}

.token-usage-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.token-usage-row.total {
  margin-top: 4px;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 4px;
  font-weight: bold;
}

@keyframes loadingDots {
  to {
    width: 1.25em;
  }
}

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0;
  }
}
</style>
