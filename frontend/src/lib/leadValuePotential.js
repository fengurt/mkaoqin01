/**
 * 高价值潜力雷达 — 画像维度与评分算法（前端确定性规则，便于与 CRM 规则对齐后迁服务端）。
 *
 * 六轴（0–100）：综合澳门综合度假村场景，用已有结构化字段做可解释的加权指数，非机器学习模型。
 *
 * 1) regionalValue — 客源地与澳门核心腹地距离（大湾区 / 香港 / 内地一线渗透等）
 * 2) propertyStrategicFit — 偏好场域与 MGM 双物业战略匹配度
 * 3) groupScale — 预计人数带来的宴会 / 餐饮 / 包厢规模机会
 * 4) occasionValue — 场景商业性（婚宴 / 贵宾礼遇 / 会议等）
 * 5) channelTrust — 触达渠道可信度（主机推荐 > 官微 > OTA …）
 * 6) relationshipEquity — 关系资产：老客层级 + 参考字段完整度；新客则用优先级与渠道作代理信号
 *
 * composite：六轴加权算术平均（权重和为 1），四舍五入为 0–100 整数。
 */

export const LEAD_RADAR_AXIS_KEYS = [
  'regionalValue',
  'propertyStrategicFit',
  'groupScale',
  'occasionValue',
  'channelTrust',
  'relationshipEquity',
]

const WEIGHTS = {
  regionalValue: 0.18,
  propertyStrategicFit: 0.18,
  groupScale: 0.15,
  occasionValue: 0.2,
  channelTrust: 0.14,
  relationshipEquity: 0.15,
}

function toIntScore(x) {
  return Math.max(0, Math.min(100, Math.round(x)))
}

function scoreRegional(origin) {
  const table = {
    GBA: 92,
    HK: 88,
    MAINLAND: 76,
    MACAU_LOCAL: 86,
    SEA: 54,
    INTL: 48,
    UNKNOWN: 36,
  }
  return table[origin] ?? table.UNKNOWN
}

function scoreVenue(venue) {
  const table = {
    MGM_COTAI: 96,
    MGM_PENINSULA: 96,
    OTHER_MACAU_IR: 72,
    UNSPECIFIED: 42,
  }
  return table[venue] ?? table.UNSPECIFIED
}

function scorePartySize(raw) {
  if (raw == null || raw === '') return 44
  const n = Number(raw)
  if (!Number.isFinite(n) || n <= 0) return 44
  if (n <= 2) return 52
  if (n <= 6) return 68
  if (n <= 12) return 84
  if (n <= 24) return 92
  return 98
}

function scoreOccasion(occ) {
  const table = {
    WEDDING_BANQUET: 94,
    VIP_TABLE: 91,
    CONFERENCE: 79,
    LEISURE: 56,
    OTHER: 50,
  }
  return table[occ] ?? table.OTHER
}

function scoreChannel(ch) {
  const table = {
    HOST_REFERRAL: 96,
    WECHAT_OFFICIAL: 76,
    PARTNER: 71,
    OTA: 54,
    WALK_IN: 52,
    UNKNOWN: 40,
  }
  return table[ch] ?? table.UNKNOWN
}

function monthsSinceIso(iso) {
  if (!iso || typeof iso !== 'string') return null
  const t = Date.parse(iso.includes('T') ? iso : `${iso}T12:00:00`)
  if (Number.isNaN(t)) return null
  const diff = Date.now() - t
  return diff / (1000 * 60 * 60 * 24 * 30.44)
}

function scoreLtvTier(tier) {
  if (!tier || typeof tier !== 'string') return 0
  const u = tier.toUpperCase()
  if (u.includes('PLAT') || u.includes('黑')) return 32
  if (u.includes('GOLD') || u.includes('金')) return 26
  if (u.includes('SILV') || u.includes('银')) return 18
  if (u.includes('BRON') || u.includes('铜')) return 12
  return 8
}

function scoreRelationship(lead) {
  const seg = lead.leadSegment || 'NEW_PURE'
  const pr = (lead.priority || 'NORMAL').toUpperCase()
  let base = 52
  if (pr === 'URGENT') base += 16
  else if (pr === 'HIGH') base += 11
  else if (pr === 'NORMAL') base += 4

  if (seg === 'OLD_REACTIVATION') {
    let s = 58 + scoreLtvTier(lead.refLtvTier)
    const mo = monthsSinceIso(lead.refLastVisitAt)
    if (mo != null) {
      if (mo <= 18) s += 14
      else if (mo <= 36) s += 8
      else if (mo <= 60) s += 3
    }
    const filled = [lead.refLastProperty, lead.refHostName, lead.refMemberIdMasked, lead.refNotes].filter(
      (x) => x && String(x).trim(),
    ).length
    s += Math.min(12, filled * 3.5)
    return toIntScore(s)
  }

  const ch = lead.channelTouchpoint || 'UNKNOWN'
  if (ch === 'HOST_REFERRAL') base += 12
  if (ch === 'WECHAT_OFFICIAL') base += 6
  return toIntScore(base)
}

/**
 * @param {Record<string, unknown>} lead — API 线索对象（camelCase）
 * @returns {{ byAxis: Record<string, number>, composite: number, axisOrder: string[] }}
 */
export function computeLeadValueRadar(lead) {
  if (!lead || typeof lead !== 'object') {
    const zeros = Object.fromEntries(LEAD_RADAR_AXIS_KEYS.map((k) => [k, 0]))
    return { byAxis: zeros, composite: 0, axisOrder: [...LEAD_RADAR_AXIS_KEYS] }
  }

  const byAxis = {
    regionalValue: toIntScore(scoreRegional(lead.approxOriginRegion || 'UNKNOWN')),
    propertyStrategicFit: toIntScore(scoreVenue(lead.preferredVenue || 'UNSPECIFIED')),
    groupScale: toIntScore(scorePartySize(lead.estimatedPartySize)),
    occasionValue: toIntScore(scoreOccasion(lead.eventOccasion || 'OTHER')),
    channelTrust: toIntScore(scoreChannel(lead.channelTouchpoint || 'UNKNOWN')),
    relationshipEquity: scoreRelationship(lead),
  }

  let composite = 0
  for (const key of LEAD_RADAR_AXIS_KEYS) {
    composite += (byAxis[key] ?? 0) * (WEIGHTS[key] ?? 0)
  }
  composite = toIntScore(composite)

  return { byAxis, composite, axisOrder: [...LEAD_RADAR_AXIS_KEYS] }
}

/**
 * @param {Record<string, unknown>} lead
 * @returns {number[]} 与 LEAD_RADAR_AXIS_KEYS 同序的 0–100 分
 */
export function leadRadarScoresArray(lead) {
  const { byAxis } = computeLeadValueRadar(lead)
  return LEAD_RADAR_AXIS_KEYS.map((k) => byAxis[k] ?? 0)
}
