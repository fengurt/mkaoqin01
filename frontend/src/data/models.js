export const STATUS_LABEL = {
  CHECK_IN: '签到',
  OFFICE: '在岗',
  OUTING: '外出拜访',
  DINING: '商务用餐',
  BUSINESS_TRIP: '出差',
  CHECK_OUT: '签退',
  OFFLINE: '离线',
}

export const STATUS_ICON = {
  CHECK_IN: 'login',
  OFFICE: 'apartment',
  OUTING: 'directions_run',
  DINING: 'restaurant',
  BUSINESS_TRIP: 'flight_takeoff',
  CHECK_OUT: 'logout',
  OFFLINE: 'warning',
}

export const PERIOD_OPTIONS = [
  { key: 'day', label: '日视图' },
  { key: 'week', label: '周视图' },
  { key: 'month', label: '月视图' },
]

export const normalizeAttendanceRecord = (row) => ({
  id: row.id,
  userId: row.userId ?? 1,
  status: row.status || 'OFFICE',
  statusLabel: STATUS_LABEL[row.status] || row.status || '未知',
  icon: STATUS_ICON[row.status] || 'event',
  location: row.location || '未上报地点',
  reason: row.reason || '无备注',
  occurredAt: row.occurredAt || '',
})

export const normalizeTeamMember = (row) => ({
  userId: row.userId,
  userName: row.userName || '未知员工',
  status: row.status || 'OFFLINE',
  statusLabel: STATUS_LABEL[row.status] || row.status || '未知',
  icon: STATUS_ICON[row.status] || 'warning',
  location: row.location || '未上报地点',
  reason: row.reason || '无备注',
  occurredAt: row.occurredAt || '',
})

export const formatClock = (dateText) => (dateText ? new Date(dateText).toLocaleTimeString() : '-')
export const formatDateTime = (dateText) => (dateText ? new Date(dateText).toLocaleString() : '-')
