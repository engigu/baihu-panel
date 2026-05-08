/**
 * 极简模板渲染工具 (v2)
 */
export function renderTemplate(template: string, data: Record<string, any>): string {
  if (!template) return '';
  
  // 简单的变量替换 {{var}}
  return template.replace(/{{(.*?)}}/g, (match, key) => {
    const k = key.trim();
    const value = data[k];
    return value !== undefined ? String(value) : match;
  });
}

/**
 * 常用变量标签
 */
export const TEMPLATE_TAGS = [
  { label: '{{task_id}}', value: 'task_id', desc: '任务 ID' },
  { label: '{{task_name}}', value: 'task_name', desc: '任务名称' },
  { label: '{{start_time}}', value: 'start_time', desc: '开始时间' },
  { label: '{{duration}}', value: 'duration', desc: '任务耗时(ms)' },
  { label: '{{output}}', value: 'output', desc: '任务输出日志' },
  { label: '{{status}}', value: 'status', desc: '执行状态' },
];
