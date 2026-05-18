<template>
  <div class="team-week-shell">
    <header class="team-week-head">
      <h3>{{ title }}</h3>
      <button type="button" class="team-week-close" @click="$emit('close')">关闭</button>
    </header>

    <div class="team-week-toolbar">
      <button type="button" class="team-week-nav" aria-label="上一周" @click="shiftWeek(-1)">
        <span class="material-symbols-outlined">chevron_left</span>
      </button>
      <div class="team-week-range">
        <strong>{{ weekRangeLabel }}</strong>
        <button v-if="!isCurrentWeek" type="button" class="team-week-today" @click="goCurrentWeek">本周</button>
      </div>
      <button type="button" class="team-week-nav" aria-label="下一周" @click="shiftWeek(1)">
        <span class="material-symbols-outlined">chevron_right</span>
      </button>
    </div>

    <div v-if="showLegend" class="team-week-legend" aria-label="班次图例">
      <span
        v-for="item in SCHEDULE_TAG_LEGEND"
        :key="item.code"
        class="team-week-legend-chip"
        :class="`team-week-legend-chip--${item.kind}`"
      >
        {{ item.label }}
      </span>
    </div>

    <div class="team-week-grid-wrap">
      <van-loading v-if="loading" class="team-week-loading" type="spinner" color="#2c5ee8" />
      <div class="team-week-scroll">
        <table class="team-week-table" :aria-busy="loading">
          <thead>
            <tr>
              <th scope="col" class="col-sticky-name">员工</th>
              <th
                v-for="col in weekColumns"
                :key="col.date"
                scope="col"
                class="col-date"
                :class="{ 'col-date--today': col.isToday }"
              >
                <button type="button" class="col-date-btn" @click="onPickDate(col.date)">
                  <span class="col-weekday">{{ col.weekday }}</span>
                  <span class="col-daynum">{{ col.dayNum }}</span>
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rosterRows" :key="row.displayName" :class="{ 'row-unmatched': !row.matched }">
              <th scope="row" class="col-sticky-name">
                <span class="row-name">{{ row.displayName }}</span>
                <span v-if="row.roleLabel" class="row-role" :class="`row-role--${row.role}`">{{ row.roleLabel }}</span>
              </th>
              <td
                v-for="col in weekColumns"
                :key="`${row.displayName}-${col.date}`"
                class="col-cell"
                :class="cellClass(row, col.date)"
              >
                <template v-if="row.matched">
                  <span class="cell-abbr">{{ cellFor(row.userId, col.date).abbr }}</span>
                  <span v-if="cellFor(row.userId, col.date).time" class="cell-time">{{ cellFor(row.userId, col.date).time }}</span>
                </template>
                <span v-else class="cell-muted">未匹配</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <p class="team-week-hint">横轴为日期，纵轴为员工；左右滑动查看完整表格</p>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { getAuthUsers, getScheduleMonth } from '../api'
import { buildUsersCatalog } from '../lib/scheduleGridExport'
import { SCHEDULE_TAG_LEGEND, formatScheduleGridCell } from '../lib/scheduleCalendarLabel'
import { resolveTeamScheduleRoster } from '../lib/teamScheduleRoster'
import { toLocalYMD } from '../lib/fortuneUtils'

const props = defineProps({
  weekStart: { type: String, default: '' },
  active: { type: Boolean, default: false },
  title: { type: String, default: '团队排班周视图' },
  showLegend: { type: Boolean, default: true },
})

const emit = defineEmits(['update:weekStart', 'select-date', 'close'])

/** 固定名册先渲染，避免弹窗懒加载时 watch 未触发导致表格为空 */
const rosterRows = ref(resolveTeamScheduleRoster([]))
const weekStartLocal = ref('')
const scheduleByUserDate = ref({})
const loading = ref(false)

function getWeekStart(dateInput) {
  const currentDate = new Date(dateInput)
  const day = currentDate.getDay()
  const shift = day === 0 ? -6 : 1 - day
  currentDate.setDate(currentDate.getDate() + shift)
  return toLocalYMD(currentDate)
}

const currentWeekStart = computed(() => getWeekStart(new Date()))

const isCurrentWeek = computed(() => weekStartLocal.value === currentWeekStart.value)

