import { createApp, h, ref, resolveComponent } from 'vue';
import ElementPlus from 'element-plus';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import { installer as AstianGoHubUI } from '@/package';
import { getI18n } from '@/i18n';
import { getStore } from '@/store';
import { setGlobalLang } from '@/utils/i18n';
import { auth, safeHtml } from '@/directives';
import clickOutsideDirective from '@/directives/click-outside/clickOutside';
import 'normalize.css/normalize.css';
import 'element-plus/theme-chalk/index.css';
import '@/styles/index.css';

const markdownContent = [
  '## Lexical 0.49 regression',
  '',
  'This editor uses the upgraded Lexical packages.',
  '',
  '- Lists, links and formatting remain editable',
  '- State serialization is covered by the automated regression test',
  '',
  '> The toolbar below is the production editor component.',
  '',
  '[Lexical documentation](https://lexical.dev)',
].join('\n');

const app = createApp({
  setup() {
    const content = ref<RichTextPayload>({
      richTextContent: '',
      richTextContentJson: '',
    });

    return () =>
      h('main', { class: 'lexical-regression-page' }, [
        h('section', { class: 'lexical-regression-card' }, [
          h('p', { class: 'lexical-regression-eyebrow' }, 'Regression check'),
          h('h1', 'Lexical 0.49 editor'),
          h(
            'p',
            'Type, select text and use the toolbar to validate the production component.'
          ),
          h('div', { class: 'lexical-regression-editor' }, [
            h(resolveComponent('cl-lexical-editor'), {
              modelValue: content.value,
              'onUpdate:modelValue': value => (content.value = value),
              markdownContent,
            }),
          ]),
        ]),
      ]);
  },
});

setGlobalLang('en');
app.use(ElementPlus as any);
app.use(AstianGoHubUI as any);
app.use(getStore() as any);
app.use(getI18n() as any);
app.component('font-awesome-icon', FontAwesomeIcon);
app.directive('auth', auth as any);
app.directive('click-outside', clickOutsideDirective);
app.directive('safe-html', safeHtml);
app.mount('#app');

const style = document.createElement('style');
style.textContent = `
  body { background: #f3f6fb; }
  .lexical-regression-page { min-height: 100vh; padding: 48px; }
  .lexical-regression-card { max-width: 920px; min-height: 620px; margin: 0 auto; padding: 32px; border-radius: 14px; background: #fff; box-shadow: 0 12px 36px rgba(24, 39, 75, .12); }
  .lexical-regression-eyebrow { margin: 0; color: #409eff; font-weight: 600; text-transform: uppercase; letter-spacing: .08em; font-size: 12px; }
  .lexical-regression-card h1 { margin: 8px 0; color: #172033; }
  .lexical-regression-card > p { color: #5b6475; }
  .lexical-regression-editor { height: 420px; margin-top: 28px; border: 1px solid #dcdfe6; border-radius: 8px; overflow: hidden; }
`;
document.head.append(style);
