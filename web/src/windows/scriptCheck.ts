/**
 * Checks a script content for dangerous or incompatible Windows shell commands.
 * Under background tasks (non-interactive mode with stdin redirected), commands
 * like 'timeout' and 'pause' will fail or hang.
 * @param filename File path or name to verify extension.
 * @param content The text content of the script.
 * @returns Array of warnings. Empty if no issues are detected or file is not a Windows script.
 */
export function checkWindowsScriptWarnings(filename: string, content: string): string[] {
  const name = filename.toLowerCase()
  if (!name.endsWith('.bat') && !name.endsWith('.cmd') && !name.endsWith('.ps1')) {
    return []
  }

  const warnings: string[] = []
  if (/\btimeout\b/i.test(content)) {
    warnings.push('⚠️ 检测到 "timeout" 命令：在后台非交互模式运行（标准输入被重定向）下，"timeout" 会引发语法报错或立即退出。如果在循环中使用该命令，会导致 CPU 占满及前端界面卡死。建议替换为 "ping -n 61 127.0.0.1 >nul" (bat) 或 "Start-Sleep -s 60" (ps1)。')
  }
  if (/\bpause\b/i.test(content)) {
    warnings.push('⚠️ 检测到 "pause" 命令：后台非交互执行中，"pause" 无法等待键盘输入，会直接报错退出或挂起。')
  }
  return warnings
}
