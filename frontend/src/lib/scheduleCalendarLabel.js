/** Chronoscape / importdata 排班类型展示（与 20260517.json tagItemName 对齐） */
export const SCHEDULE_TAG_LEGEND = [
  { code: 'RDO', label: 'RDO', kind: 'leave' },
  { code: 'AL', label: 'AL', kind: 'leave' },
  { code: 'PHCL', label: 'PHCL', kind: 'leave' },
  { code: 'RDOC', label: 'RDOC', kind: 'leave' },
  { code: 'STANDBY24', label: '24h待命', kind: 'work' },
  { code: '1330-2306', label: '1330-2306', kind: 'work' },
  { code: '1030-2006', label: '1030-2006', kind: 'work' },
  { code: '1100-2036', label: '1100-2036', kind: 'work' },
]

const TIME_SHIFT = /^(\d{1,2})[:.]?(\d{2})-(\d{1,2})[:.]?(\d{2})$/

/** 月历格子底部简称 */
export function formatScheduleCalendarBottom(dayInfo) {
  if (!dayInfo || typeof dayInfo !== 'object') return ''
  if (dayInfo.calendarLabel) return String(dayInfo.calendarLabel).slice(0, 11)
  const mode = dayInfo.mode || ''
  const code = String(dayInfo.code || '').trim()
  if (mode === 'leave') {
    return code || String(dayInfo.shortLabel || '休').slice(0, 8)
  }
  if (code === 'STANDBY24') return '24h待命'
  const compact = code.replace(/\s/g, '')
  if (TIME_SHIFT.test(compact)) return compact.slice(0, 11)
  if (code) return code.slice(0, 11)
  if (dayInfo.shortLabel) return String(dayInfo.shortLabel).slice(0, 11)
  return ''
}

/** 选中日详情文案（完整排班说明） */
export function formatScheduleDaySummary(dayPayload) {
  if (!dayPayload) return '该日暂无排班记录'
  const pill = String(dayPayload.pillText || '').trim()
  if (pill) return pill
  const mode = dayPayload.mode
  const code = dayPayload.code || ''
  if (mode === 'leave') return `休假 · ${code}`
  if (code === 'STANDBY24') return '常规班 · 24小时手机待命'
  if (code) return `常规班 · ${code}`
  return '该日暂无排班记录'
}

export function scheduleDayCssClass(dayInfo) {
  if (!dayInfo?.mode) return ''
  if (dayInfo.mode === 'leave') return 'schedule-cal-day--leave'
  if (dayInfo.code === 'STANDBY24') return 'schedule-cal-day--standby'
  return 'schedule-cal-day--shift'
}

/** 周视图格子：班次缩写 + 上下班时间 */
export function formatScheduleGridCell(dayInfo) {
  if (!dayInfo || typeof dayInfo !== 'object') {
    return { abbr: '—', time: '', className: '' }
  }
  const abbr = formatScheduleCalendarBottom(dayInfo) || '—'
  let time = ''
  if (dayInfo.mode === 'work') {
    const start = String(dayInfo.startTime || '').trim()
    const end = String(dayInfo.endTime || '').trim()
    if (start && end) {
      time = `${start}–${end}`
    } else {
      const fromCode = formatTimeRangeFromShiftCode(dayInfo.code)
      if (fromCode) time = fromCode
    }
  }
  return { abbr, time, className: scheduleDayCssClass(dayInfo) }
}

function formatTimeRangeFromShiftCode(code) {
  const compact = String(code || '').replace(/\s/g, '')
  const match = TIME_SHIFT.exec(compact)
  if (!match) return ''
  const padPair = (hour, minute) => `${String(hour).padStart(2, '0')}:${minute}`
  return `${padPair(match[1], match[2])}–${padPair(match[3], match[4])}`
}
