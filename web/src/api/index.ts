const BASE_URL = '/api'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
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
    window.location.href = '/login'
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
    const res = await fetch(`${BASE_URL}/auth/me`, {
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
    list: (params?: { page?: number; page_size?: number; name?: string }) => {
      const query = new URLSearchParams()
      if (params?.page) query.set('page', String(params.page))
      if (params?.page_size) query.set('page_size', String(params.page_size))
      if (params?.name) query.set('name', params.name)
      return request<TaskListResponse>(`/tasks?${query}`)
    },
    create: (data: Partial<Task>) => request<Task>('/tasks', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: Partial<Task>) => request<Task>(`/tasks/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request(`/tasks/${id}`, { method: 'DELETE' }),
    execute: (id: number) => request(`/execute/task/${id}`, { method: 'POST' })
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
    detail: (id: number) => request<LogDetail>(`/logs/${id}`)
  },
  dashboard: {
    stats: () => request<Stats>('/stats'),
    sentence: () => request<{ sentence: string }>('/sentence'),
    sendStats: () => request<DailyStats[]>('/sendstats'),
    taskStats: () => request<TaskStatsItem[]>('/taskstats')
  },
  settings: {
    changePassword: (data: { old_password: string; new_password: string }) =>
      request('/settings/password', { method: 'POST', body: JSON.stringify(data) }),
    getSite: () => request<SiteSettings>('/settings/site'),
    getPublicSite: () => request<{ title: string; subtitle: string; icon: string }>('/settings/public'),
    updateSite: (data: SiteSettings) =>
      request('/settings/site', { method: 'PUT', body: JSON.stringify(data) }),
    getScheduler: () => request<SchedulerSettings>('/settings/scheduler'),
    updateScheduler: (data: SchedulerSettings) =>
      request('/settings/scheduler', { method: 'PUT', body: JSON.stringify(data) }),
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
    downloadBackup: () => `${BASE_URL}/settings/backup/download`,
    restoreBackup: async (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      const res = await fetch(`${BASE_URL}/settings/restore`, {
        method: 'POST',
        credentials: 'include',
        body: formData
      })
      const json: ApiResponse<null> = await res.json()
      if (json.code === 401) {
        window.location.href = '/login'
        throw new Error('请先登录')
      }
      if (json.code !== 200) throw new Error(json.msg || '恢复失败')
    }
  },
  files: {
    tree: () => request<FileNode[]>('/files/tree'),
    getContent: (path: string) => request<{ path: string; content: string }>(`/files/content?path=${encodeURIComponent(path)}`),
    saveContent: (path: string, content: string) => request('/files/content', { method: 'POST', body: JSON.stringify({ path, content }) }),
    create: (path: string, isDir: boolean) => request('/files/create', { method: 'POST', body: JSON.stringify({ path, isDir }) }),
    delete: (path: string) => request('/files/delete', { method: 'POST', body: JSON.stringify({ path }) }),
    rename: (oldPath: string, newPath: string) => request('/files/rename', { method: 'POST', body: JSON.stringify({ oldPath, newPath }) }),
    uploadArchive: async (file: File, targetPath?: string) => {
      const formData = new FormData()
      formData.append('file', file)
      if (targetPath) formData.append('path', targetPath)
      
      const res = await fetch(`${BASE_URL}/files/upload`, {
        method: 'POST',
        credentials: 'include',
        body: formData
      })
      const json: ApiResponse<null> = await res.json()
      if (json.code === 401) {
        window.location.href = '/login'
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
      
      const res = await fetch(`${BASE_URL}/files/uploadfiles`, {
        method: 'POST',
        credentials: 'include',
        body: formData
      })
      const json: ApiResponse<null> = await res.json()
      if (json.code === 401) {
        window.location.href = '/login'
        throw new Error('请先登录')
      }
      if (json.code !== 200) throw new Error(json.msg || '上传失败')
    }
  },
  deps: {
    list: (type?: string) => {
      const query = type ? `?type=${type}` : ''
      return request<Dependency[]>(`/deps${query}`)
    },
    create: (data: { name: string; version?: string; type: string; remark?: string }) =>
      request<Dependency>('/deps', { method: 'POST', body: JSON.stringify(data) }),
    delete: (id: number) => request(`/deps/${id}`, { method: 'DELETE' }),
    install: (data: { name: string; version?: string; type: string }) =>
      request('/deps/install', { method: 'POST', body: JSON.stringify(data) }),
    uninstall: (id: number) => request(`/deps/uninstall/${id}`, { method: 'POST' }),
    getInstalled: (type: string) => request<Dependency[]>(`/deps/installed?type=${type}`)
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
  schedule: string
  timeout: number
  clean_config: string
  envs: string
  enabled: boolean
  last_run: string
  next_run: string
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
}

export interface EnvListResponse {
  data: EnvVar[]
  total: number
  page: number
  page_size: number
}

export interface Stats {
  tasks: number
  scripts: number
  envs: number
  logs: number
  scheduled: number
  running: number
}


export interface TaskLog {
  id: number
  task_id: number
  task_name: string
  command: string
  status: string
  duration: number
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
  status: string
  duration: number
  created_at: string
}

export interface AboutInfo {
  version: string
  build_time: string
  mem_usage: string
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
  type: string
  remark: string
  log: string
  created_at: string
  updated_at: string
}
