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
      <section class="week-card">
        <div class="week-switch">
          <button class="icon-btn" type="button" @click="switchWeek(-1)">
            <span class="material-symbols-outlined">chevron_left</span>
          </button>
          <div class="week-title">当前周 {{ weekRangeLabel }}</div>
          <button class="icon-btn" type="button" @click="switchWeek(1)">
            <span class="material-symbols-outlined">chevron_right</span>
          </button>
        </div>
        <div class="week-days">
          <button
            v-for="day in weekDays"
            :key="day.date"
            class="day-pill"
            :class="{ active: day.date === selectedDate, today: day.isToday }"
            @click="selectDate(day.date)"
          >
            <span>{{ day.weekday }}</span>
            <strong>{{ day.dayNumber }}</strong>
          </button>
        </div>
      </section>

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

    <AppBottomNav current="me" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import AppBottomNav from '../../components/AppBottomNav.vue'
import { getAdminTeam, getAttendanceByDate } from '../../api'
import { normalizeAttendanceRecord, normalizeTeamMember } from '../../data/models'

const team = ref([])
const selectedDate = ref(new Date().toISOString().slice(0, 10))
const weekStartDate = ref(getWeekStart(new Date()))
const selectedMemberIndex = ref(0)
const showMonthPicker = ref(false)
const selectedMemberRecords = ref([])
const router = useRouter()

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
    const dateText = date.toISOString().slice(0, 10)
    const todayText = new Date().toISOString().slice(0, 10)
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
  return currentDate.toISOString().slice(0, 10)
}

const loadData = async () => {
  const response = await getAdminTeam('day', selectedDate.value)
  team.value = (response.data.items || []).map(normalizeTeamMember)
  if (selectedMemberIndex.value >= team.value.length) {
    selectedMemberIndex.value = 0
  }
  await loadSelectedMemberRecords()
}

const loadSelectedMemberRecords = async () => {
  if (!selectedMember.value?.userId) {
    selectedMemberRecords.value = []
    return
  }
  const response = await getAttendanceByDate(selectedMember.value.userId, selectedDate.value)
  selectedMemberRecords.value = (response.data.records || []).map(normalizeAttendanceRecord)
}

const selectDate = async (date) => {
  selectedDate.value = date
  await loadData()
}

const switchWeek = async (weekDelta) => {
  const nextStart = new Date(`${weekStartDate.value}T00:00:00`)
  nextStart.setDate(nextStart.getDate() + weekDelta * 7)
  weekStartDate.value = nextStart.toISOString().slice(0, 10)
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
.team-content { padding: 14px 14px 84px; display:flex; flex-direction:column; gap:12px; }
.week-card, .summary-card, .member-card { border:1px solid #d8e0f5; border-radius:14px; background:#fff; padding:12px; box-shadow:0 8px 18px rgba(15, 40, 120, 0.06); }
.week-switch { display:flex; align-items:center; justify-content:space-between; margin-bottom:10px; }
.week-title { font-size: 14px; font-weight:700; color:#102a5c; }
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
.avatar-item { border:1px solid #d8e0f5; background:#fff; border-radius:12px; min-width:56px; height:66px; display:flex; flex-direction:column; align-items:center; justify-content:center; gap:4px; color:#64748b; }
.avatar-item span { width:30px; height:30px; border-radius:999px; background:#e8eeff; color:#2c5ee8; display:flex; align-items:center; justify-content:center; font-weight:700; }
.avatar-item.active { border-color:#2c5ee8; background:#edf2ff; color:#2c5ee8; }
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
