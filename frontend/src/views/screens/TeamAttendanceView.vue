<template>
  <div class="team-page">
    <header class="team-topbar">
      <button class="back-btn" type="button" @click="$router.push('/me')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>团队考勤</h1>
      <button class="month-btn" type="button" @click="showMonthPicker = true">{{ currentMonthLabel }}</button>
    </header>

    <main class="team-content">
      <section class="week-card week-card--interactive" @click="onWeekCardClick">
        <div class="week-switch">
          <button class="icon-btn" type="button" @click.stop="switchWeek(-1)">
            <span class="material-symbols-outlined">chevron_left</span>
          </button>
          <button type="button" class="week-title week-title-btn" @click.stop="openTeamScheduleGrid">
            当前周 {{ weekRangeLabel }}
            <span class="material-symbols-outlined week-title-icon" aria-hidden="true">grid_on</span>
          </button>
          <button class="icon-btn" type="button" @click.stop="switchWeek(1)">
            <span class="material-symbols-outlined">chevron_right</span>
          </button>
        </div>
        <div class="week-days">
          <button
            v-for="day in weekDays"
            :key="day.date"
            class="day-pill"
            :class="{ active: day.date === selectedDate, today: day.isToday }"
            @click.stop="selectDate(day.date)"
          >
            <span>{{ day.weekday }}</span>
            <strong>{{ day.dayNumber }}</strong>
          </button>
        </div>
      </section>

      <section class="grid-io-card grid-io-card--compact">
        <div class="grid-io-head">
          <h2 class="grid-io-title">班次导入 / 导出</h2>
          <button type="button" class="grid-io-guide-btn" @click="showGridGuide = true">
            <span class="material-symbols-outlined" aria-hidden="true">menu_book</span>
            Agent 指南
          </button>
        </div>
        <div class="grid-io-toolbar">
          <van-field v-model="gridFrom" class="grid-io-field" label="始" placeholder="YYYY-MM-DD" :border="false" />
          <span class="grid-io-sep" aria-hidden="true">—</span>
          <van-field v-model="gridTo" class="grid-io-field" label="终" placeholder="YYYY-MM-DD" :border="false" />
          <van-button size="small" type="primary" plain :loading="exportGridBusy" @click="exportScheduleGrid">导出</van-button>
          <van-button size="small" type="primary" :loading="importGridBusy" @click="triggerGridImport">导入</van-button>
        </div>
        <input ref="gridFileRef" type="file" accept=".json,application/json" class="hidden-file" @change="onGridFileSelected" />
      </section>

      <ScheduleGridGuidePopup v-model="showGridGuide" />

      <section class="summary-card">
        <h2>当日团队考勤汇报</h2>
        <div class="kpi-grid">
          <div class="kpi-item"><span>团队总人数</span><strong>{{ summary.totalMembers }}</strong></div>
          <div class="kpi-item"><span>已汇报人数</span><strong>{{ summary.reportedMembers }}</strong></div>
          <div class="kpi-item"><span>拜访客户数</span><strong>{{ summary.clientVisitCount }}</strong></div>
          <div class="kpi-item"><span>餐厅用餐数</span><strong>{{ summary.diningCount }}</strong></div>
          <div class="kpi-item"><span>外出次数</span><strong>{{ summary.outingCount }}</strong></div>
          <div class="kpi-item"><span>访问客户数</span><strong>{{ summary.accessClientCount }}</strong></div>
        </div>
      </section>

      <section class="member-card">
        <div class="member-head">
          <h3>员工当日汇总</h3>
          <div class="member-nav">
            <button class="icon-btn" type="button" @click="moveSelected(-1)">
              <span class="material-symbols-outlined">chevron_left</span>
            </button>
            <button class="icon-btn" type="button" @click="moveSelected(1)">
              <span class="material-symbols-outlined">chevron_right</span>
            </button>
          </div>
        </div>
        <div class="avatar-row">
          <button
            v-for="(member, index) in team"
            :key="member.userId"
            class="avatar-item"
            :class="{ active: index === selectedMemberIndex }"
            @click="selectedMemberIndex = index"
            @dblclick="goMemberHistory(member)"
          >
            <span>{{ memberInitial(member.userName) }}</span>
            <small>{{ shortName(member.userName) }}</small>
            <em class="avatar-declare" :class="`avatar-declare--${memberDeclareState(member.userId).key}`">
              {{ memberDeclareState(member.userId).text }}
            </em>
          </button>
        </div>
        <div v-if="selectedMember" class="member-detail">
          <div class="detail-head">
            <h4>{{ selectedMember.userName }}</h4>
            <button class="detail-link" type="button" @click="goMemberHistory(selectedMember)">
              查看历史汇总
              <span class="material-symbols-outlined">arrow_forward</span>
            </button>
          </div>
          <div class="behavior-grid">
            <div class="behavior-item">
              <span>签到次数</span>
              <strong>{{ selectedBehaviorSummary.checkInCount }}</strong>
            </div>
            <div class="behavior-item">
              <span>签退次数</span>
              <strong>{{ selectedBehaviorSummary.checkOutCount }}</strong>
            </div>
            <div class="behavior-item">
              <span>外出/出差</span>
              <strong>{{ selectedBehaviorSummary.outingLikeCount }}</strong>
            </div>
            <div class="behavior-item">
              <span>用餐记录</span>
              <strong>{{ selectedBehaviorSummary.diningCount }}</strong>
            </div>
          </div>
          <div class="mini-timeline">
            <div class="mini-title">当日行为记录（可无时间）</div>
            <div v-if="selectedMemberRecords.length === 0" class="timeline-empty">当日暂无行为记录</div>
            <div v-for="record in selectedMemberRecords" :key="record.id" class="mini-item">
              <span class="mini-time">{{ clockText(record.occurredAt) }}</span>
              <span class="mini-status">{{ record.statusLabel }}</span>
              <span class="mini-desc">{{ record.reason }} · {{ record.location }}</span>
            </div>
          </div>
        </div>
      </section>
    </main>

    <van-popup
      v-model:show="showTeamScheduleGrid"
      position="bottom"
      round
      teleport="body"
      :z-index="2600"
      :style="{ height: '92%' }"
      :close-on-click-overlay="true"
      safe-area-inset-bottom
    >
      <TeamScheduleWeekGrid
        :active="showTeamScheduleGrid"
        :week-start="weekStartDate"
        title="团队排班周视图"
        :show-legend="true"
        @update:week-start="weekStartDate = $event"
        @select-date="onTeamSchedulePickDate"
        @close="showTeamScheduleGrid = false"
      />
    </van-popup>

    <van-popup v-model:show="showMonthPicker" position="bottom" round :style="{ height: '42%' }">
      <div class="month-popup">
        <h3>选择月份</h3>
        <div class="month-grid">
          <button v-for="month in monthOptions" :key="month.value" class="month-cell" @click="pickMonth(month.value)">
            {{ month.label }}
          </button>
        </div>
      </div>
    </van-popup>

    <AppBottomNav />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { showFailToast, showSuccessToast } from 'vant'
