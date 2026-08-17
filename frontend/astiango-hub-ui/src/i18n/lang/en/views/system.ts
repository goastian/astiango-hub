const system: LViewsSystem = {
  menuItems: {
    customize: 'Customization',
    dependency: 'Dependencies',
    environment: 'Environment',
    ai: 'AI Assistant',
    models: 'AI Models',
  },
  ai: {
    llmProvider: 'LLM Provider',
    name: 'Name',
    enabled: 'Enabled',
    apiKey: 'API Key',
    deploymentName: 'Deployment Name',
    apiBaseUrl: 'API Base URL',
    apiVersion: 'API Version',
    models: 'Models',
    defaultModel: 'Default Model',
    addCustomModel: 'Add custom model',
    noCustomModels: 'No custom models added',
    modelAlreadyExists: 'Model already exists',
    temperature: 'Temperature',
    maxTokens: 'Max Tokens',
    topP: 'Top P',
    unset: 'Unset',
    disabled: 'Disabled',
    actions: {
      new: {
        llmProvider: 'New LLM Provider',
      },
      edit: {
        llmProvider: 'Edit LLM Provider',
      },
    },
  },
  customize: {
    customTitle: 'Custom Site Title',
    showCustomTitle: 'Show Custom Site Title',
    customLogo: 'Custom Logo',
    showCustomLogo: 'Show Custom Site Logo',
    hidePlatformVersion: 'Hide Platform Version',
    uploadLogoTip:
      'Support image formats including JPG, PNG, and SVG. File size should be less than 1MB.',
    uploadLogoErrors: {
      invalidFileType: 'Invalid file type',
      fileSizeExceeded: 'File size exceeded',
    },
  },
  dependency: {
    autoInstall: 'Auto Install',
  },
};

export default system;
