# AstianGO Hub-UI

This is the UI components and modules to support the frontend development
for [AstianGO Hub](https://github.com/goastian/astiango-hub).

This package is part of an independent fork. Upstream attribution remains in the bundled BSD-3-Clause LICENSE and the repository NOTICE.

## How to Install

Use `npm` or `yarn` to install `astiango-hub-ui`.

```
# npm
npm install astiango-hub-ui -S

# or use yarn
yarn add astiango-hub-ui -S
```

## How to Use

It is similar to [Element-Plus](https://github.com/element-plus/element-plus), you can import components from
AstianGO Hub-UI. AstianGO Hub-UI is built based on Element-Plus so that you can comfortably use it with Element-Plus.

### Use Globally Installed Components

Below is an example of entry file (main.ts) using AstianGO Hub-UI globally.

```ts
import {createApp} from 'vue';
import AstianGoHubUI from 'astiango-hub-ui';

const app = createApp(App);
app
  .use(AstianGoHubUI)  // install globally
  .mount('#app');
```

Below is an example of using globally installed AstianGO Hub-UI in a Vue 3 component.

```vue
<template>
  <cl-form :model="form">
    <cl-form-item :span="2" label="Key" prop="key" required>
      <el-input v-model="form.key"/>
    </cl-form-item>
    <cl-form-item :span="2" label="Value" prop="value" required>
      <el-input v-model="form.value"/>
    </cl-form-item>
  </cl-form>
</template>

<script setup lang="ts">
import {ref} from 'vue';

const form = ref({
  key: 'test-key',
  value: 'test-value',
});
</script>
```

### Use Standalone Components

Below is an example of using standalone components in a Vue 3 component.

```vue

<template>
  <cl-form :model="form">
    <cl-form-item :span="2" label="Key" prop="key" required>
      <el-input v-model="form.key"/>
    </cl-form-item>
    <cl-form-item :span="2" label="Value" prop="value" required>
      <el-input v-model="form.value"/>
    </cl-form-item>
  </cl-form>
</template>

<script lang="ts">
import {defineComponent, ref} from 'vue';
import {ClForm, ClFormItem} from 'astiango-hub-ui';

export default defineComponent({
  components: {
    ClForm,
    ClFormItem,
  },
  setup() {
    const form = ref({
      key: 'test-key',
      value: 'test-value',
    });
    return {
      form,
    };
  },
});
</script>
```

### Use Web Application

AstianGO Hub-UI has a built-in web application for AstianGO Hub frontend. You can simply use it to start AstianGO Hub frontend Vue 3
SPA.

```ts
// index.ts or index.js or other entry file
import {createApp} from 'astiango-hub-ui';

createApp();
```

And that's it! After you build or start serving it, you can view the AstianGO Hub frontend web application in the browser.

## Development

### Add a new model

Related directories:

- `src/views`
- `src/components`
- `src/store`
- `src/router`
- `src/i18n`
- `src/services`
- `src/interfaces/i18n`
