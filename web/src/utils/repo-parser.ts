import type { RepoConfig, Task } from '@/api'
import { PATHS } from '@/constants'

/**
 * 仓库命令解析结果
 */
export interface ParsedRepoResult {
  repoConfig: Partial<RepoConfig>
  task: Partial<Task>
}

/**
 * 解析带引号的命令行字符串为参数数组
 */
export function parseArgs(command: string): string[] {
  const args: string[] = []
  const regex = /[^\s"']+|"([^"]*)"|'([^']*)'/g
  let match
  while ((match = regex.exec(command)) !== null) {
    args.push(match[1] || match[2] || match[0])
  }
  return args
}

/**
 * 解析 Baihu 格式的导入命令
 */
export function parseBaihuCommand(command: string): ParsedRepoResult | null {
  const s = command.trim()
  if (!s) return null

  const args = parseArgs(s)
  let i = 0
  // 跳过开头的 'baihu' 或 'reposync'
  if (args[i] === 'baihu') i++
  if (args[i] === 'reposync') i++

  const repoConfig: Partial<RepoConfig> = {}
  const task: Partial<Task> = {}
  let hasValidField = false

  for (; i < args.length; i++) {
    const arg = args[i]
    if (!arg || !arg.startsWith('--')) continue

    hasValidField = true

    if (arg === '--single-file') {
      repoConfig.single_file = true
      const value = args[i + 1]
      if (value === 'true' || value === 'false') {
        repoConfig.single_file = value === 'true'
        i++
      }
      continue
    }

    const value = args[i + 1]
    if (value === undefined || value.startsWith('--')) continue

    i++ // 推进索引并跳过已处理的 value

    switch (arg) {
      case '--source-type':
        repoConfig.source_type = value
        break
      case '--source-url':
        repoConfig.source_url = value
        // 如果 URL 存在，尝试生成默认名称
        if (value) {
          try {
            const urlPaths = value.split('/')
            const name = urlPaths[urlPaths.length - 1]?.replace('.git', '') || '未命名仓库'
            task.name = '同步 ' + name
          } catch { /* ignore */ }
        }
        break
      case '--target-path':
        // 处理脚本目录占位符
        if (value.startsWith(`${PATHS.SCRIPTS_DIR_PLACEHOLDER}/`)) {
          repoConfig.target_path = value.replace(`${PATHS.SCRIPTS_DIR_PLACEHOLDER}/`, '')
        } else if (value === PATHS.SCRIPTS_DIR_PLACEHOLDER) {
          repoConfig.target_path = ''
        } else {
          repoConfig.target_path = value
        }
        break
      case '--branch':
        repoConfig.branch = value
        break
      case '--path':
        repoConfig.sparse_path = value
        break
      case '--proxy-url':
        repoConfig.proxy_url = value
        repoConfig.proxy = 'custom'
        break
      case '--auth-token':
        repoConfig.auth_token = value
        break
      case '--whitelist-paths':
        repoConfig.whitelist_paths = value
        break
      case '--blacklist':
        repoConfig.blacklist = value
        break
      case '--dependence':
        repoConfig.dependence = value
        break
      case '--extensions':
        repoConfig.extensions = value
        break
      case '--task-timeout':
        task.timeout = parseInt(value) || 30
        break
      case '--task-langs':
        try {
          const langs = JSON.parse(value)
          if (Array.isArray(langs)) {
            task.languages = langs.map(l => ({
              name: l.name || '',
              version: l.version || ''
            }))
          }
        } catch (e) {
          console.error('Parse task-langs failed', e)
        }
        break
      case '--repo-name':
        repoConfig.repo_dir_name = value
        break
      case '--pre-command':
        task.pre_command = value
        break
      case '--post-command':
        task.post_command = value
        break
    }
  }

  if (!hasValidField) return null

  // 设置默认规则
  repoConfig.auto_add_cron = true
  repoConfig.commenttotask = 'true'

  return { repoConfig, task }
}

/**
 * 解析 青龙 (Qinglong) 格式的导入命令
 */
export function parseQlCommand(command: string): ParsedRepoResult | null {
  const s = command.trim()
  if (!s || !s.startsWith('ql repo')) return null

  const args = parseArgs(s)
  const repoConfig: Partial<RepoConfig> = {}
  const task: Partial<Task> = {}

  if (args[2]) {
    repoConfig.source_url = args[2]
    repoConfig.source_type = 'git'
    // 生成名称
    try {
      const urlPaths = args[2].split('/')
      const name = urlPaths.length > 0 ? urlPaths[urlPaths.length - 1]?.replace('.git', '') : '未命名仓库'
      task.name = '同步 ' + (name || '未命名仓库')
    } catch {
      task.name = '同步 未命名仓库'
    }
  }

  if (args[3]) repoConfig.whitelist_paths = args[3]
  if (args[4]) repoConfig.blacklist = args[4]
  if (args[5]) repoConfig.dependence = args[5]
  if (args[6]) repoConfig.branch = args[6]
  if (args[7]) repoConfig.extensions = args[7]

  repoConfig.auto_add_cron = true
  repoConfig.commenttotask = 'true'
  repoConfig.repo_source = 'ql'

  return { repoConfig, task }
}

/**
 * 根据 Task 生成用于导出的 baihu reposync 命令
 */
export function generateBaihuCommand(task: Task): string {
  try {
    const config = JSON.parse(task.config) as RepoConfig
    const args: string[] = ['baihu', 'reposync']

    args.push('--source-type', config.source_type || 'git')
    args.push('--source-url', config.source_url || '')

    // 处理 target_path
    let targetPath = config.target_path || ''
    if (!targetPath) {
      targetPath = PATHS.SCRIPTS_DIR_PLACEHOLDER
    } else if (targetPath.startsWith(`${PATHS.SCRIPTS_DIR_PLACEHOLDER}/`) || targetPath === PATHS.SCRIPTS_DIR_PLACEHOLDER) {
      // 保持原样
    } else if (targetPath.startsWith('/') || /^[a-zA-Z]:/.test(targetPath)) {
      // 绝对路径，保持原样
    } else {
      // 相对路径，转换为带占位符的格式
      targetPath = `${PATHS.SCRIPTS_DIR_PLACEHOLDER}/${targetPath.replace(/^\/+/, '')}`
    }
    
    // 如果 targetPath 不是默认值，才需输出 --target-path
    if (targetPath !== PATHS.SCRIPTS_DIR_PLACEHOLDER) {
      args.push('--target-path', targetPath)
    }

    if (config.branch) {
      args.push('--branch', config.branch)
    }
    if (config.repo_dir_name) {
      args.push('--repo-name', config.repo_dir_name)
    }
    if (config.sparse_path) {
      args.push('--path', config.sparse_path)
    }
    if (config.single_file) {
      args.push('--single-file')
    }
    if (config.proxy && config.proxy !== 'none') {
      args.push('--proxy', config.proxy)
      if (config.proxy === 'custom' && config.proxy_url) {
        args.push('--proxy-url', config.proxy_url)
      }
    }
    if (config.auth_token) {
      args.push('--auth-token', config.auth_token)
    }
    if (config.whitelist_paths) {
      args.push('--whitelist-paths', config.whitelist_paths)
    }
    if (config.blacklist) {
      args.push('--blacklist', config.blacklist)
    }
    if (config.dependence) {
      args.push('--dependence', config.dependence)
    }
    if (config.commenttotask === 'true') {
      args.push('--commenttotask', 'true')
    }
    if (config.extensions) {
      args.push('--extensions', config.extensions)
    }
    if (task.pre_command) {
      args.push('--pre-command', task.pre_command)
    }
    if (task.post_command) {
      args.push('--post-command', task.post_command)
    }
    if (task.timeout !== undefined && task.timeout !== 30) {
      args.push('--task-timeout', String(task.timeout))
    }
    if (task.languages && task.languages.length > 0) {
      const langs = task.languages.map(l => ({ name: l.name, version: l.version }))
      args.push('--task-langs', JSON.stringify(langs))
    }

    // 转义并加引号（仅在包含特殊字符或为空时加引号，提升指令可读性）
    return args.map(arg => {
      if (arg === 'baihu' || arg === 'reposync') return arg
      if (arg === '') return "''"
      // 如果只包含安全字符（字母、数字、短横线、下划线、点、斜杠、冒号、@），则无需加引号
      if (/^[a-zA-Z0-9_\-\.\/:@]+$/.test(arg)) {
        return arg
      }
      return "'" + arg.replace(/'/g, "'\\''") + "'"
    }).join(' ')
  } catch (e) {
    console.error('generateBaihuCommand failed', e)
    return ''
  }
}
