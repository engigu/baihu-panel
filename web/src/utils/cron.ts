import cronstrue from 'cronstrue'
import 'cronstrue/locales/zh_CN'

export function getCronDescription(cron: string, lang?: string) {
  if (!cron || cron.trim() === '') return ''
  try {
    const locale = lang?.startsWith('en') ? 'en' : 'zh_CN'
    return cronstrue.toString(cron, { 
      locale,
      use24HourTimeFormat: true,
      throwExceptionOnParseError: true,
    })
  } catch (e) {
    return ''
  }
}
