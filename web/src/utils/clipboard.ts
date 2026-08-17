/**
 * 兼容 HTTP 环境的剪贴板复制工具
 * 优先使用现代 navigator.clipboard API，如果不支持（如非 HTTPS 环境），
 * 则降级使用 document.execCommand('copy')
 * 
 * @param text 要复制的文本
 * @returns 复制是否成功
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  // 优先尝试现代 API
  if (navigator.clipboard && navigator.clipboard.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch (err) {
      console.warn('Clipboard API 复制失败, 尝试降级方案:', err)
    }
  }

  // 备选方案：使用传统的 execCommand
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    
    // 防止唤起虚拟键盘，并使其不可见
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    textarea.style.left = '-9999px'
    
    const parent = document.activeElement?.parentNode || document.body
    parent.appendChild(textarea)
    
    // 兼容 iOS 的选中方式
    textarea.select()
    textarea.setSelectionRange(0, 99999) 
    
    const success = document.execCommand('copy')
    parent.removeChild(textarea)
    
    if (success) {
      return true
    } else {
      console.error('execCommand 复制失败')
      return false
    }
  } catch (err) {
    console.error('降级复制失败:', err)
    return false
  }
}
