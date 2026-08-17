<script setup lang="ts">
import { computed, onBeforeMount, ref, watch } from 'vue';
import { translate } from '@/utils';
import { ElMessage } from 'element-plus';
import { getLLMProviderItems } from '@/utils/llm';

const props = defineProps<{
  loading?: boolean;
  modelValue?: LLMProvider;
}>();

const emit =
  defineEmits<(e: 'update:modelValue', value: LLMProvider) => void>();

const t = translate;

const formRef = ref();

const showApiKey = ref(false);

const showApiBaseUrl = computed(() => {
  const { modelValue } = props;
  if (!modelValue?.type) return false;
  return ['azure-openai', 'openai-compatible'].includes(modelValue.type);
});

const showApiVersion = computed(() => {
  const { modelValue } = props;
  if (!modelValue?.type) return false;
  return ['azure-openai'].includes(modelValue.type);
});

const llmProviderItems = computed(() => {
  return getLLMProviderItems();
});

const activeLlmProviderItem = computed(() => {
  const { modelValue } = props;
  if (!modelValue?.type) return null;
  return llmProviderItems.value.find(item => item.type === modelValue.type);
});

const defaultModels = computed(() => {
  return activeLlmProviderItem.value?.defaultModels || [];
});

const defaultApiVersion = computed(() => {
  return activeLlmProviderItem.value?.defaultApiVersions?.[0] || '';
});

const customModelInput = ref('');
const customModels = computed(() => {
  const { modelValue } = props;
  return (modelValue?.models || []).filter(
    model => !defaultModels.value.includes(model)
  );
});

const addCustomModel = () => {
  const { modelValue } = props;
  if (!modelValue) return;

  if (!customModelInput.value.trim()) return;

  // If model already exists, don't add it again
  if ((modelValue.models || []).includes(customModelInput.value.trim())) {
    ElMessage.warning(t('views.system.ai.modelAlreadyExists'));
    return;
  }

  // Initialize models array if it doesn't exist
  if (!modelValue.models) {
    modelValue.models = [];
  }

  // Add the custom model
  modelValue.models.push(customModelInput.value.trim());
  customModelInput.value = '';

  emit('update:modelValue', modelValue);
};

const removeCustomModel = (model: string) => {
  const { modelValue } = props;
  if (!modelValue?.models) return;

  const index = modelValue.models.indexOf(model);
  if (index !== -1) {
    modelValue.models.splice(index, 1);
  }

  emit('update:modelValue', modelValue);
};

const isModelEnabled = (model: string) => {
  const { modelValue } = props;
  return (modelValue?.models || []).includes(model);
};

const toggleModel = (model: string) => {
  const { modelValue } = props;
  if (!modelValue) return;
  if (!modelValue.models) {
    modelValue.models = [];
  }

  const index = modelValue.models.indexOf(model);
  if (index === -1) {
    // Enable model
    modelValue.models.push(model);

    // If this is the first model being added, set it as default
    if (modelValue.models.length === 1 && !modelValue.default_model) {
      modelValue.default_model = model;
    }
  } else {
    // Disable model
    modelValue.models.splice(index, 1);

    // If default model is being disabled, update default model
    if (modelValue.default_model === model) {
      modelValue.default_model = modelValue.models.length > 0 ? modelValue.models[0] : '';
    }
  }

  emit('update:modelValue', modelValue);
};

const updateDefaultModels = () => {
  const { modelValue } = props;
  if (!modelValue) return;
  if (defaultModels.value.length > 0) {
    modelValue.models = [...defaultModels.value];
    // Set default model to first enabled model
    if (modelValue.models.length > 0 && !modelValue.default_model) {
      modelValue.default_model = modelValue.models[0];
    }
  }
};

const updateDefaultApiVersion = () => {
  const { modelValue } = props;
  if (!modelValue) return;
  if (defaultApiVersion.value) {
    modelValue.api_version = defaultApiVersion.value;
  }
};

const validate = async () => {
  const { modelValue } = props;
  if (!modelValue) return;

  // Validate the form
  await formRef.value.validate();
};

const isCreate = computed(() => {
  const { modelValue } = props;
  return !modelValue?._id;
});
watch(isCreate, () => {
  showApiKey.value = isCreate.value;
});

const onTypeChange = () => {
  const { modelValue } = props;
  if (!modelValue) return;
  if (!isCreate) return;

  modelValue.name = activeLlmProviderItem.value?.name;
  updateDefaultModels();
  updateDefaultApiVersion();

  emit('update:modelValue', modelValue);
};

onBeforeMount(() => {
  updateDefaultModels();
  updateDefaultApiVersion();
});

defineExpose({
  validate,
});

defineOptions({ name: 'ClLlmProviderForm' });
</script>

