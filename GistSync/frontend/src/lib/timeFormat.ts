export interface SnapshotTimeFormatOptions {
  locale?: string
  timeZone?: string
}

const DEFAULT_LOCALE = 'zh-CN'
const DEFAULT_TIME_ZONE = 'Asia/Shanghai'
const INVALID_DATE_VALUE = Number.NaN

export function formatSnapshotCreatedAt(input: string, options: SnapshotTimeFormatOptions = {}): string {
  const date = new Date(input)
  if (Number.isNaN(date.getTime() ?? INVALID_DATE_VALUE)) {
    return input
  }
  const locale = options.locale ?? DEFAULT_LOCALE
  const timeZone = options.timeZone ?? DEFAULT_TIME_ZONE
  const formatter = new Intl.DateTimeFormat(locale, {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
  const parts = formatter.formatToParts(date)
  const values = extractDateParts(parts)
  return `${values.year}-${values.month}-${values.day} ${values.hour}:${values.minute}:${values.second}`
}

function extractDateParts(parts: Intl.DateTimeFormatPart[]): Record<string, string> {
  const out: Record<string, string> = {
    year: '',
    month: '',
    day: '',
    hour: '',
    minute: '',
    second: '',
  }
  for (const part of parts) {
    if (part.type in out) {
      out[part.type] = part.value
    }
  }
  return out
}