const weekColumns = computed(() => {
  const base = new Date(`${weekStartLocal.value || currentWeekStart.value}T00:00:00`)
  const todayStr = toLocalYMD(new Date())
  const weekdayLabels = ['一', '二', '三', '四', '五', '六', '日']
  return Array.from({ length: 7 }).map((_, index) => {
    const date = new Date(base)
    date.setDate(base.getDate() + index)
    const dateStr = toLocalYMD(date)
    return {
      date: dateStr,
      weekday: weekdayLabels[index],
      dayNum: String(date.getDate()),
      isToday: dateStr === todayStr,
    }
  })
})

const weekRangeLabel = computed(() => {
  const first = weekColumns.value[0]?.date || ''
  const last = weekColumns.value[6]?.date || ''
  if (!first || !last) return ''
  return `${first} ~ ${last}`
})

const monthsForWeek = computed(() => {
  const months = new Set()
  weekColumns.value.forEach((col) => {
    if (col.date) months.add(col.date.slice(0, 7))
  })
  return [...months]
})

const cellFor = (userId, ymd) => {
  const key = `${userId}:${ymd}`
  return scheduleByUserDate.value[key] || { abbr: '—', time: '', className: '' }
}

const cellClass = (row, ymd) => {
  if (!row.matched) return 'col-cell--muted'
  const cell = cellFor(row.userId, ymd)
  return cell.className ? `col-cell--${cell.className.replace('schedule-cal-day--', '')}` : ''
}

const resolveAuthUsersList = (payload) => {
  if (!payload || typeof payload !== 'object') return []
  const raw = payload.items ?? payload.users ?? payload.data
  return Array.isArray(raw) ? raw : []
}

const loadAuthUsers = async () => {
  const { data } = await getAuthUsers()
  return buildUsersCatalog(resolveAuthUsersList(data))
}

const loadUserMonth = async (userId, month) => {
  const { data } = await getScheduleMonth(userId, month)
  return data?.days || {}
}

const loadWeekGrid = async () => {
  loading.value = true
  rosterRows.value = resolveTeamScheduleRoster([])
  try {
    const users = await loadAuthUsers()
    rosterRows.value = resolveTeamScheduleRoster(users)
    const nextCells = {}
    const matchedRows = rosterRows.value.filter((row) => row.userId > 0)
    const tasks = []
    for (const row of matchedRows) {
      for (const month of monthsForWeek.value) {
        tasks.push(async () => {
          const days = await loadUserMonth(row.userId, month)
          weekColumns.value.forEach((col) => {
            if (!col.date.startsWith(month)) return
            const dayInfo = days[col.date]
            nextCells[`${row.userId}:${col.date}`] = formatScheduleGridCell(dayInfo)
          })
        })
      }
    }
    const poolSize = 8
    let cursor = 0
    const workers = Array.from({ length: Math.min(poolSize, tasks.length) }, async () => {
      while (cursor < tasks.length) {
        const task = tasks[cursor]
        cursor += 1
        await task()
      }
    })
    await Promise.all(workers)
    scheduleByUserDate.value = nextCells
  } catch {
    rosterRows.value = resolveTeamScheduleRoster([])
    scheduleByUserDate.value = {}
  } finally {
    loading.value = false
  }
}

const shiftWeek = async (delta) => {
  const next = new Date(`${weekStartLocal.value}T00:00:00`)
  next.setDate(next.getDate() + delta * 7)
  weekStartLocal.value = toLocalYMD(next)
  emit('update:weekStart', weekStartLocal.value)
  await loadWeekGrid()
}

const goCurrentWeek = async () => {
  weekStartLocal.value = currentWeekStart.value
  emit('update:weekStart', weekStartLocal.value)
  await loadWeekGrid()
}

const onPickDate = (ymd) => {
  emit('select-date', ymd)
  emit('close')
}

const syncWeekStart = () => {
  weekStartLocal.value = props.weekStart || currentWeekStart.value
}

watch(
  () => props.active,
  async (open) => {
    if (!open) return
    syncWeekStart()
    await loadWeekGrid()
  },
  { immediate: true },
)

watch(
  () => props.weekStart,
  async (value) => {
    if (!props.active || !value || value === weekStartLocal.value) return
    weekStartLocal.value = value
    await loadWeekGrid()
  },
)
</script>

