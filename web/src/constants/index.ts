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
  NORMAL: 'task',
  REPO: 'repo',
} as const

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
