/** 与后端 normalizePersonName 一致：用于导入时姓名模糊匹配 */
export function normalizePersonName(name) {
  return String(name || '')
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fff]/g, '')
}

export function buildUsersCatalog(authUsers) {
  return (authUsers || []).map((row) => ({
    userId: Number(row.id ?? row.userId) || 0,
    account: String(row.account || ''),
    displayName: String(row.displayName || row.display_name || ''),
    role: String(row.role || 'employee'),
    normalizedName: normalizePersonName(row.displayName || row.display_name),
  })).filter((u) => u.userId > 0)
}

/** 排班名册：非管理员账号均纳入（与团队考勤一致） */
export function buildEmployees(users) {
  return (users || []).filter((u) => String(u.role || '').toLowerCase() !== 'admin')
}

export function buildChronoscapeRoster(employees) {
  return (employees || []).map((u, index) => ({
    simObjectId: 70000 + u.userId,
    objectId: u.userId,
    objectName: u.displayName,
    account: u.account,
    sortOrder: index + 1,
    isActive: true,
  }))
}

const DEFAULT_IMPORT_HINTS = {
  matchUserBy: ['userId', 'account', 'displayName', 'objectName', 'normalizedName'],
  matchShiftBy: ['tagItemName', 'code', 'mode+code'],
  chronoscapeCells: 'cells[] with objectName + date + tagItemName also supported on import',
}

/**
 * 若后端仍为旧版导出（无 users），用管理员账号列表 API 补全 users/employees/roster。
 */
export async function enrichScheduleGridExport(exportData, loadAuthUsers) {
  const data = exportData && typeof exportData === 'object' ? { ...exportData } : {}
  const existingUsers = Array.isArray(data.users) ? data.users : []
  if (existingUsers.length > 0) {
    if (!Array.isArray(data.employees) || data.employees.length === 0) {
      data.employees = buildEmployees(existingUsers)
    }
    if (!Array.isArray(data.roster) || data.roster.length === 0) {
      data.roster = buildChronoscapeRoster(data.employees)
    }
    return data
  }
  if (typeof loadAuthUsers !== 'function') {
    return data
  }
  const authUsers = await loadAuthUsers()
  const users = buildUsersCatalog(authUsers)
  data.users = users
  data.employees = buildEmployees(users)
  data.roster = buildChronoscapeRoster(data.employees)
  data.schemaVersion = data.schemaVersion || 2
  data.importHints = data.importHints || DEFAULT_IMPORT_HINTS
  data._usersEnrichedFrom = 'auth/users'
  return data
}