<style scoped>
.team-week-shell {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 10px 10px 12px;
  background: #f6f8ff;
  box-sizing: border-box;
}
.team-week-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  flex-shrink: 0;
}
.team-week-head h3 {
  margin: 0;
  font-size: 17px;
  color: #102a5c;
}
.team-week-close {
  border: 1px solid #d8e0f5;
  background: #fff;
  color: #2c5ee8;
  border-radius: 999px;
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
}
.team-week-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.team-week-nav {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border: 1px solid #d8e0f5;
  border-radius: 999px;
  background: #fff;
  color: #2c5ee8;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.team-week-range {
  flex: 1;
  min-width: 0;
  text-align: center;
  font-size: 13px;
  color: #102a5c;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}
.team-week-range strong {
  font-weight: 700;
}
.team-week-today {
  border: 0;
  background: #edf2ff;
  color: #2c5ee8;
  font-size: 11px;
  font-weight: 700;
  border-radius: 999px;
  padding: 2px 10px;
  cursor: pointer;
}
.team-week-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.team-week-legend-chip {
  font-size: 9px;
  font-weight: 700;
  padding: 2px 5px;
  border-radius: 5px;
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #475569;
}
.team-week-legend-chip--leave {
  background: #fef3c7;
  border-color: #fcd34d;
  color: #92400e;
}
.team-week-legend-chip--work {
  background: #eff6ff;
  border-color: #bfdbfe;
  color: #1d4ed8;
}
.team-week-grid-wrap {
  position: relative;
  flex: 1;
  min-height: 200px;
  display: flex;
  flex-direction: column;
  border: 1px solid #d8e0f5;
  border-radius: 12px;
  background: #fff;
  overflow: hidden;
}
.team-week-loading {
  position: absolute;
  inset: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.72);
}
.team-week-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  -webkit-overflow-scrolling: touch;
}
.team-week-table {
  width: max-content;
  min-width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 11px;
}
.team-week-table th,
.team-week-table td {
  border-bottom: 1px solid #e8edf8;
  border-right: 1px solid #e8edf8;
  padding: 6px 4px;
  vertical-align: middle;
  text-align: center;
}
.col-sticky-name {
  position: sticky;
  left: 0;
  z-index: 2;
  min-width: 108px;
  max-width: 108px;
  background: #fafcff;
  text-align: left;
  padding-left: 8px !important;
  box-shadow: 2px 0 6px rgba(15, 40, 120, 0.06);
}
.team-week-table thead th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #f1f5ff;
  font-weight: 700;
  color: #102a5c;
}
.team-week-table thead .col-sticky-name {
  z-index: 3;
}
.col-date {
  min-width: 72px;
}
.col-date--today .col-daynum {
  color: #c2410c;
}
.col-date-btn {
  border: 0;
  background: transparent;
  padding: 0;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  color: inherit;
  font: inherit;
  cursor: pointer;
}
.col-weekday {
  font-size: 10px;
  color: #64748b;
}
.col-daynum {
  font-size: 14px;
  font-weight: 800;
}
.row-name {
  display: block;
  font-size: 11px;
  font-weight: 700;
  color: #102a5c;
  line-height: 1.25;
}
.row-role {
  display: inline-block;
  margin-top: 3px;
  font-size: 9px;
  font-weight: 700;
  border-radius: 4px;
  padding: 1px 4px;
  line-height: 1.2;
}
.row-role--overall {
  background: #fef3c7;
  color: #92400e;
}
.row-role--lead {
  background: #dbeafe;
  color: #1d4ed8;
}
.row-unmatched .row-name {
  color: #94a3b8;
}
.col-cell {
  min-width: 72px;
  max-width: 88px;
}
.cell-abbr {
  display: block;
  font-weight: 800;
  color: #1e3a8a;
  line-height: 1.2;
  word-break: break-word;
}
.cell-time {
  display: block;
  margin-top: 2px;
  font-size: 9px;
  color: #64748b;
  line-height: 1.2;
}
.cell-muted {
  color: #cbd5e1;
  font-size: 10px;
}
.col-cell--leave .cell-abbr {
  color: #b45309;
}
.col-cell--standby .cell-abbr {
  color: #6d28d9;
}
.col-cell--muted {
  background: #fafafa;
}
.team-week-hint {
  margin: 8px 2px 0;
  font-size: 11px;
  color: #94a3b8;
  text-align: center;
  flex-shrink: 0;
}
</style>
