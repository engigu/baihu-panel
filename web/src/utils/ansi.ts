import AnsiUp from 'ansi-to-html'

const emojiMap: Record<string, string> = {
  ':success:': '✅',
  ':check:': '✅',
  ':done:': '✅',
  ':error:': '❌',
  ':fail:': '❌',
  ':x:': '❌',
  ':warn:': '⚠️',
  ':warning:': '⚠️',
  ':info:': 'ℹ️',
  ':rocket:': '🚀',
  ':sparkles:': '✨',
  ':fire:': '🔥',
  ':bug:': '🐛',
  ':lock:': '🔒',
  ':link:': '🔗',
  ':memo:': '📝',
  ':bulb:': '💡',
  ':clock:': '🕒',
  ':finish:': '🏁',
  ':start:': '🛫',
  ':cloud:': '☁️',
  ':bell:': '🔔',
}

const ansiUp = new AnsiUp({
  newline: false,
  escapeXML: true,
  stream: false,
  colors: {
    // Optimized 16-color palette for dark terminal background
    0:  '#09090b', // Black (Zinc-950)
    1:  '#f87171', // Red (Tailwind red-400)
    2:  '#4ade80', // Green (Tailwind green-400)
    3:  '#fbbf24', // Yellow (Tailwind amber-400)
    4:  '#60a5fa', // Blue (Tailwind blue-400)
    5:  '#c084fc', // Magenta (Tailwind purple-400)
    6:  '#22d3ee', // Cyan (Tailwind cyan-400)
    7:  '#e4e4e7', // White (Zinc-200)
    8:  '#71717a', // Bright Black (Zinc-500)
    9:  '#ef4444', // Bright Red (Tailwind red-500)
    10: '#22c55e', // Bright Green (Tailwind green-500)
    11: '#f59e0b', // Bright Yellow (Tailwind amber-500)
    12: '#3b82f6', // Bright Blue (Tailwind blue-500)
    13: '#a855f7', // Bright Magenta (Tailwind purple-500)
    14: '#06b6d4', // Bright Cyan (Tailwind cyan-500)
    15: '#ffffff'  // Bright White
  }
})

/**
 * Convert ANSI escape codes to HTML and parse emoji shortcodes
 */
export function ansiToHtml(ansi: string): string {
  if (!ansi) return ''
  
  // Parse emoji shortcodes
  let processed = ansi.replace(/:([a-z0-9_-]+):/g, (match) => {
    return emojiMap[match] || match
  })

  return ansiUp.toHtml(processed)
}

/**
 * Highlight keywords in HTML content while avoiding HTML tags
 */
export function highlightHtml(html: string, keyword: string): string {
  if (!keyword.trim()) return html
  
  const escaped = keyword.trim().replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  // Match keyword but not inside HTML tags
  const regex = new RegExp(`(${escaped})(?![^<]*>)`, 'gi')
  return html.replace(regex, '<mark class="bg-yellow-400/30 text-yellow-100 border-b border-yellow-400/50 px-0.5 rounded-sm transition-colors">$1</mark>')
}
