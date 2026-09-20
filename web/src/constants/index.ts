// 应用路径常量
export const PATHS = {
  // 脚本文件目录
  SCRIPTS_DIR: '/app/data/scripts',
  // 数据目录
  DATA_DIR: '/app/data',
  // 配置目录
  CONFIGS_DIR: '/app/configs',
  // 环境目录
  ENVS_DIR: '/app/envs',
  // 脚本目录占位符
  SCRIPTS_DIR_PLACEHOLDER: '$SCRIPTS_DIR$',
} as const

/**
 * 前端归一化路径：如果路径属于脚本目录或包含脚本目录前缀，归一化为 $SCRIPTS_DIR$/...
 * 如果是非脚本目录的外部绝对路径，原样保留。
 */
export function normalizeScriptPath(rawPath?: string, scriptsDir: string = PATHS.SCRIPTS_DIR): string {
  if (!rawPath || !rawPath.trim()) {
    return PATHS.SCRIPTS_DIR_PLACEHOLDER
  }
  const clean = rawPath.trim().replace(/\\/g, '/')
  if (clean.startsWith(PATHS.SCRIPTS_DIR_PLACEHOLDER)) {
    return clean
  }
  const cleanScripts = scriptsDir.replace(/\\/g, '/').replace(/\/$/, '')
  if (clean.toLowerCase().startsWith(cleanScripts.toLowerCase())) {
    const rel = clean.substring(cleanScripts.length).replace(/^\//, '')
    return rel ? `${PATHS.SCRIPTS_DIR_PLACEHOLDER}/${rel}` : PATHS.SCRIPTS_DIR_PLACEHOLDER
  }
  // 外部绝对路径原样保留
  return clean
}

/**
 * 前端还原/展示路径
 */
export function resolveScriptPath(logicPath?: string, scriptsDir: string = PATHS.SCRIPTS_DIR): string {
  if (!logicPath || !logicPath.trim() || logicPath === PATHS.SCRIPTS_DIR_PLACEHOLDER) {
    return scriptsDir
  }
  const clean = logicPath.trim().replace(/\\/g, '/')
  if (clean.startsWith(PATHS.SCRIPTS_DIR_PLACEHOLDER)) {
    const rel = clean.substring(PATHS.SCRIPTS_DIR_PLACEHOLDER.length).replace(/^\//, '')
    const base = scriptsDir.replace(/\\/g, '/').replace(/\/$/, '')
    return rel ? `${base}/${rel}` : base
  }
  return clean
}

// 文件扩展名对应的运行命令
export const FILE_RUNNERS: Record<string, string> = {
  py: 'python',
  js: 'node',
  sh: 'bash',
  bash: 'bash',
} as const

// 任务状态
export const TASK_STATUS = {
  SUCCESS: 'success',
  FAILED: 'failed',
  RUNNING: 'running',
  PENDING: 'pending',
  TIMEOUT: 'timeout',
  CANCELLED: 'cancelled',
} as const

export const TASK_STATUS_TEXT: Record<string, string> = {
  [TASK_STATUS.SUCCESS]: '已成功',
  [TASK_STATUS.FAILED]: '执行失败',
  [TASK_STATUS.RUNNING]: '正在运行',
  [TASK_STATUS.PENDING]: '等待队列',
  [TASK_STATUS.TIMEOUT]: '执行超时',
  [TASK_STATUS.CANCELLED]: '手动取消',
  'UNEXECUTED': '尚未执行',
} as const

// 任务类型
export const TASK_TYPE = {
  ALL: 'all',
  NORMAL: 'task',
  APP: 'app',
  REPO: 'repo',
} as const

export interface TaskTypeConfigItem {
  label: string
  color: string
  bgColor: string
  borderColor: string
}

// 任务类型统一配置表（极客科技高级配色与标签规范）
export const TASK_TYPE_CONFIG = {
  all: {
    label: '全部类型',
    color: 'text-sky-500 dark:text-sky-400',
    bgColor: 'bg-sky-500/10',
    borderColor: 'border-sky-500/20',
  },
  [TASK_TYPE.NORMAL]: {
    label: '脚本任务',
    color: 'text-cyan-500 dark:text-cyan-400',
    bgColor: 'bg-cyan-500/10',
    borderColor: 'border-cyan-500/20',
  },
  [TASK_TYPE.REPO]: {
    label: '仓库同步',
    color: 'text-violet-500 dark:text-violet-400',
    bgColor: 'bg-violet-500/10',
    borderColor: 'border-violet-500/20',
  },
  [TASK_TYPE.APP]: {
    label: '已装应用',
    color: 'text-emerald-500 dark:text-emerald-400',
    bgColor: 'bg-emerald-500/10',
    borderColor: 'border-emerald-500/20',
  },
} as const

export function getTaskTypeConfig(type?: string): TaskTypeConfigItem {
  const key = (type || 'task') as keyof typeof TASK_TYPE_CONFIG
  return (TASK_TYPE_CONFIG[key] || TASK_TYPE_CONFIG[TASK_TYPE.NORMAL]) as TaskTypeConfigItem
}

// 触发类型
export const TRIGGER_TYPE = {
  CRON: 'cron',
  BAIHU_STARTUP: 'baihu_startup',
} as const

// Agent 状态
export const AGENT_STATUS = {
  ONLINE: 'online',
  OFFLINE: 'offline',
} as const

// 环境变量类型
export const ENV_TYPE = {
  NORMAL: 'normal',
  SECRET: 'secret',
} as const

// 任务事件类型
export const TASK_EVENTS = {
  SUCCESS: 'task_success',
  FAILED: 'task_failed',
  TIMEOUT: 'task_timeout',
  RUNNING: 'task_running',
  QUEUED: 'task_queued',
  CANCELLED: 'task_cancelled',
} as const

// 日志事件类型
export const LOG_EVENTS = {
  ADDED: 'app_log_added',
} as const

// 系统事件类型 (对应后端的 system_ws_service.go)
export const SYSTEM_EVENTS = {
  INTERCONNECT_CHILD_STATUS: 'interconnect_child_status',
} as const
