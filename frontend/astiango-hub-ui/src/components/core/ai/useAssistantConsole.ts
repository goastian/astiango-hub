import { computed, ref, watch, reactive, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import useRequest from '@/services/request';
import { getRequestBaseUrl } from '@/utils';
import { debounce } from 'lodash';
import { ElMessage } from 'element-plus';
import { AxiosError } from 'axios';
import { ClChatInput } from '@/components';

const useAssistantConsole = () => {
  const { t } = useI18n();
  const { get } = useRequest();

  // Refs
  const messageListRef = ref<{ scrollToBottom: () => Promise<void> } | null>(
    null
  );
  const chatInputRef = ref<InstanceType<typeof ClChatInput> | null>(null);

  // State management
  const currentConversation = ref<ChatConversation | null>(null);
  const currentConversationId = ref<string>('');
  const conversations = ref<ChatConversation[]>([]);
  const chatHistory = reactive<ChatMessage[]>([]);
  const isGenerating = ref(false);
  const streamError = ref('');
  const isLoadingConversations = ref(false);
  const isLoadingMessages = ref(false);
  const historyDialogVisible = ref(false);
  const configDialogVisible = ref(false);
  const abortController = ref<AbortController | null>(null);
  const availableProviders = ref<LLMProvider[]>([]);

  // Computed properties
  const currentConversationTitle = computed(() => {
    if (!currentConversationId.value) return t('components.ai.chatbot.newChat');
    return (
      currentConversation.value?.title || t('components.ai.chatbot.newChat')
    );
  });

  const chatbotConfig = ref<ChatbotConfig>({
    systemPrompt:
      'You are a helpful AI assistant for AstianGO Hub, a web crawling and data extraction platform.',
    temperature: 0.7,
    maxTokens: 1000,
  });

  // Load conversations
  const loadConversations = async () => {
    isLoadingConversations.value = true;
    try {
      const res = await get('/ai/chat/conversations', {
        page: 1,
        size: 500,
        sort: '-last_message_at',
      });
      conversations.value = res.data || [];
    } catch (error) {
      console.error('Failed to load conversations:', error);
    } finally {
      isLoadingConversations.value = false;
    }
  };

  // Load messages for a conversation
  const loadConversationMessages = debounce(async (conversationId: string) => {
    if (!conversationId) return;

    isLoadingMessages.value = true;
    try {
      const res = await get(
        `/ai/chat/conversations/${conversationId}/messages`
      );
      const messages = (res.data || []).map((msg: any) => {
        const message: ChatMessage = {
          ...msg,
          timestamp: new Date(msg.created_at || Date.now()),
        };
        return message;
      });

      messages.sort(
        (a: ChatMessage, b: ChatMessage) =>
          (a.timestamp?.getTime() || 0) - (b.timestamp?.getTime() || 0)
      );

      chatHistory.splice(0, chatHistory.length, ...messages);
      currentConversationId.value = conversationId;
    } catch (error) {
      console.error('Failed to load conversation messages:', error);
    } finally {
      isLoadingMessages.value = false;
    }
  });

  // Load current conversation details
  const loadCurrentConversation = debounce(async (conversationId: string) => {
    if (!conversationId) {
      currentConversation.value = null;
      return;
    }
    try {
      const res = await get(`/ai/chat/conversations/${conversationId}`);
      currentConversation.value = res.data;
    } catch (error) {
      if ((error as AxiosError)?.response?.status === 404) {
        currentConversationId.value = '';
        return;
      }
      console.error('Failed to load conversation details:', error);
      currentConversation.value = null;
    }
  });

  // Load LLM providers
  const loadLLMProviders = debounce(async () => {
    try {
      const res = await get('/ai/llm/providers', { available: true });
      availableProviders.value = res.data || [];

      if (!availableProviders.value.length) {
        resetChatbotConfig();
      }

      if (!chatbotConfig.value.providerId || !chatbotConfig.value.model) {
        if (availableProviders.value.length > 0) {
          chatbotConfig.value.providerId = availableProviders.value[0]._id;
          chatbotConfig.value.model = availableProviders.value[0].models![0];
          localStorage.setItem(
            'chatbotConfig',
            JSON.stringify(chatbotConfig.value)
          );
        }
      }
    } catch (error) {
      console.error('Failed to load LLM providers:', error);
    }
  });

  // Config management
  const loadChatbotConfig = () => {
    const storedConfig = localStorage.getItem('chatbotConfig');
    if (storedConfig) {
      try {
        const parsedConfig = JSON.parse(storedConfig);
        chatbotConfig.value = { ...chatbotConfig.value, ...parsedConfig };
      } catch (e) {
        console.error('Failed to parse stored chatbot config', e);
      }
    }
  };

  const saveChatbotConfig = (config: ChatbotConfig) => {
    configDialogVisible.value = false;
    chatbotConfig.value = { ...chatbotConfig.value, ...config };
    localStorage.setItem('chatbotConfig', JSON.stringify(chatbotConfig.value));
    ElMessage.success(t('common.message.success.save'));
  };

  const resetChatbotConfig = () => {
    chatbotConfig.value = {};
    localStorage.removeItem('chatbotConfig');
  };

  // Message handling
  const extractErrorMessage = (errorData: string): string => {
    try {
      const parsed = JSON.parse(errorData);
      if (parsed.error_detail?.message) return parsed.error_detail.message;
      if (parsed.error)
        return typeof parsed.error === 'string'
          ? parsed.error
          : JSON.stringify(parsed.error);
      if (parsed.text?.startsWith('Error:')) return parsed.text;
      if (typeof parsed === 'object') return JSON.stringify(parsed, null, 2);
      return errorData;
    } catch (e) {
      return errorData;
    }
  };

  // Message stream handling
  const createChatRequest = (message: string): ChatRequest => {
    const { providerId, model, systemPrompt, temperature, maxTokens } =
      chatbotConfig.value;

    if (!providerId || !model) {
      throw new Error(
        'Please select a provider and model before sending a message'
      );
    }

    return {
      provider_id: providerId,
      model,
      query: message,
      system_prompt: systemPrompt,
      temperature,
      max_tokens: maxTokens,
      conversation_id: currentConversationId.value,
    };
  };

  const getStreamHeaders = (): HeadersInit => {
    const token = localStorage.getItem('token');
    return {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: token } : {}),
    };
  };

  const handleStreamChunk = (
    chunk: ChatbotStreamMessage,
    currentMessage: ChatMessage
  ): void => {
    // Handle initial conversation ID update
    if (chunk.is_initial) {
      currentConversationId.value = chunk.conversation_id!;
      return;
    }

    // Update conversation title
    if (chunk.conversation_title) {
      if (!currentConversation.value) {
        currentConversation.value = {};
      }
      currentConversation.value.title = chunk.conversation_title;
    }

    // Update usage
    if (chunk.usage) {
      if (!currentMessage.usage) {
        currentMessage.usage = {
          prompt_tokens: 0,
          completion_tokens: 0,
          total_tokens: 0,
        };
      }
      currentMessage.usage.prompt_tokens! += chunk.usage.prompt_tokens || 0;
      currentMessage.usage.completion_tokens! +=
        chunk.usage.completion_tokens || 0;
      currentMessage.usage.total_tokens! += chunk.usage.total_tokens || 0;
    }

    // Initialize contents array if needed
    if (!currentMessage.contents?.length) {
      currentMessage.contents = [];
    }

    // Find content index
    const contentKey = chunk.key || '';
    const contentIndex = currentMessage.contents.findIndex(
      c => c.key === contentKey
    );

    if (contentIndex >= 0) {
      // Update existing content
      const content = currentMessage.contents[contentIndex];
      if (chunk.type === 'text') {
        content.content += chunk.content || '';
        if (chunk.is_text_done) {
          content.isStreaming = false;
        }
      } else if (chunk.type === 'action') {
        currentMessage.contents[contentIndex] = {
          ...content,
          ...chunk,
        };
      }
    } else {
      // Add new content
      if (chunk.type === 'text') {
        // Create new content object
        const newContent: ChatMessageContent = {
          ...chunk,
          content: chunk.content || '',
          isStreaming: true,
        };
        // Add new content to the message
        currentMessage.contents.push(newContent);
      } else if (chunk.type === 'action') {
        // Create new action content object
        currentMessage.contents.push({
          ...chunk,
        });
      }
    }
  };

  const processStreamData = async (
    reader: ReadableStreamDefaultReader<Uint8Array>,
    responseIndex: number,
    onMessageUpdate: (index: number) => void
  ): Promise<void> => {
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      const currentMessage = chatHistory[responseIndex];

      if (done) {
        if (currentMessage) {
          currentMessage.isStreaming = false;
          onMessageUpdate(responseIndex);
        }
        return;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data:')) {
          const eventData = line.slice(5).trim();
          if (eventData === '') continue;

          try {
            const chunk: ChatbotStreamMessage = JSON.parse(eventData);
            handleStreamChunk(chunk, currentMessage);

            if (chunk.is_done) {
              if (chatHistory[responseIndex]) {
                chatHistory[responseIndex].isStreaming = false;
                onMessageUpdate(responseIndex);
              }
              return;
            }
          } catch (e) {
            console.error('Error parsing event data:', e);
          }
        } else if (line.startsWith('event: error')) {
          const errorLine = lines.find(l => l.startsWith('data:'));
          if (errorLine) {
            const errorData = errorLine.slice(5).trim();
            throw new Error(extractErrorMessage(errorData));
          }
        }
      }
    }
  };

  const sendStreamingRequest = async (
    message: string,
    responseIndex: number,
    onMessageUpdate: (index: number) => void
  ): Promise<void> => {
    try {
      const chatRequest = createChatRequest(message);
      const baseUrl = getRequestBaseUrl();
      const url = `${baseUrl}/ai/chat/stream`;

      const response = await fetch(url, {
        method: 'POST',
        headers: getStreamHeaders(),
        body: JSON.stringify(chatRequest),
        signal: abortController.value?.signal,
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(extractErrorMessage(text));
      }

      await processStreamData(
        response.body!.getReader(),
        responseIndex,
        onMessageUpdate
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('An error occurred while processing the stream');
    }
  };

  // Conversation management
  const selectConversation = async (conversationId: string) => {
    if (currentConversationId.value === conversationId) return;

    chatInputRef.value?.focus();
    currentConversationId.value = conversationId;
    streamError.value = '';
    await loadConversationMessages(conversationId);
  };

  const createNewConversation = () => {
    currentConversationId.value = '';
    localStorage.removeItem('currentConversationId');
    streamError.value = '';
    chatHistory.splice(0, chatHistory.length);
    chatInputRef.value?.focus();
  };

  // Watch conversation ID changes
  watch(currentConversationId, async newId => {
    if (newId) {
      localStorage.setItem('currentConversationId', newId);
      await loadCurrentConversation(newId);
    } else {
      localStorage.removeItem('currentConversationId');
      currentConversation.value = null;
    }

    if (newId && !currentConversationId.value) {
      await loadConversations();
      currentConversationId.value = newId;
    }
  });

  const focusInput = () => {
    setTimeout(() => {
      chatInputRef.value?.focus();
    }, 500);
  };
  onMounted(focusInput);

  const initializeConversation = async () => {
    loadChatbotConfig();
    await loadLLMProviders();

    // Load saved conversation ID from localStorage
    const savedConversationId = localStorage.getItem('currentConversationId');
    if (savedConversationId) {
      await loadConversationMessages(savedConversationId);
      await loadCurrentConversation(savedConversationId);
      currentConversationId.value = savedConversationId;
    }
  };

  return {
    // Refs
    messageListRef,
    chatInputRef,

    // State
    currentConversation,
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

    // Methods
    loadConversations,
    loadConversationMessages,
    loadCurrentConversation,
    loadLLMProviders,
    loadChatbotConfig,
    saveChatbotConfig,
    resetChatbotConfig,
    selectConversation,
    createNewConversation,
    sendStreamingRequest,
    extractErrorMessage,
    initializeConversation,
  };
};

export default useAssistantConsole;
