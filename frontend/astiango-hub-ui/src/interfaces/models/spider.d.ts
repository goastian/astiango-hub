export declare global {
  interface Spider extends BaseModel {
    name?: string;
    display_name?: string;
    spider_type?: string;
    cmd?: string;
    param?: string;
    priority?: number;
    col_id?: string;
    col_name?: string;
    db_name?: string;
    database_id?: string;
    mode?: TaskMode;
    node_ids?: string[];
    node_tags?: string[];
    project_id?: string;
    project_name?: string;
    description?: string;
    update_ts?: string;
    create_ts?: string;
    incremental_sync?: boolean;
    auto_install?: boolean;
    git_id?: string;
    git_root_path?: string;
    template?: SpiderTemplateName;
    template_params?: SpiderTemplateParams;

    // associated data
    stat?: SpiderStat;
    last_task?: Task;
    project?: Project;
    git?: Git;
    database?: Database;
  }

  interface SpiderStat {
    _id: number;
    tasks: number;
    results: number;
    wait_duration: number;
    runtime_duration: number;
    total_duration: number;
    average_wait_duration: number;
    average_runtime_duration: number;
    average_total_duration: number;
  }

  interface SpiderRunOptions {
    mode?: string;
    node_ids?: string[];
    node_tags?: string[];
    cmd?: string;
    param?: string;
    schedule_id?: string;
    priority?: number;
  }

  type SpiderTemplateName =
    | 'scrapy'
    | 'scrapy-redis'
    | 'bs4'
    | 'selenium'
    | 'drission-page'
    | 'pyppeteer'
    | 'crawlee-python'
    | 'puppeteer'
    | 'playwright'
    | 'cheerio'
    | 'crawlee'
    | 'colly'
    | 'goquery'
    | 'jsoup'
    | 'webmagic'
    | 'xxl-crawler';

  interface SpiderTemplateParams {
    spider_name?: string;
    start_urls?: string;
    domains?: string;
  }

  interface SpiderTemplateGroup {
    lang: DependencyLang;
    label: string;
    icon?: Icon;
    templates: SpiderTemplate[];
  }

  interface SpiderTemplate {
    name: SpiderTemplateName;
    label: string;
    icon?: Icon;
    cmd: string;
    params?: SpiderTemplateParams;
    doc_url?: string;
    doc_label?: string;
  }
}