import AppBottomNav from '../../components/AppBottomNav.vue'
import ScheduleGridGuidePopup from '../../components/ScheduleGridGuidePopup.vue'
import TeamScheduleWeekGrid from '../../components/TeamScheduleWeekGrid.vue'
import { readApiErrorMessage } from '../../lib/apiError'
import { toLocalYMD } from '../../lib/fortuneUtils'
import {
  exportAdminScheduleGrid,
  getAdminTeam,
  getAttendanceByDate,
  getAuthUsers,
  getScheduleDay,
  importAdminScheduleGrid,
} from '../../api'
import { enrichScheduleGridExport } from '../../lib/scheduleGridExport'
import { normalizeAttendanceRecord, normalizeTeamMember } from '../../data/models'

const DECLARE_RECORD_STATUSES = ['OFFICE', 'OUTING', 'DINING', 'BUSINESS_TRIP']

const team = ref([])
const selectedDate = ref(toLocalYMD(new Date()))
const showTeamScheduleGrid = ref(false)
const showGridGuide = ref(false)
const weekStartDate = ref(getWeekStart(new Date()))
const selectedMemberIndex = ref(0)
const showMonthPicker = ref(false)
const selectedMemberRecords = ref([])
const memberDeclareStateByUserId = ref({})
const router = useRouter()

