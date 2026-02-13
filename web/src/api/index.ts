// 获取 base URL（从后端注入的全局变量）
const BASE_URL = (window as any).__BASE_URL__ || ''
const API_VERSION = (window as any).__API_VERSION__ || '/api/v1'
const API_BASE_URL = BASE_URL + API_VERSION

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${url}`, {
    ...options,
    credentials: 'include', // 携带 Cookie
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers
    }
  })

  const json: ApiResponse<T> = await res.json()

  if (json.code === 401) {
    // 未登录或登录过期，跳转到登录页
    window.location.href = BASE_URL + '/login'
    throw new Error(json.msg || '请先登录')
  }

  if (json.code !== 200) {
    throw new Error(json.msg || '请求失败')
  }

  return json.data
}

// 检查登录状态（不触发自动跳转）
export async function checkAuth(): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE_URL}/auth/me`, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' }
    })
    const json: ApiResponse<{ username: string }> = await res.json()
    return json.code === 200
  } catch {
    return false
  }
}

export const api = {
  auth: {
    login: (data: { username: string; password: string }) =>
      request<{ user: string }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
    logout: () => request('/auth/logout', { method: 'POST' }),
    me: () => request<{ username: string }>('/auth/me'),
    register: (data: { username: string; password: string; email: string }) =>
      request('/auth/register', { method: 'POST', body: JSON.stringify(data) })
  },
  tasks: {
    list: (params?: { page?: number; page_size?: number; name?: string; agent_id?: number }) => {
      const query = new URLSearchParams()
      if (params?.page) query.set('page', String(params.page))
      if (params?.page_size) query.set('page_size', String(params.page_size))
      if (params?.name) query.set('name', params.name)
      if (params?.agent_id) query.set('agent_id', String(params.agent_id))
      return request<TaskListResponse>(`/tasks?${query}`)
    },
    create: (data: Partial<Task>) => request<Task>('/tasks', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: Partial<Task>) => request<Task>(`/tasks/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request(`/tasks/${id}`, { method: 'DELETE' }),
    execute: (id: number) => request<ExecutionResult>(`/execute/task/${id}`, { method: 'POST' }),
    stop: (logID: number) => request(`/tasks/stop/${logID}`, { method: 'POST' })
  },
  scripts: {
    list: () => request<Script[]>('/scripts'),
    create: (data: Partial<Script>) => request<Script>('/scripts', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: Partial<Script>) => request<Script>(`/scripts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request(`/scripts/${id}`, { method: 'DELETE' })
  },
  env: {
    list: (params?: { page?: number; page_size?: number; name?: string }) => {
      const query = new URLSearchParams()
      if (params?.page) query.set('page', String(params.page))
      if (params?.page_size) query.set('page_size', String(params.page_size))
      if (params?.name) query.set('name', params.name)
      return request<EnvListResponse>(`/env?${query}`)
    },
    all: () => request<EnvVar[]>('/env/all'),
    create: (data: Partial<EnvVar>) => request<EnvVar>('/env', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: Partial<EnvVar>) => request<EnvVar>(`/env/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request(`/env/${id}`, { method: 'DELETE' })
  },
  execute: {
    command: (command: string) => request('/execute/command', { method: 'POST', body: JSON.stringify({ command }) }),
    results: () => request('/execute/results')
  },
  logs: {
    list: (params?: { page?: number; page_size?: number; task_id?: number; task_name?: string }) => {
      const query = new URLSearchParams()
      if (params?.page) query.set('page', String(params.page))
      if (params?.page_size) query.set('page_size', String(params.page_size))
      if (params?.task_id) query.set('task_id', String(params.task_id))
      if (params?.task_name) query.set('task_name', params.task_name)
      return request<LogListResponse>(`/logs?${query}`)
    },
    get: (id: number) => request<LogDetail>(`/logs/${id}`),
    detail: (id: number) => request<LogDetail>(`/logs/${id}`)
  },
  dashboard: {
    stats: () => request<Stats>('/stats'),
    sentence: () => request<{ sentence: string }>('/sentence'),
    sendStats: (days?: number) => request<DailyStats[]>(`/sendstats${days ? `?days=${days}` : ''}`),
    taskStats: (days?: number) => request<TaskStatsItem[]>(`/taskstats${days ? `?days=${days}` : ''}`)
  },
  settings: {
    changePassword: (data: { old_password: string; new_password: string }) =>
      request('/settings/password', { method: 'POST', body: JSON.stringify(data) }),
    getSite: () => request<SiteSettings>('/settings/site'),
    getPublicSite: () => request<{ title: string; subtitle: string; icon: string; demo_mode: boolean }>('/settings/public'),
    updateSite: (data: SiteSettings) =>
      request('/settings/site', { method: 'PUT', body: JSON.stringify(data) }),
    getScheduler: () => request<SchedulerSettings>('/settings/scheduler'),
    updateScheduler: (data: SchedulerSettings) =>
      request('/settings/scheduler', { method: 'PUT', body: JSON.stringify(data) }),
    getPaths: () => request<{ scripts_dir: string }>('/settings/paths'),
    getAbout: () => request<AboutInfo>('/settings/about'),
    getLoginLogs: (params?: { page?: number; page_size?: number; username?: string }) => {
      const query = new URLSearchParams()
      if (params?.page) query.set('page', String(params.page))
      if (params?.page_size) query.set('page_size', String(params.page_size))
      if (params?.username) query.set('username', params.username)
      return request<LoginLogListResponse>(`/settings/loginlogs?${query}`)
    },
    createBackup: () => request('/settings/backup', { method: 'POST' }),
    getBackupStatus: () => request<{ has_backup: boolean; backup_time: string }>('/settings/backup/status'),
    downloadBackup: () => `${API_BASE_URL}/settings/backup/download`,
    restoreBackup: async (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      const res = await fetch(`${API_BASE_URL}/settings/restore`, {
        method: 'POST',
        credentials: 'include',
        body: formData
      })
      const json: ApiResponse<null> = await res.json()
      if (json.code === 401) {
        window.location.href = BASE_URL + '/login'
        throw new Error('请先登录')
      }
      if (json.code !== 200) throw new Error(json.msg || '恢复失败')
    }
  },
  files: {
    tree: () => request<FileNode[]>('/files/tree'),
    getContent: (path: string) => request<{ path: string; content: string }>(`/files/content?path=${encodeURIComponent(path)}`),
    download: (path: string) => `${API_BASE_URL}/files/download?path=${encodeURIComponent(path)}`,
    saveContent: (path: string, content: string) => request('/files/content', { method: 'POST', body: JSON.stringify({ path, content }) }),
    create: (path: string, isDir: boolean) => request('/files/create', { method: 'POST', body: JSON.stringify({ path, isDir }) }),
    delete: (path: string) => request('/files/delete', { method: 'POST', body: JSON.stringify({ path }) }),
    rename: (oldPath: string, newPath: string) => request('/files/rename', { method: 'POST', body: JSON.stringify({ oldPath, newPath }) }),
    move: (oldPath: string, newPath: string) => request('/files/move', { method: 'POST', body: JSON.stringify({ oldPath, newPath }) }),
    uploadArchive: async (file: File, targetPath?: string) => {
      const formData = new FormData()
      formData.append('file', file)
      if (targetPath) formData.append('path', targetPath)

      const res = await fetch(`${API_BASE_URL}/files/upload`, {
        method: 'POST',
        credentials: 'include',
        body: formData
      })
      const json: ApiResponse<null> = await res.json()
      if (json.code === 401) {
        window.location.href = BASE_URL + '/login'
        throw new Error('请先登录')
      }
      if (json.code !== 200) throw new Error(json.msg || '上传失败')
    },
    uploadFiles: async (files: FileList, paths: string[], targetPath?: string) => {
      const formData = new FormData()
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        if (file) {
          formData.append('files', file)
          formData.append('paths', paths[i] || file.name)
        }
      }
      if (targetPath) formData.append('path', targetPath)

      const res = await fetch(`${API_BASE_URL}/files/uploadfiles`, {
        method: 'POST',
        credentials: 'include',
        body: formData
      })
      const json: ApiResponse<null> = await res.json()
      if (json.code === 401) {
        window.location.href = BASE_URL + '/login'
        throw new Error('请先登录')
      }
      if (json.code !== 200) throw new Error(json.msg || '上传失败')
    }
  },
  deps: {
    list: (params?: { language?: string; lang_version?: string }) => {
      const query = new URLSearchParams()
      if (params?.language) query.set('language', params.language)
      if (params?.lang_version) query.set('lang_version', params.lang_version)
      return request<Dependency[]>(`/deps?${query}`)
    },
    create: (data: { name: string; version?: string; language: string; lang_version?: string; remark?: string }) =>
      request<Dependency>('/deps', { method: 'POST', body: JSON.stringify(data) }),
    delete: (id: number) => request(`/deps/${id}`, { method: 'DELETE' }),
    install: (data: any) => request<any>('/deps/install', { method: 'POST', body: JSON.stringify(data) }),
    getInstallCmd: (data: any) => request<{ command: string }>('/deps/install-cmd', { method: 'POST', body: JSON.stringify(data) }),
    uninstall: (id: number) => request<any>(`/deps/uninstall/${id}`, { method: 'POST' }),
    reinstall: (id: number) => request(`/deps/reinstall/${id}`, { method: 'POST' }),
    reinstallAll: (language: string, lang_version?: string) => {
      const query = new URLSearchParams({ language })
      if (lang_version) query.set('lang_version', lang_version)
      return request(`/deps/reinstall-all?${query}`, { method: 'POST' })
    },
    getReinstallAllCmd: (language: string, lang_version?: string) => {
      const query = new URLSearchParams({ language })
      if (lang_version) query.set('lang_version', lang_version)
      return request<{ command: string }>(`/deps/reinstall-all-cmd?${query}`, { method: 'POST' })
    },
    getInstalled: (language: string, lang_version?: string) => {
      const query = new URLSearchParams({ language })
      if (lang_version) query.set('lang_version', lang_version)
      return request<Dependency[]>(`/deps/installed?${query}`)
    }
  },
  agents: {
    list: () => request<Agent[]>('/agents'),
    getVersion: () => request<{ version: string; platforms: { os: string; arch: string; filename: string }[] }>('/agents/version'),
    update: (id: number, data: { name: string; description?: string; enabled: boolean }) =>
      request('/agents/' + id, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request('/agents/' + id, { method: 'DELETE' }),
    forceUpdate: (id: number) => request('/agents/' + id + '/update', { method: 'POST' }),
    downloadUrl: (os: string, arch: string) => `${API_BASE_URL}/agent/download?os=${os}&arch=${arch}`,
    // 令牌管理
    listTokens: () => request<AgentToken[]>('/agents/tokens'),
    createToken: (data: { remark?: string; max_uses?: number; expires_at?: string }) =>
      request<AgentToken>('/agents/tokens', { method: 'POST', body: JSON.stringify(data) }),
    deleteToken: (id: number) => request('/agents/tokens/' + id, { method: 'DELETE' })
  },
  mise: {
    list: () => request<MiseLanguage[]>('/mise/ls'),
    sync: () => request<void>('/mise/sync', { method: 'POST' }),
    plugins: () => request<string[]>('/mise/plugins'),
    versions: (plugin: string) => request<string[]>(`/mise/versions?plugin=${plugin}`),
    verifyCommand: (plugin: string, version: string) => request<{ command: string }>(`/mise/verify-cmd?plugin=${plugin}&version=${version}`)
  }
}

export interface FileNode {
  name: string
  path: string
  isDir: boolean
  children?: FileNode[]
}

export interface Task {
  id: number
  name: string
  command: string
  type: string
  config: string
  schedule: string
  timeout: number
  work_dir: string
  clean_config: string
  envs: string
  languages: { name: string; version: string }[]
  agent_id: number | null
  enabled: boolean
  last_run: string
  next_run: string
}

export interface RepoConfig {
  source_type: string
  source_url: string
  target_path: string
  branch: string
  sparse_path: string
  single_file: boolean
  proxy: string
  proxy_url: string
  auth_token: string
  concurrency?: number
}

export interface ExecutionResult {
  TaskID: number
  Success: boolean
  Output: string
  Error: string
  Start: string
  End: string
}

export interface TaskListResponse {
  data: Task[]
  total: number
  page: number
  page_size: number
}

export interface Script {
  id: number
  name: string
  content: string
}

export interface EnvVar {
  id: number
  name: string
  value: string
  remark: string
  hidden: boolean
}

export interface EnvListResponse {
  data: EnvVar[]
  total: number
  page: number
  page_size: number
}

export interface Stats {
  tasks: number
  today_execs: number
  envs: number
  logs: number
  scheduled: number
  running: number
}


export interface TaskLog {
  id: number
  task_id: number
  task_name: string
  task_type: string
  command: string
  status: string
  duration: number
  error: string | null
  start_time: string | null
  end_time: string | null
  created_at: string
}

export interface LogListResponse {
  data: TaskLog[]
  total: number
  page: number
  page_size: number
}

export interface LogDetail {
  id: number
  task_id: number
  command: string
  output: string
  error: string | null
  status: string
  duration: number
  start_time: string | null
  end_time: string | null
  created_at: string
}

export interface AboutInfo {
  version: string
  build_time: string
  mem_usage: string
  goroutines: number
  uptime: string
  task_count: number
  log_count: number
  env_count: number
}

export interface SiteSettings {
  title: string
  subtitle: string
  icon: string
  page_size: string
  cookie_days: string
}

export interface SchedulerSettings {
  worker_count: string
  queue_size: string
  rate_interval: string
}


export interface LoginLog {
  id: number
  username: string
  ip: string
  user_agent: string
  status: string
  message: string
  created_at: string
}

export interface LoginLogListResponse {
  data: LoginLog[]
  total: number
  page: number
  page_size: number
}

export interface DailyStats {
  day: string
  total: number
  success: number
  failed: number
}

export interface TaskStatsItem {
  task_id: number
  task_name: string
  count: number
}

export interface Dependency {
  id: number
  name: string
  version: string
  language: string
  lang_version: string
  remark: string
  log: string
  created_at: string
  updated_at: string
}

export interface Agent {
  id: number
  name: string
  token: string
  machine_id: string
  description: string
  status: string
  last_seen: string
  ip: string
  version: string
  build_time: string
  hostname: string
  os: string
  arch: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface AgentToken {
  id: number
  token: string
  remark: string
  max_uses: number
  used_count: number
  expires_at: string | null
  enabled: boolean
  created_at: string
}

export interface MiseLanguage {
  plugin: string
  version: string
  source: { type?: string; path?: string } | string
  active: boolean
  install_path?: string
  installed_at?: string  // 安装日期
}