<template>
  <cl-form
    v-loading="loading"
    v-if="modelValue"
    class="form"
    ref="formRef"
    :model="modelValue"
  >
    <cl-form-item
      :label="t('views.system.ai.llmProvider')"
      :span="4"
      prop="type"
      required
    >
      <el-select
        v-model="modelValue.type"
        :placeholder="t('views.system.ai.llmProvider')"
        @change="onTypeChange"
      >
        <el-option
          v-for="{ type, name, icon } in llmProviderItems"
          :key="type"
          :value="type"
        >
          <span style="margin-right: 5px">
            <cl-icon :icon="icon" />
          </span>
          <span>{{ name }}</span>
        </el-option>
        <template #label>
          <span style="margin-right: 5px">
            <cl-icon :icon="activeLlmProviderItem?.icon" />
          </span>
          <span>{{ activeLlmProviderItem?.name }}</span>
        </template>
      </el-select>
    </cl-form-item>
    <cl-form-item
      :label="t('views.system.ai.name')"
      :span="4"
      prop="name"
      required
    >
      <el-input
        v-model="modelValue.name"
        :placeholder="t('views.system.ai.name')"
      />
    </cl-form-item>
    <cl-form-item
      :label="t('views.system.ai.apiKey')"
      :span="4"
      prop="api_key"
      required
    >
      <el-input
        v-model="modelValue.api_key"
        :placeholder="t('views.system.ai.apiKey')"
        :type="showApiKey ? 'text' : 'password'"
      >
        <template #suffix>
          <div
            style="cursor: pointer"
            @click.prevent="() => (showApiKey = !showApiKey)"
          >
            <cl-icon :icon="showApiKey ? ['fa', 'eye-slash'] : ['fa', 'eye']" />
          </div>
        </template>
      </el-input>
    </cl-form-item>
    <cl-form-item
      v-if="showApiBaseUrl"
      :label="t('views.system.ai.apiBaseUrl')"
      :span="4"
      prop="api_base_url"
      required
    >
      <el-input
        v-model="modelValue.api_base_url"
        :placeholder="t('views.system.ai.apiBaseUrl')"
      />
    </cl-form-item>
    <cl-form-item
      v-if="showApiVersion"
      :label="t('views.system.ai.apiVersion')"
      :span="4"
      prop="api_version"
      required
    >
      <el-input
        v-model="modelValue.api_version"
        :placeholder="t('views.system.ai.apiVersion')"
      />
    </cl-form-item>
    <cl-form-item :label="t('views.system.ai.models')" :span="4" prop="models">
      <div class="models-section">
        <!-- Default models from provider -->
        <div v-if="defaultModels.length > 0" class="default-models">
          <div class="model-list">
            <el-checkbox
              v-for="model in defaultModels"
              :key="model"
              :model-value="isModelEnabled(model)"
              @change="() => toggleModel(model)"
              class="model-checkbox"
            >
              <span>{{ model }}</span>
              <span class="default-model-badge" v-if="model === modelValue.default_model">
                ({{ t('common.mode.default') }})
              </span>
            </el-checkbox>
          </div>
        </div>

        <!-- Custom models -->
        <div class="custom-models">
          <!-- Add custom model input -->
          <div class="add-model">
            <el-input
              v-model="customModelInput"
              :placeholder="t('views.system.ai.addCustomModel')"
              @keyup.enter.prevent="addCustomModel"
            >
              <template #append>
                <cl-fa-icon-button
                  @click="addCustomModel"
                  :icon="['fa', 'plus']"
                />
              </template>
            </el-input>
          </div>

          <!-- Custom model list -->
          <div v-if="customModels.length > 0" class="model-list">
            <div
              v-for="model in customModels"
              :key="model"
              class="custom-model-item"
            >
              <el-checkbox
                :model-value="isModelEnabled(model)"
                @change="() => toggleModel(model)"
              >
                {{ model }}
              </el-checkbox>
              <span class="delete-btn" @click="removeCustomModel(model)">
                <cl-icon :icon="['fa', 'times']" />
              </span>
            </div>
          </div>
          <div v-else class="no-custom-models">
            {{ t('views.system.ai.noCustomModels') }}
          </div>
        </div>
      </div>
    </cl-form-item>

    <cl-form-item :label="t('views.system.ai.defaultModel')" :span="4" prop="default_model">
      <el-select
        v-model="modelValue.default_model"
        :placeholder="t('views.system.ai.defaultModel')"
        :disabled="!modelValue.models || modelValue.models.length === 0"
      >
        <el-option
          v-for="model in modelValue.models || []"
          :key="model"
          :label="model"
          :value="model"
        />
      </el-select>
    </cl-form-item>
  </cl-form>
</template>

<style scoped>
.models-section {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .section-title {
    font-weight: 600;
    margin-bottom: 8px;
  }

  .model-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    .model-checkbox {
      min-width: 120px;
    }
  }

  .custom-models {
    .add-model {
      margin-bottom: 12px;
      max-width: 400px;
    }

    .custom-model-item {
      display: flex;
      align-items: center;
      margin-bottom: 4px;

      .el-checkbox {
        flex-grow: 1;
      }

      .delete-btn {
        margin-left: 8px;
        min-width: 32px;
        cursor: pointer;
        color: var(--el-text-color-secondary);
      }
    }

    .no-custom-models {
      color: var(--el-text-color-secondary);
      font-style: italic;
    }
  }

  .default-model-badge {
    margin-left: 5px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}
</style>
