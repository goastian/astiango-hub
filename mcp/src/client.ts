import axios, { AxiosInstance, AxiosResponse } from 'axios';
import { isReadOnlyMethod } from './security.js';

export interface AstianGoHubConfig {
  url: string;
  apiToken?: string;
  timeout?: number;
}

export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  total?: number;
}

export interface PaginationParams {
  page?: number;
  size?: number;
}

export interface SpiderTemplateParams {
  project_name?: string;
  spider_name?: string;
  start_urls?: string;
  allowed_domains?: string;
}

export interface Spider {
  _id: string;
  name: string;
  col_id?: string;
  col_name?: string;
  db_name?: string;
  description?: string;
  database_id?: string;
  project_id?: string;
  mode?: string; // random, all, selected-nodes
  node_ids?: string[];
  git_id?: string;
  git_root_path?: string;
  template?: string;
  template_params?: SpiderTemplateParams;
  cmd: string;
  param?: string;
  priority?: number; // 1-10, default 5
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface Task {
  _id: string;
  spider_id: string;
  status: string; // pending, assigned, running, finished, error, cancelled, abnormal
  node_id?: string;
  cmd: string;
  param?: string;
  error?: string;
  pid?: number;
  schedule_id?: string;
  mode?: string;
  priority?: number;
  node_ids?: string[];
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface Node {
  _id: string;
  key?: string;
  name: string;
  ip: string;
  mac: string;
  hostname: string;
  description?: string;
  is_master: boolean;
  status: string;
  enabled?: boolean;
  active?: boolean;
  active_at?: Date;
  current_runners?: number;
  max_runners?: number;
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface Schedule {
  _id: string;
  name: string;
  description?: string;
  spider_id: string;
  cron: string;
  entry_id?: number; // cron entry ID
  cmd?: string;
  param?: string;
  mode?: string;
  node_ids?: string[];
  priority?: number;
  enabled: boolean;
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface Project {
  _id: string;
  name: string;
  description?: string;
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface Database {
  _id: string;
  name: string;
  description?: string;
  data_source: string;
  host: string;
  port: number;
  uri?: string;
  database?: string;
  username?: string;
  status: string;
  error?: string;
  active: boolean;
  active_at?: Date;
  is_default?: boolean;
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface Git {
  _id: string;
  url: string;
  name: string;
  auth_type?: string;
  username?: string;
  current_branch?: string;
  status: string;
  error?: string;
  created_at?: Date;
  updated_at?: Date;
  created_by?: string;
  updated_by?: string;
}

export interface SpiderStat {
  _id: string;
  last_task_id?: string;
  tasks: number;
  results: number;
  wait_duration?: number; // in seconds
  runtime_duration?: number; // in seconds  
  total_duration?: number; // in seconds
  average_wait_duration?: number;
  average_runtime_duration?: number;
  average_total_duration?: number;
  created_at?: Date;
  updated_at?: Date;
}

export interface TaskStat {
  _id: string;
  started_at?: Date;
  ended_at?: Date;
  wait_duration?: number; // in milliseconds
  runtime_duration?: number; // in milliseconds
  total_duration?: number; // in milliseconds
  result_count: number;
  created_at?: Date;
  updated_at?: Date;
}

export class AstianGoHubClient {
  private client: AxiosInstance;
  private baseURL: string;

  constructor(
    apiUrl: string,
    apiToken?: string,
    timeout: number = 30000,
    allowMutations: boolean = false
  ) {
    this.baseURL = apiUrl.replace(/\/$/, ''); // Remove trailing slash

    this.client = axios.create({
      baseURL: this.baseURL,
      timeout,
      headers: {
        'Content-Type': 'application/json',
        ...(apiToken && { Authorization: `Bearer ${apiToken}` }),
      },
    });

    this.client.interceptors.request.use(config => {
      if (!allowMutations && !isReadOnlyMethod(config.method)) {
        return Promise.reject(
          new Error('MCP is read-only. Set ASTIANGO_MCP_ALLOW_MUTATIONS=true to allow write operations.')
        );
      }
      return config;
    });

    // Add response interceptor for error handling
    this.client.interceptors.response.use(
      (response: AxiosResponse) => response,
      error => {
        const message = error.response?.data?.error || error.message;
        throw new Error(`AstianGO Hub API Error: ${message}`);
      }
    );
  }

  // Spiders
  async getSpiders(params?: PaginationParams): Promise<ApiResponse<Spider[]>> {
    const response = await this.client.get('/spiders', { params });
    return response.data;
  }

  async getSpider(id: string): Promise<ApiResponse<Spider>> {
    const response = await this.client.get(`/spiders/${id}`);
    return response.data;
  }

  async createSpider(spider: Partial<Spider>): Promise<ApiResponse<Spider>> {
    const response = await this.client.post('/spiders', spider);
    return response.data;
  }

  async updateSpider(
    id: string,
    spider: Partial<Spider>
  ): Promise<ApiResponse<Spider>> {
    const response = await this.client.put(`/spiders/${id}`, spider);
    return response.data;
  }

  async deleteSpider(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/spiders/${id}`);
    return response.data;
  }

  async runSpider(
    id: string,
    params?: {
      cmd?: string;
      param?: string;
      priority?: number;
      mode?: string;
      node_ids?: string[];
    }
  ): Promise<ApiResponse<string[]>> {
    const response = await this.client.post(`/spiders/${id}/run`, params);
    return response.data;
  }

  async getSpiderFiles(id: string, path?: string): Promise<ApiResponse<any[]>> {
    const params = path ? { path } : {};
    const response = await this.client.get(`/spiders/${id}/files`, { params });
    return response.data;
  }

  async getSpiderFileContent(
    id: string,
    path: string
  ): Promise<ApiResponse<string>> {
    const response = await this.client.get(`/spiders/${id}/files/content`, {
      params: { path },
    });
    return response.data;
  }

  async saveSpiderFile(
    id: string,
    path: string,
    content: string
  ): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/spiders/${id}/files/save`, {
      path,
      content,
    });
    return response.data;
  }

  // Tasks
  async getTasks(
    params?: PaginationParams & { spider_id?: string; status?: string }
  ): Promise<ApiResponse<Task[]>> {
    const response = await this.client.get('/tasks', { params });
    return response.data;
  }

  async getTask(id: string): Promise<ApiResponse<Task>> {
    const response = await this.client.get(`/tasks/${id}`);
    return response.data;
  }

  async cancelTask(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/tasks/${id}/cancel`);
    return response.data;
  }

  async restartTask(id: string): Promise<ApiResponse<string[]>> {
    const response = await this.client.post(`/tasks/${id}/restart`);
    return response.data;
  }

  async deleteTask(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/tasks/${id}`);
    return response.data;
  }

  async getTaskLogs(
    id: string,
    params?: { page?: number; size?: number }
  ): Promise<ApiResponse<string[]>> {
    const response = await this.client.get(`/tasks/${id}/logs`, { params });
    return response.data;
  }

  async getTaskResults(
    id: string,
    params?: PaginationParams
  ): Promise<ApiResponse<any[]>> {
    const response = await this.client.get(`/tasks/${id}/results`, { params });
    return response.data;
  }

  // Nodes
  async getNodes(params?: PaginationParams): Promise<ApiResponse<Node[]>> {
    const response = await this.client.get('/nodes', { params });
    return response.data;
  }

  async getNode(id: string): Promise<ApiResponse<Node>> {
    const response = await this.client.get(`/nodes/${id}`);
    return response.data;
  }

  async updateNode(id: string, node: Partial<Node>): Promise<ApiResponse<Node>> {
    const response = await this.client.put(`/nodes/${id}`, node);
    return response.data;
  }

  async enableNode(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/nodes/${id}/enable`);
    return response.data;
  }

  async disableNode(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/nodes/${id}/disable`);
    return response.data;
  }

  // Schedules
  async getSchedules(
    params?: PaginationParams
  ): Promise<ApiResponse<Schedule[]>> {
    const response = await this.client.get('/schedules', { params });
    return response.data;
  }

  async getSchedule(id: string): Promise<ApiResponse<Schedule>> {
    const response = await this.client.get(`/schedules/${id}`);
    return response.data;
  }

  async createSchedule(
    schedule: Partial<Schedule>
  ): Promise<ApiResponse<Schedule>> {
    const response = await this.client.post('/schedules', schedule);
    return response.data;
  }

  async updateSchedule(
    id: string,
    schedule: Partial<Schedule>
  ): Promise<ApiResponse<Schedule>> {
    const response = await this.client.put(`/schedules/${id}`, schedule);
    return response.data;
  }

  async deleteSchedule(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/schedules/${id}`);
    return response.data;
  }

  async enableSchedule(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/schedules/${id}/enable`);
    return response.data;
  }

  async disableSchedule(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/schedules/${id}/disable`);
    return response.data;
  }

  // Projects
  async getProjects(params?: PaginationParams): Promise<ApiResponse<Project[]>> {
    const response = await this.client.get('/projects', { params });
    return response.data;
  }

  async getProject(id: string): Promise<ApiResponse<Project>> {
    const response = await this.client.get(`/projects/${id}`);
    return response.data;
  }

  async createProject(project: Partial<Project>): Promise<ApiResponse<Project>> {
    const response = await this.client.post('/projects', project);
    return response.data;
  }

  async updateProject(id: string, project: Partial<Project>): Promise<ApiResponse<Project>> {
    const response = await this.client.put(`/projects/${id}`, project);
    return response.data;
  }

  async deleteProject(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/projects/${id}`);
    return response.data;
  }

  // Databases
  async getDatabases(params?: PaginationParams): Promise<ApiResponse<Database[]>> {
    const response = await this.client.get('/databases', { params });
    return response.data;
  }

  async getDatabase(id: string): Promise<ApiResponse<Database>> {
    const response = await this.client.get(`/databases/${id}`);
    return response.data;
  }

  async createDatabase(database: Partial<Database>): Promise<ApiResponse<Database>> {
    const response = await this.client.post('/databases', database);
    return response.data;
  }

  async updateDatabase(id: string, database: Partial<Database>): Promise<ApiResponse<Database>> {
    const response = await this.client.put(`/databases/${id}`, database);
    return response.data;
  }

  async deleteDatabase(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/databases/${id}`);
    return response.data;
  }

  async testDatabaseConnection(id: string): Promise<ApiResponse<boolean>> {
    const response = await this.client.post(`/databases/${id}/test`);
    return response.data;
  }

  // Git repositories
  async getGitRepos(params?: PaginationParams): Promise<ApiResponse<Git[]>> {
    const response = await this.client.get('/gits', { params });
    return response.data;
  }

  async getGitRepo(id: string): Promise<ApiResponse<Git>> {
    const response = await this.client.get(`/gits/${id}`);
    return response.data;
  }

  async createGitRepo(git: Partial<Git>): Promise<ApiResponse<Git>> {
    const response = await this.client.post('/gits', git);
    return response.data;
  }

  async updateGitRepo(id: string, git: Partial<Git>): Promise<ApiResponse<Git>> {
    const response = await this.client.put(`/gits/${id}`, git);
    return response.data;
  }

  async deleteGitRepo(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/gits/${id}`);
    return response.data;
  }

