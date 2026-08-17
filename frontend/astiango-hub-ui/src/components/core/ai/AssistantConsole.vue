<script setup lang="ts">
import { onBeforeMount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { ClChatInput } from '@/components';
import { debounce } from 'lodash';
import { useRouter } from 'vue-router';
import useAssistantConsole from './useAssistantConsole';

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

const { t } = useI18n();

const router = useRouter();

const {
  messageListRef,
  chatInputRef,
  currentConversationId,
  conversations,
  chatHistory,
  isGenerating,
  streamError,
  isLoadingConversations,
  isLoadingMessages,
  historyDialogVisible,
  configDialogVisible,
  abortController,
  availableProviders,
  chatbotConfig,
  currentConversationTitle,
  loadConversations,
  saveChatbotConfig,
  selectConversation,
  createNewConversation,
  sendStreamingRequest,
  extractErrorMessage,
  initializeConversation,
} = useAssistantConsole();

// Message handling
const sendMessage = async (message: string) => {
  if (!message.trim()) return;

  streamError.value = '';
  abortController.value = new AbortController();

  chatHistory.push({
    role: 'user',
    content: message,
    timestamp: new Date(),
    conversation_id: currentConversationId.value || '',
    status: 'completed',
  });
  // Scroll to bottom after adding user message
  messageListRef.value?.scrollToBottom();

  const responseIndex = chatHistory.length;
  chatHistory.push({
    role: 'assistant',
    timestamp: new Date(),
    isStreaming: true,
    conversation_id: currentConversationId.value || '',
    status: 'pending',
    contentsMap: {},
  });
  // Scroll to bottom after adding system message placeholder
  messageListRef.value?.scrollToBottom();

  isGenerating.value = true;

  try {
    await sendStreamingRequest(message, responseIndex, () => {
      messageListRef.value?.scrollToBottom();
    });
  } catch (error) {
    console.error('Error sending message:', error);

    if (error instanceof DOMException && error.name === 'AbortError') {
      chatHistory.splice(responseIndex, 1);
    } else {
      streamError.value =
        error instanceof Error
          ? extractErrorMessage(error.message)
          : 'An error occurred while sending your message';
    }
  } finally {
    isGenerating.value = false;
    focusChatInput();
    abortController.value = null;
  }
};

const cancelMessage = () => {
  if (abortController.value) {
    abortController.value.abort();
    abortController.value = null;
    isGenerating.value = false;

    const streamingMessageIndex = chatHistory.findIndex(
      (msg: ChatMessage) => msg.isStreaming
    );
    if (streamingMessageIndex >= 0) {
      chatHistory.splice(streamingMessageIndex, 1);
    }
  }
};

const selectProviderModel = ({
  providerId,
  model,
}: {
  providerId: string;
  model: string;
}) => {
  chatbotConfig.value.providerId = providerId;
  chatbotConfig.value.model = model;
  localStorage.setItem('chatbotConfig', JSON.stringify(chatbotConfig.value));
};

const addProviderModel = () => {
  router.push('/system/models');
};

const openConfig = () => {
  configDialogVisible.value = true;
};

const openHistory = debounce(() => {
  historyDialogVisible.value = true;
  loadConversations();
});

const focusChatInput = debounce(() => {
  chatInputRef.value?.focus();
});

const stopMessageStreaming = () => {
  chatHistory
    .filter((msg: ChatMessage) => msg.isStreaming)
    .forEach((msg: ChatMessage) => {
      msg.isStreaming = false;
    });
};

watch(isGenerating, () => {
  if (!isGenerating.value) {
    stopMessageStreaming();
  }
});

// Initialize
onBeforeMount(initializeConversation);
watch(
  () => props.visible,
  async () => {
    if (props.visible) {
      await initializeConversation();
    }
  }
);

defineOptions({ name: 'ClAssistantConsole' });
</script>

<template>
  <div class="assistant-console">
    <div class="console-header">
      <span v-if="visible" class="chat-toggle-btn" @click="emit('close')">
        <cl-icon :icon="['fa', 'angles-right']" class="toggle-indicator" />
      </span>
      <div class="chat-conversation-title" :title="currentConversationTitle">
        {{ currentConversationTitle }}
      </div>
      <el-tooltip :content="t('components.ai.chatbot.new')">
        <el-button type="text" @click="createNewConversation" class="new-btn">
          <cl-icon :icon="['fas', 'plus']" />
        </el-button>
      </el-tooltip>
      <el-popover
        v-model:visible="historyDialogVisible"
        trigger="click"
        :show-arrow="false"
        placement="bottom-end"
        width="320"
      >
        <template #reference>
          <div class="history-btn-wrapper">
            <el-tooltip :content="t('components.ai.chatbot.history')">
              <el-button
                type="text"
                @click.prevent="openHistory"
                class="history-btn"
              >
                <cl-icon :icon="['fas', 'history']" />
              </el-button>
            </el-tooltip>
          </div>
        </template>
        <cl-chat-history
          :conversations="conversations"
          :selected-conversation-id="currentConversationId"
          :is-loading="isLoadingConversations"
          @select="selectConversation"
          @close="historyDialogVisible = false"
        />
      </el-popover>
      <el-tooltip :content="t('components.ai.chatbot.config.title')">
        <el-button type="text" @click="openConfig" class="config-btn">
          <cl-icon :icon="['fas', 'cog']" />
        </el-button>
      </el-tooltip>
    </div>

    <div class="chat-container">
      <div class="chat-content">
        <cl-chat-message-list
          :messages="chatHistory"
          :is-loading="isLoadingMessages"
          :error="streamError"
          ref="messageListRef"
        />

        <cl-chat-input
          ref="chatInputRef"
          :is-loading="isGenerating"
          :providers="availableProviders"
          :selected-provider-model="{
            providerId: chatbotConfig.providerId,
            model: chatbotConfig.model,
          }"
          :selected-model="chatbotConfig.model"
          @send="sendMessage"
          @cancel="cancelMessage"
          @model-change="selectProviderModel"
          @add-model="addProviderModel"
        />
      </div>
    </div>

    <cl-chat-config-dialog
      :visible="configDialogVisible"
      :providers="availableProviders"
      :current-config="chatbotConfig"
      @close="configDialogVisible = false"
      @confirm="saveChatbotConfig"
    />
  </div>
</template>

<style scoped>
.assistant-console {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--el-bg-color);
}

.console-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 64px;
  padding: 16px;
  gap: 8px;
  border-bottom: 1px solid var(--el-border-color-light);
  background-color: var(--el-color-white);
}

.chat-container {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-toggle-btn {
  display: flex;
  align-items: center;
  padding: 8px;
  animation: fadeIn 0.3s ease-in-out;
  color: var(--el-color-primary-dark-2);
  cursor: pointer;
  border-radius: 20px;
  transition: all 0.3s;

  &:hover {
    color: white;
    background-color: var(--el-color-primary-dark-2);
  }
}

.chat-toggle-btn .button-text {
  margin: 0 8px;
  display: inline-block;
}

.chat-toggle-btn .toggle-indicator {
  margin-left: 4px;
  transition: transform 0.3s;
}

.chat-toggle-btn.is-active {
  background-color: var(--el-color-primary-dark-2);
}

.chat-toggle-btn.is-active .toggle-indicator {
  transform: rotate(180deg);
}

.chat-conversation-title {
  font-size: 14px;
  color: var(--el-text-color-regular);
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
  width: 100%;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateX(-20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
</style>