const monthBounds = () => {
  const d = new Date(`${selectedDate.value}T12:00:00`)
  const from = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`
  const last = new Date(d.getFullYear(), d.getMonth() + 1, 0)
  const to = `${last.getFullYear()}-${String(last.getMonth() + 1).padStart(2, '0')}-${String(last.getDate()).padStart(2, '0')}`
  return { from, to }
}

const gridFrom = ref(monthBounds().from)
const gridTo = ref(monthBounds().to)
const exportGridBusy = ref(false)
const importGridBusy = ref(false)
const gridFileRef = ref(null)

const resolveAuthUsersList = (payload) => {
  if (!payload || typeof payload !== 'object') return []
  const raw = payload.items ?? payload.users ?? payload.data
  return Array.isArray(raw) ? raw : []
}

const loadAuthUsersForExport = async () => {
  const { data } = await getAuthUsers()
  return resolveAuthUsersList(data)
}

const exportScheduleGrid = async () => {
  const from = gridFrom.value.trim()
  const to = gridTo.value.trim()
  if (!/^\d{4}-\d{2}-\d{2}$/.test(from) || !/^\d{4}-\d{2}-\d{2}$/.test(to)) {
    showFailToast('请填写正确的开始/结束日期（YYYY-MM-DD）')
    return
  }
  if (from > to) {
    showFailToast('开始日期不能晚于结束日期')
    return
  }
  exportGridBusy.value = true
  try {
    const { data: rawExport } = await exportAdminScheduleGrid(from, to)
    if (!rawExport || typeof rawExport !== 'object') {
      showFailToast('导出数据无效')
      return
    }
    let data
    try {
      data = await enrichScheduleGridExport(rawExport, loadAuthUsersForExport)
    } catch {
      showFailToast('账号列表加载失败，无法补全员工名册')
      return
    }
    const employees = Array.isArray(data.employees) ? data.employees : []
    if (employees.length === 0) {
      showFailToast('未获取到员工账号，请确认已用管理员登录')
      return
    }
    const jsonText = JSON.stringify(data, null, 2)
    const blob = new Blob([jsonText], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `schedule-grid-${from}_${to}.json`
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
    const assignCount = Array.isArray(data.assignments) ? data.assignments.length : 0
    const userCount = Array.isArray(data.users) ? data.users.length : 0
    const empCount = employees.length
    const enrichedNote = data._usersEnrichedFrom ? '（已合并账号列表）' : ''
    showSuccessToast(`已导出：${assignCount} 条排班 · ${userCount} 个账号（员工 ${empCount}）${enrichedNote}`)
  } catch (error) {
    const status = error?.response?.status
    if (status === 404) {
      showFailToast('导出接口不可用，请重新运行 scripts/dev-up.sh 重启后端')
    } else {
      showFailToast(await readApiErrorMessage(error, '导出失败'))
    }
  } finally {
    exportGridBusy.value = false
  }
}

const triggerGridImport = () => gridFileRef.value?.click()

const onGridFileSelected = async (event) => {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  importGridBusy.value = true
  try {
    const text = await file.text()
    const payload = JSON.parse(text)
    const { data } = await importAdminScheduleGrid(payload)
    const applied = data.applied ?? 0
    const skipped = data.skipped ?? 0
    const unmatched = Array.isArray(data.unmatched) ? data.unmatched.length : 0
    showSuccessToast(`导入完成：写入 ${applied} 条，跳过 ${skipped}，未匹配 ${unmatched}`)
    await loadData()
  } catch (error) {
    showFailToast(await readApiErrorMessage(error, '导入失败'))
  } finally {
    importGridBusy.value = false
  }
}

const monthOptions = computed(() => {
  const year = new Date(selectedDate.value).getFullYear()
  return Array.from({ length: 12 }).map((_, index) => ({
    value: `${year}-${String(index + 1).padStart(2, '0')}-01`,
    label: `${index + 1}月`,
  }))
})

const currentMonthLabel = computed(() => `${new Date(selectedDate.value).getMonth() + 1}月`)

const weekDays = computed(() => {
  const base = new Date(`${weekStartDate.value}T00:00:00`)
  return Array.from({ length: 7 }).map((_, index) => {
    const date = new Date(base)
    date.setDate(base.getDate() + index)
    const dateText = toLocalYMD(date)
    const todayText = toLocalYMD(new Date())
    return {
      date: dateText,
      weekday: ['一', '二', '三', '四', '五', '六', '日'][index],
      dayNumber: String(date.getDate()).padStart(2, '0'),
      isToday: dateText === todayText,
    }
  })
})

const weekRangeLabel = computed(() => {
  const first = weekDays.value[0]?.date || ''
  const last = weekDays.value[6]?.date || ''
  return `${first.slice(5)} ~ ${last.slice(5)}`
})

const selectedMember = computed(() => team.value[selectedMemberIndex.value] || null)

const openTeamScheduleGrid = () => {
  showTeamScheduleGrid.value = true
}

const onWeekCardClick = (event) => {
  if (event.target.closest('.day-pill') || event.target.closest('.icon-btn')) return
  openTeamScheduleGrid()
}

const onTeamSchedulePickDate = async (ymd) => {
  showTeamScheduleGrid.value = false
  selectedDate.value = ymd
  weekStartDate.value = getWeekStart(new Date(`${ymd}T12:00:00`))
  await loadData()
}
const selectedBehaviorSummary = computed(() => {
  const rows = selectedMemberRecords.value
  return {
    checkInCount: rows.filter((item) => item.status === 'CHECK_IN').length,
    checkOutCount: rows.filter((item) => item.status === 'CHECK_OUT').length,
    outingLikeCount: rows.filter((item) => item.status === 'OUTING' || item.status === 'BUSINESS_TRIP').length,
    diningCount: rows.filter((item) => item.status === 'DINING').length,
  }
})

const summary = computed(() => {
  const reportedMembers = team.value.filter((member) => Boolean(member.occurredAt)).length
  const reasonText = (member) => `${member.reason || ''} ${member.location || ''}`
  const includesKeywords = (text, keywords) => keywords.some((keyword) => text.includes(keyword))
  const clientVisitCount = team.value.filter((member) => includesKeywords(reasonText(member), ['拜访客户', '拜访'])).length
  const diningCount = team.value.filter((member) => includesKeywords(reasonText(member), ['用餐', '餐厅', '饭店'])).length
  const outingCount = team.value.filter((member) => member.status === 'OUTING' || member.status === 'BUSINESS_TRIP' || includesKeywords(reasonText(member), ['外出'])).length
  const accessClientCount = team.value.filter((member) => includesKeywords(reasonText(member), ['访问客户', '访问'])).length
  return {
    totalMembers: team.value.length,
    reportedMembers,
    clientVisitCount,
    diningCount,
    outingCount,
    accessClientCount,
  }
})

function getWeekStart(dateInput) {
  const currentDate = new Date(dateInput)
  const day = currentDate.getDay()
  const shift = day === 0 ? -6 : 1 - day
  currentDate.setDate(currentDate.getDate() + shift)
  return toLocalYMD(currentDate)
}

const loadData = async () => {
  try {
    const response = await getAdminTeam('day', selectedDate.value)
    team.value = (response.data.items || []).map(normalizeTeamMember)
    await loadMemberDeclareStates()
    if (selectedMemberIndex.value >= team.value.length) {
      selectedMemberIndex.value = 0
    }
    await loadSelectedMemberRecords()
  } catch (error) {
    team.value = []
    selectedMemberRecords.value = []
    memberDeclareStateByUserId.value = {}
    showFailToast(error?.response?.data?.error || '团队考勤加载失败，请检查网络或稍后重试')
  }
}

const loadMemberDeclareStates = async () => {
  if (!team.value.length) {
    memberDeclareStateByUserId.value = {}
    return
  }
  const statusEntries = await Promise.all(
    team.value.map(async (member) => {
      const userId = member.userId
      if (!userId) return [userId, { key: 'unknown', text: '状态异常' }]
      try {
        const [scheduleResponse, attendanceResponse] = await Promise.all([
          getScheduleDay(userId, selectedDate.value),
          getAttendanceByDate(userId, selectedDate.value),
        ])
        const schedulePayload = scheduleResponse?.data || null
        if (schedulePayload?.mode === 'leave') {
          return [userId, { key: 'leave', text: '休假免申报' }]
        }
        const records = Array.isArray(attendanceResponse?.data?.records)
          ? attendanceResponse.data.records.map(normalizeAttendanceRecord)
          : []
        const hasDeclared = records.some((record) => DECLARE_RECORD_STATUSES.includes(record.status))
        return [userId, hasDeclared ? { key: 'declared', text: '已申报' } : { key: 'missing', text: '未申报' }]
      } catch {
        return [userId, { key: 'unknown', text: '状态异常' }]
      }
    }),
  )
  memberDeclareStateByUserId.value = Object.fromEntries(statusEntries)
}

const memberDeclareState = (userId) => memberDeclareStateByUserId.value[userId] || { key: 'unknown', text: '状态异常' }

const loadSelectedMemberRecords = async () => {
  if (!selectedMember.value?.userId) {
    selectedMemberRecords.value = []
    return
  }
  try {
    const response = await getAttendanceByDate(selectedMember.value.userId, selectedDate.value)
    selectedMemberRecords.value = (response.data.records || []).map(normalizeAttendanceRecord)
  } catch {
    selectedMemberRecords.value = []
    showFailToast('员工当日记录加载失败')
  }
}

const selectDate = async (date) => {
  selectedDate.value = date
  await loadData()
}

const switchWeek = async (weekDelta) => {
  const nextStart = new Date(`${weekStartDate.value}T00:00:00`)
  nextStart.setDate(nextStart.getDate() + weekDelta * 7)
  weekStartDate.value = toLocalYMD(nextStart)
  selectedDate.value = weekStartDate.value
  await loadData()
}

const pickMonth = async (monthDate) => {
  showMonthPicker.value = false
  weekStartDate.value = getWeekStart(new Date(`${monthDate}T00:00:00`))
  selectedDate.value = weekStartDate.value
  await loadData()
}

const moveSelected = (delta) => {
  if (!team.value.length) return
  const nextIndex = (selectedMemberIndex.value + delta + team.value.length) % team.value.length
  selectedMemberIndex.value = nextIndex
}

const memberInitial = (name) => (name || '员').slice(0, 1)
const shortName = (name) => (name || '').slice(0, 2)
const clockText = (dateText) => {
  if (!dateText) return '未记时'
  const date = new Date(dateText)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
const goMemberHistory = async (member) => {
  if (!member?.userId) return
  await router.push(`/team-member/${member.userId}?name=${encodeURIComponent(member.userName || '')}`)
}

watch(selectedMemberIndex, () => {
  void loadSelectedMemberRecords()
})

watch(selectedDate, () => {
  const bounds = monthBounds()
  gridFrom.value = bounds.from
  gridTo.value = bounds.to
})

onMounted(async () => {
  weekStartDate.value = getWeekStart(new Date(`${selectedDate.value}T00:00:00`))
  await loadData()
})
</script>

<style scoped>
.team-page { min-height: 100vh; background: #f6f8ff; }
.team-topbar { height: 64px; display:flex; align-items:center; justify-content:space-between; background:#fff; border-bottom:1px solid #d8e0f5; padding: 0 12px; }
.team-topbar h1 { margin: 0; font-size: 20px; color:#102a5c; }
.back-btn, .month-btn, .icon-btn { border:1px solid #d8e0f5; background:#fff; border-radius:999px; color:#2951c7; height:36px; min-width:36px; display:flex; align-items:center; justify-content:center; padding: 0 12px; }
.team-content { padding: 14px 14px var(--app-nav-clearance); display:flex; flex-direction:column; gap:12px; }
.week-card, .summary-card, .member-card, .grid-io-card { border:1px solid #d8e0f5; border-radius:14px; background:#fff; padding:12px; box-shadow:0 8px 18px rgba(15, 40, 120, 0.06); }
.week-card--interactive { cursor: pointer; }
.week-card--interactive:active { background: #fafcff; }
.grid-io-card--compact { padding:8px 10px; }
.grid-io-head { display:flex; align-items:center; justify-content:space-between; gap:8px; margin-bottom:6px; }
.grid-io-title { margin:0; font-size:14px; font-weight:700; color:#102a5c; }
.grid-io-guide-btn {
  display:inline-flex; align-items:center; gap:2px;
  border:1px solid #dbe5ff; background:#f8fafc; color:#2c5ee8;
  border-radius:999px; padding:4px 10px; font-size:11px; font-weight:600;
}
.grid-io-guide-btn .material-symbols-outlined { font-size:16px; }
.grid-io-toolbar {
  display:grid;
  grid-template-columns: minmax(0,1fr) auto minmax(0,1fr) auto auto;
  gap:4px 6px;
  align-items:center;
}
.grid-io-sep { color:#94a3b8; font-size:12px; padding:0 2px; }
.grid-io-field { background:#f8fafc; border-radius:8px; overflow:hidden; }
.grid-io-field :deep(.van-cell) { padding:2px 8px; min-height:32px; }
.grid-io-field :deep(.van-field__label) { width:1.6em; margin-right:4px; font-size:12px; color:#64748b; }
.grid-io-field :deep(.van-field__control) { font-size:12px; }
.hidden-file { display:none; }
@media (max-width: 420px) {
  .grid-io-toolbar { grid-template-columns: 1fr 1fr; }
  .grid-io-sep { display:none; }
  .grid-io-toolbar .van-button { grid-column: span 1; }
}
.week-switch { display:flex; align-items:center; justify-content:space-between; margin-bottom:10px; }
.week-title { font-size: 14px; font-weight:700; color:#102a5c; }
.week-title-btn {
  border: 0;
  background: transparent;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 8px;
  cursor: pointer;
}
.week-title-btn:active { background: #edf2ff; }
.week-title-icon { font-size: 18px; color: #2c5ee8; }
.week-days { display:grid; grid-template-columns: repeat(7,minmax(0,1fr)); gap:6px; }
.day-pill { border:1px solid #d8e0f5; background:#fff; border-radius:10px; padding:6px 2px; display:flex; flex-direction:column; gap:2px; align-items:center; color:#475569; }
.day-pill.active { border-color:#2c5ee8; background:#edf2ff; color:#2c5ee8; font-weight:700; }
.day-pill.today strong { color:#f97316; }
.day-pill strong { font-size: 13px; }
.summary-card h2 { margin:0 0 10px; font-size:16px; color:#102a5c; }
.kpi-grid { display:grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap:8px; }
.kpi-item { border:1px solid #e4eafc; border-radius:10px; padding:8px; display:flex; justify-content:space-between; align-items:center; font-size:12px; color:#51607e; }
.kpi-item strong { color:#2c5ee8; font-size:15px; }
.member-head { display:flex; align-items:center; justify-content:space-between; margin-bottom:8px; }
.member-head h3 { margin:0; font-size:16px; color:#102a5c; }
.member-nav { display:flex; gap:6px; }
.avatar-row { display:flex; gap:8px; overflow:auto; padding-bottom:4px; }
.avatar-item { border:1px solid #d8e0f5; background:#fff; border-radius:12px; min-width:72px; min-height:86px; display:flex; flex-direction:column; align-items:center; justify-content:center; gap:3px; color:#64748b; padding:4px 6px; }
.avatar-item span { width:30px; height:30px; border-radius:999px; background:#e8eeff; color:#2c5ee8; display:flex; align-items:center; justify-content:center; font-weight:700; }
.avatar-item.active { border-color:#2c5ee8; background:#edf2ff; color:#2c5ee8; }
.avatar-declare {
  margin-top: 1px;
  font-style: normal;
  font-size: 10px;
  line-height: 1.2;
  font-weight: 700;
  border-radius: 999px;
  padding: 1px 6px;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.avatar-declare--declared { color:#166534; background:#dcfce7; }
.avatar-declare--missing { color:#991b1b; background:#fee2e2; }
.avatar-declare--leave { color:#1d4ed8; background:#dbeafe; }
.avatar-declare--unknown { color:#475569; background:#e2e8f0; }
.member-detail { border:1px solid #e4eafc; border-radius:10px; margin-top:10px; padding:10px; background:#fafcff; }
.detail-head { display:flex; justify-content:space-between; align-items:center; gap:8px; }
.member-detail h4 { margin:0; color:#102a5c; }
.detail-link { border:0; background:transparent; color:#2c5ee8; display:flex; align-items:center; gap:2px; font-size:12px; }
.behavior-grid { margin-top:10px; display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.behavior-item { border:1px solid #dbe5ff; border-radius:10px; padding:8px; display:flex; justify-content:space-between; font-size:12px; color:#51607e; }
.behavior-item strong { color:#2c5ee8; font-size:15px; }
.mini-timeline { margin-top:10px; border:1px solid #e4eafc; border-radius:10px; padding:8px; display:flex; flex-direction:column; gap:6px; max-height:156px; overflow:auto; }
.mini-title { font-size:12px; color:#64748b; }
.timeline-empty { color:#64748b; font-size:12px; }
.mini-item { display:grid; grid-template-columns:46px 56px minmax(0,1fr); gap:8px; align-items:center; font-size:12px; color:#475569; }
.mini-time { color:#64748b; }
.mini-status { color:#2c5ee8; font-weight:700; }
.mini-desc { white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.month-popup { padding: 16px; }
.month-popup h3 { margin:0 0 12px; color:#102a5c; }
.month-grid { display:grid; grid-template-columns: repeat(4,minmax(0,1fr)); gap:8px; }
.month-cell { border:1px solid #d8e0f5; background:#fff; border-radius:10px; padding:8px 0; color:#2c5ee8; }
</style>
