import { translate } from '@/utils';

const t = translate;

export const getIconBySelectorType = (selectorType: SelectorType): Icon => {
  switch (selectorType) {
    case 'css':
      return ['fab', 'css'];
    case 'xpath':
      return ['fa', 'code'];
    case 'regex':
      return ['fa', 'search'];
  }
};

export const getIconByExtractType = (extractType?: ExtractType): Icon => {
  switch (extractType) {
    case 'text':
      return ['fa', 'file-alt'];
    case 'attribute':
      return ['fa', 'tag'];
    case 'html':
      return ['fa', 'file-code'];
    default:
      return ['fa', 'question'];
  }
};

export const getIconByItemType = (itemType?: AutoProbeItemType): Icon => {
  switch (itemType) {
    case 'page_pattern':
      return ['fa', 'network-wired'];
    case 'field':
      return ['fa', 'tag'];
    case 'list':
      return ['fa', 'list'];
    case 'pagination':
      return ['fa', 'ellipsis-h'];
    default:
      return ['fa', 'question'];
  }
};

export const getIconByPageElementType = (itemType?: PageElementType): Icon => {
  switch (itemType) {
    case 'field':
      return ['fa', 'tag'];
    case 'list':
      return ['fa', 'list'];
    case 'list-item':
      return ['fa', 'bars'];
    case 'pagination':
      return ['fa', 'ellipsis-h'];
    default:
      return ['fa', 'question'];
  }
};

export const getViewPortOptions = () => {
  return [
    {
      label: t('components.autoprobe.form.viewports.pc.normal'),
      value: 'pc-normal',
      viewport: { width: 1280, height: 800 },
    },
    {
      label: t('components.autoprobe.form.viewports.pc.wide'),
      value: 'pc-wide',
      viewport: { width: 1920, height: 1080 },
    },
    {
      label: t('components.autoprobe.form.viewports.pc.small'),
      value: 'pc-small',
      viewport: { width: 1024, height: 768 },
    },
  ] as ViewPortSelectOption[];
};
