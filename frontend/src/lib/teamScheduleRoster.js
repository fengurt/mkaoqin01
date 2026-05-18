import { normalizePersonName } from './scheduleGridExport'

/** 团队排班周视图固定名册（顺序与角色标签） */
export const TEAM_SCHEDULE_ROSTER = [
  { displayName: 'Justin Lu', role: 'overall', roleLabel: '总负责' },
  { displayName: 'Albee Liu', role: 'lead', roleLabel: '团队负责人' },
  { displayName: 'Betty Zhang', role: 'member', roleLabel: '' },
  { displayName: 'Heather Zou', role: 'member', roleLabel: '' },
  { displayName: 'Simon Kok', role: 'member', roleLabel: '' },
  { displayName: 'Sonia Song', role: 'member', roleLabel: '' },
  { displayName: 'Ashley Lei', role: 'member', roleLabel: '' },
  { displayName: 'Isaac Su', role: 'lead', roleLabel: '团队负责人' },
  { displayName: 'Kalei Kong', role: 'member', roleLabel: '' },
  { displayName: 'Simon Wu', role: 'member', roleLabel: '' },
  { displayName: 'Emily Li', role: 'member', roleLabel: '' },
  { displayName: 'Max Wang', role: 'member', roleLabel: '' },
  { displayName: 'Stacey Pong', role: 'member', roleLabel: '' },
  { displayName: 'Owen Liang', role: 'member', roleLabel: '' },
  { displayName: 'Bella Guo', role: 'lead', roleLabel: '团队负责人' },
  { displayName: 'Elva Ao', role: 'member', roleLabel: '' },
  { displayName: 'Sky Wang', role: 'member', roleLabel: '' },
  { displayName: 'William Chen', role: 'member', roleLabel: '' },
  { displayName: 'Joyce Yi', role: 'member', roleLabel: '' },
  { displayName: 'Leah Zhou', role: 'member', roleLabel: '' },
  { displayName: 'Jeremy Cai', role: 'member', roleLabel: '' },
  { displayName: 'Vicky Yue', role: 'lead', roleLabel: '团队负责人' },
  { displayName: 'SiSi Sou', role: 'member', roleLabel: '' },
  { displayName: 'Sammi Xian', role: 'member', roleLabel: '' },
  { displayName: 'Duke Sui', role: 'member', roleLabel: '' },
]

function findUserForRosterEntry(entry, usersCatalog) {
  const targetNorm = normalizePersonName(entry.displayName)
  if (!targetNorm) return null
  const exact = usersCatalog.find((user) => normalizePersonName(user.displayName) === targetNorm)
  if (exact) return exact
  return (
    usersCatalog.find((user) => {
      const norm = normalizePersonName(user.displayName)
      return norm && (norm.includes(targetNorm) || targetNorm.includes(norm))
    }) || null
  )
}

/** 将固定名册与账号列表合并为带 userId 的行 */
export function resolveTeamScheduleRoster(usersCatalog) {
  const catalog = Array.isArray(usersCatalog) ? usersCatalog : []
  return TEAM_SCHEDULE_ROSTER.map((entry) => {
    const matched = findUserForRosterEntry(entry, catalog)
    return {
      ...entry,
      userId: matched?.userId || 0,
      account: matched?.account || '',
      matched: Boolean(matched?.userId),
    }
  })
}
