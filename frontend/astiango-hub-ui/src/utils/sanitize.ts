import DOMPurify from 'dompurify';

const ALLOWED_HTML_TAGS = [
  'a',
  'b',
  'blockquote',
  'br',
  'code',
  'div',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'hr',
  'i',
  'li',
  'ol',
  'p',
  'pre',
  'span',
  'strong',
  'table',
  'tbody',
  'td',
  'th',
  'thead',
  'tr',
  'ul',
];

const ALLOWED_HTML_ATTRIBUTES = ['class', 'href', 'rel', 'target', 'title'];
const SAFE_URL = /^(?:(?:https?|mailto):|\/(?!\/)|#)/i;

DOMPurify.addHook('afterSanitizeAttributes', node => {
  if (node.tagName === 'A' && node.getAttribute('target') === '_blank') {
    node.setAttribute('rel', 'noopener noreferrer');
  }
});

/** Sanitizes all rich text that may originate from users, crawled data or APIs. */
export const sanitizeHtml = (value?: unknown): string => {
  return DOMPurify.sanitize(String(value ?? ''), {
    ALLOWED_TAGS: ALLOWED_HTML_TAGS,
    ALLOWED_ATTR: ALLOWED_HTML_ATTRIBUTES,
    ALLOWED_URI_REGEXP: SAFE_URL,
    ALLOW_DATA_ATTR: false,
    FORBID_ATTR: ['style'],
    FORBID_TAGS: [
      'button',
      'embed',
      'form',
      'iframe',
      'input',
      'math',
      'object',
      'script',
      'style',
      'svg',
    ],
  });
};

/** SVG assets are never trusted just because they are loaded from the bundle. */
export const sanitizeSvg = (value?: unknown): string => {
  return DOMPurify.sanitize(String(value ?? ''), {
    USE_PROFILES: { html: false, svg: true, svgFilters: false },
    ALLOW_DATA_ATTR: false,
    FORBID_ATTR: ['href', 'xlink:href'],
    FORBID_TAGS: [
      'animate',
      'animateMotion',
      'animateTransform',
      'foreignObject',
      'iframe',
      'script',
      'set',
    ],
  });
};

/** Returns a URL suitable for href/src bindings, or an inert URL for unsafe input. */
export const sanitizeUrl = (value?: unknown): string => {
  const url = String(value ?? '').trim();
  return SAFE_URL.test(url) ? url : 'about:blank';
};
