export const toLocalYMD = (date) => {
  const d = date instanceof Date ? date : new Date(date)
  const y = d.getFullYear()
  const mo = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${mo}-${day}`
}

export const monthRange = (anchorDate) => {
  const d = anchorDate instanceof Date ? anchorDate : new Date(`${anchorDate}T12:00:00`)
  const from = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`
  const last = new Date(d.getFullYear(), d.getMonth() + 1, 0)
  return { from, to: toLocalYMD(last) }
}

export const resolveFortuneImageSrc = (imageUrl) => {
  if (!imageUrl) return ''
  if (imageUrl.startsWith('http://') || imageUrl.startsWith('https://')) return imageUrl
  return imageUrl
}

export const buildFortuneMonthMap = (items) => {
  const map = {}
  for (const row of items || []) {
    if (row?.date) map[row.date] = row
  }
  return map
}
