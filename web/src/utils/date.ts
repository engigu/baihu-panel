import { format, differenceInCalendarDays } from 'date-fns'


export function formatDate(date: Date | string | number | undefined): string {
    if (!date) return ''
    const targetDate = new Date(date)
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
    return format(targetDate, 'yyyy-MM-dd')
}


export function formatDateTime(date: Date | string | number | undefined): string {
    if (!date) return ''
    const targetDate = new Date(date)
    const dateStr = formatDate(targetDate)
    const timeStr = format(targetDate, 'HH:mm:ss')
    return `${dateStr} ${timeStr}`
}
