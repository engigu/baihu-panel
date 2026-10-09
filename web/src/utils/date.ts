import { format, differenceInCalendarDays, isSameYear } from 'date-fns'


export function formatDate(date: Date | string | number | undefined, formatString: string | undefined = undefined): string {
    if (!date) return ''
    const targetDate = new Date(date)
    if (isNaN(targetDate.getTime())) {
        return typeof date === 'string' ? date : ''
    }
    const now = new Date()
    const diffDays = differenceInCalendarDays(now, targetDate)
    if (diffDays === 2) {
        return '前天'
    }
    if (diffDays === 1) {
        return '昨天'
    }
    if (diffDays === 0) {
        return '今天'
    }
    if (diffDays === -1) {
        return '明天'
    }
    if (diffDays === -2) {
        return '后天'
    }
    if (!formatString) {
        formatString = isSameYear(now, targetDate) ? "MM-dd" : "yyyy-MM-dd"
    }
    try {
        return format(targetDate, formatString)
    } catch {
        return typeof date === 'string' ? date : ''
    }
}


export function formatDateTime(date: Date | string | number | undefined, dateFormatString: string | undefined = undefined, timeFormatString: string | undefined = undefined): string {
    if (!date) return ''
    const targetDate = new Date(date)
    if (isNaN(targetDate.getTime())) {
        return typeof date === 'string' ? date : ''
    }
    const dateStr = formatDate(targetDate, dateFormatString)
    if (!timeFormatString) {
        timeFormatString = "HH:mm:ss"
    }
    try {
        const timeStr = format(targetDate, timeFormatString)
        return `${dateStr} ${timeStr}`
    } catch {
        return dateStr
    }
}

export interface UptimePart {
    value: number
    unit: string
}

export interface FormattedUptime {
    compact: string
    full: string
    parts: UptimePart[]
}

/**
 * 格式化运行时长（前端自主计算，传入秒数纯数值）
 * compact: 紧凑自适应字符串
 * full: 完整精确时长（用于 title 悬停查看）
 * parts: 结构化数字与单位拆分列表（用于与数值卡片保持一致的“大数字 + 小灰色单位”排印）
 */
export function formatUptime(uptimeSeconds: number | undefined | null): FormattedUptime {
    if (uptimeSeconds === undefined || uptimeSeconds === null || isNaN(uptimeSeconds) || uptimeSeconds < 0) {
        return { compact: '-', full: '-', parts: [] }
    }

    const totalSec = Math.floor(uptimeSeconds)
    const s = Math.floor(totalSec % 60)
    const m = Math.floor((totalSec / 60) % 60)
    const h = Math.floor((totalSec / 3600) % 24)
    const d = Math.floor(totalSec / 86400)

    let full = ''
    let compact = ''
    const parts: UptimePart[] = []

    if (d > 0) {
        full = `${d}天${h}小时${m}分钟${s}秒`
        parts.push({ value: d, unit: '天' })
        if (h > 0) {
            parts.push({ value: h, unit: '小时' })
            compact = `${d}天${h}小时`
        } else {
            compact = `${d}天`
        }
    } else if (h > 0) {
        full = `${h}小时${m}分钟${s}秒`
        parts.push({ value: h, unit: '小时' })
        if (m > 0) {
            parts.push({ value: m, unit: '分' })
            compact = `${h}小时${m}分`
        } else {
            compact = `${h}小时`
        }
    } else if (m > 0) {
        full = `${m}分钟${s}秒`
        parts.push({ value: m, unit: '分' })
        if (s > 0) {
            parts.push({ value: s, unit: '秒' })
            compact = `${m}分${s}秒`
        } else {
            compact = `${m}分钟`
        }
    } else {
        full = `${s}秒`
        compact = full
        parts.push({ value: s, unit: '秒' })
    }

    return { compact, full, parts }
}