  async pullGitRepo(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/gits/${id}/pull`);
    return response.data;
  }

  async cloneGitRepo(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/gits/${id}/clone`);
    return response.data;
  }

  // Statistics
  async getSpiderStats(id: string): Promise<ApiResponse<SpiderStat>> {
    const response = await this.client.get(`/spiders/${id}/stats`);
    return response.data;
  }

  async getTaskStats(id: string): Promise<ApiResponse<TaskStat>> {
    const response = await this.client.get(`/tasks/${id}/stats`);
    return response.data;
  }

  // Health check
  async healthCheck(): Promise<boolean> {
    try {
      const response = await this.client.get('/health');
      return response.status === 200;
    } catch {
      return false;
    }
  }

  // AI/LLM Features
  async getLLMProviders(params?: PaginationParams): Promise<ApiResponse<any[]>> {
    const response = await this.client.get('/ai/llm/providers', { params });
    return response.data;
  }

  async getLLMProvider(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/llm/providers/${id}`);
    return response.data;
  }

  async createLLMProvider(provider: any): Promise<ApiResponse<any>> {
    const response = await this.client.post('/ai/llm/providers', { data: provider });
    return response.data;
  }

  async updateLLMProvider(id: string, provider: any): Promise<ApiResponse<any>> {
    const response = await this.client.put(`/ai/llm/providers/${id}`, { data: provider });
    return response.data;
  }

  async deleteLLMProvider(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/ai/llm/providers/${id}`);
    return response.data;
  }

  // Chat Conversations
  async getConversations(params?: PaginationParams & { filter?: string }): Promise<ApiResponse<any[]>> {
    const response = await this.client.get('/ai/conversations', { params });
    return response.data;
  }

  async getConversation(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/conversations/${id}`);
    return response.data;
  }

  async createConversation(conversation: any): Promise<ApiResponse<any>> {
    const response = await this.client.post('/ai/conversations', { data: conversation });
    return response.data;
  }

  async updateConversation(id: string, conversation: any): Promise<ApiResponse<any>> {
    const response = await this.client.put(`/ai/conversations/${id}`, { data: conversation });
    return response.data;
  }

  async deleteConversation(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/ai/conversations/${id}`);
    return response.data;
  }

  async getConversationMessages(id: string): Promise<ApiResponse<any[]>> {
    const response = await this.client.get(`/ai/conversations/${id}/messages`);
    return response.data;
  }

  async getChatMessage(conversationId: string, messageId: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/conversations/${conversationId}/messages/${messageId}`);
    return response.data;
  }

  // AutoProbe V2
  async getAutoProbesV2(params?: PaginationParams & { filter?: string }): Promise<ApiResponse<any[]>> {
    const response = await this.client.get('/ai/autoprobes', { params });
    return response.data;
  }

  async getAutoProbeV2(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/autoprobes/${id}`);
    return response.data;
  }

  async createAutoProbeV2(autoprobe: any): Promise<ApiResponse<any>> {
    const response = await this.client.post('/ai/autoprobes', { data: autoprobe });
    return response.data;
  }

  async updateAutoProbeV2(id: string, autoprobe: any): Promise<ApiResponse<any>> {
    const response = await this.client.patch(`/ai/autoprobes/${id}`, { data: autoprobe });
    return response.data;
  }

  async deleteAutoProbeV2(id: string): Promise<ApiResponse<void>> {
    const response = await this.client.delete(`/ai/autoprobes/${id}`);
    return response.data;
  }

  async runAutoProbeV2Task(id: string, params?: { query?: string; view_port?: any }): Promise<ApiResponse<any>> {
    const response = await this.client.post(`/ai/autoprobes/${id}/tasks`, params);
    return response.data;
  }

  async getAutoProbeV2Tasks(id: string, params?: PaginationParams & { filter?: string }): Promise<ApiResponse<any[]>> {
    const response = await this.client.get(`/ai/autoprobes/${id}/tasks`, { params });
    return response.data;
  }

  async getAutoProbeV2Preview(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/autoprobes/${id}/preview`);
    return response.data;
  }

  async getAutoProbeV2Pattern(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/autoprobes/${id}/pattern`);
    return response.data;
  }

  async getAutoProbeV2PatternResults(id: string): Promise<ApiResponse<any[]>> {
    const response = await this.client.get(`/ai/autoprobes/${id}/pattern/results`);
    return response.data;
  }

  // AutoProbe V1 (legacy)
  async getAutoProbes(params?: PaginationParams & { filter?: string }): Promise<ApiResponse<any[]>> {
    const response = await this.client.get('/ai/autoprobes/v1', { params });
    return response.data;
  }

  async getAutoProbe(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/autoprobes/v1/${id}`);
    return response.data;
  }

  async createAutoProbe(autoprobe: any): Promise<ApiResponse<any>> {
    const response = await this.client.post('/ai/autoprobes/v1', { data: autoprobe });
    return response.data;
  }

  async runAutoProbeTask(id: string, params?: any): Promise<ApiResponse<any>> {
    const response = await this.client.post(`/ai/autoprobes/v1/${id}/tasks`, params);
    return response.data;
  }

  async getAutoProbePreview(id: string): Promise<ApiResponse<any>> {
    const response = await this.client.get(`/ai/autoprobes/v1/${id}/preview`);
    return response.data;
  }
}
