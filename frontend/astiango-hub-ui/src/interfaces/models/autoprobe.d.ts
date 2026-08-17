export declare global {
  interface AutoProbe extends BaseModel {
    name?: string;
    url?: string;
    query?: string;
    last_task_id?: string;
    last_task_status?: AutoProbeTaskStatus;
    last_task_error?: string;
    default_task_id?: string;
    run_on_create?: boolean;
    page_pattern?: PagePattern;
    page_data?: PageData;
    viewport?: PageViewPort;
  }

  // V2 AutoProbe interface that matches backend AutoProbeV2
  interface AutoProbeV2 extends BaseModel {
    name?: string;
    description?: string;
    url?: string;
    query?: string;
    last_task_id?: string;
    last_task_status?: AutoProbeTaskStatus;
    last_task_error?: string;
    default_task_id?: string;
    run_on_create?: boolean;
    page_pattern?: PagePatternV2;
    page_data?: PageData;
    viewport?: PageViewPort;
  }

  // Hierarchical pattern structure for V2
  interface PatternV2 extends BaseModel {
    name: string;
    type: PatternTypeV2;
    selector_type?: SelectorType;
    selector?: string;
    is_absolute_selector?: boolean;
    extraction_type?: ExtractType;
    attribute_name?: string;
    children?: PatternV2[];
    parent_id?: string;
  }

  type PatternTypeV2 = 'field' | 'list' | 'list-item' | 'action' | 'content';

  interface PatternDataV2 extends BaseModel {
    task_id: string;
    pattern_id: string;
    data?: any;
  }

  interface PagePatternV2 {
    name: string;
    children?: PatternV2[];
  }

  interface AutoProbeNavItemV2<T = any> extends NavItem<T> {
    name?: string;
    type?: PatternTypeV2;
    children?: AutoProbeNavItemV2[];
    parent?: AutoProbeNavItemV2;
  }

  type AutoProbeTaskStatus =
    | 'pending'
    | 'running'
    | 'completed'
    | 'failed'
    | 'cancelled';

  type SelectorType = 'css' | 'xpath' | 'regex';
  type ExtractType = 'text' | 'attribute' | 'html';

  interface BaseSelector {
    name: string;
    selector_type: SelectorType;
    selector: string;
  }

  interface FieldRule extends BaseSelector {
    extraction_type: ExtractType;
    attribute_name?: string;
    default_value?: string;
  }

  interface ItemPattern {
    fields?: FieldRule[];
    lists?: ListRule[];
  }

  interface ListRule {
    name: string;
    list_selector_type: SelectorType;
    list_selector: string;
    item_selector_type: SelectorType;
    item_selector: string;
    item_pattern: ItemPattern;
  }

  type PaginationRule = BaseSelector;

  interface PagePattern {
    name: string;
    fields?: FieldRule[];
    lists?: ListRule[];
    pagination?: PaginationRule;
  }

  type PageData = Record<string, string | number | boolean | PageData[]>;

  interface AutoProbeTask extends BaseModel {
    autoprobe_id: string;
    url?: string;
    query?: string;
    status: AutoProbeTaskStatus;
    error?: string;
    html?: string;
    // page_pattern?: PagePattern;
    page_pattern?: PagePatternV2;
    page_data?: PageData;
    page_elements?: PageElement[];
    provider_id?: string;
    model?: string;
    usage?: LLMResponseUsage;
  }

  type AutoProbeItemType = 'page_pattern' | 'list' | 'field' | 'pagination';

  interface AutoProbeNavItem<T = any> extends NavItem<T> {
    name?: string;
    type?: AutoProbeItemType;
    rule?: ListRule | FieldRule | PaginationRule;
    children?: AutoProbeNavItem[];
    parent?: AutoProbeNavItem;
    fieldCount?: number;
  }

  interface AutoProbeResults {
    data?: PageData | PageData[];
    fields?: AutoProbeNavItem[];
    activeField?: AutoProbeNavItem;
  }

  interface PageViewPort {
    width: number;
    height: number;
  }

  interface ElementCoordinates {
    top: number;
    left: number;
    width: number;
    height: number;
  }

  type PageElementType = 'list' | 'list-item' | 'field' | 'pagination';

  interface PageElement {
    name: string;
    type: PageElementType;
    coordinates: ElementCoordinates;
    children?: PageElement[];
    active?: boolean;
  }

  interface PagePreviewResult {
    screenshot_base64: string;
    page_elements: PageElement[];
  }

  type ViewPortValue = 'pc-normal' | 'pc-wide' | 'pc-small';

  interface ViewPortSelectOption extends SelectOption<ViewPortValue> {
    viewport: PageViewPort;
  }
}
