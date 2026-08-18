import type { Directive } from 'vue';
import { sanitizeHtml, sanitizeSvg } from '@/utils/sanitize';

const render = (el: HTMLElement, value: unknown, svg = false) => {
  el.innerHTML = svg ? sanitizeSvg(value) : sanitizeHtml(value);
};

const safeHtmlDirective: Directive<HTMLElement, unknown> = {
  mounted(el, binding) {
    render(el, binding.value, Boolean(binding.modifiers.svg));
  },
  updated(el, binding) {
    render(el, binding.value, Boolean(binding.modifiers.svg));
  },
};

export default safeHtmlDirective;
