import { format, differenceInCalendarDays } from 'date-fns'


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
        formatString = "yyyy-MM-dd"
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
