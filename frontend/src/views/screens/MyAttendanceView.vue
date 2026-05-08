<template>
  <div class="st-screen-page">
    <header class="st-topbar">
      <div class="st-topbar-left">
        <div class="st-avatar-wrap"><span class="material-symbols-outlined">person</span></div>
        <h1 class="st-brand">Intervoice</h1>
      </div>
      <button class="st-icon-btn" type="button"><span class="material-symbols-outlined">notifications</span></button>
    </header>

    <main class="st-content">
      <div>
        <h2 class="st-title">我的考勤</h2>
        <p class="st-sub">{{ periodLabel }} · 实时记录</p>
      </div>

      <div class="st-chip-row">
        <router-link class="st-chip" :class="{ 'st-chip-active': period === 'day' }" to="/my-attendance/day">日</router-link>
        <router-link class="st-chip" :class="{ 'st-chip-active': period === 'week' }" to="/my-attendance/week">周</router-link>
        <router-link class="st-chip" :class="{ 'st-chip-active': period === 'month' }" to="/my-attendance/month">月</router-link>
      </div>

      <section class="st-card">
        <h3 class="st-card-title">考勤统计</h3>
        <div class="st-kpi-grid">
          <div class="st-kpi-item"><span>总记录</span><strong>{{ summary.totalRecords }}</strong></div>
          <div class="st-kpi-item"><span>外勤</span><strong>{{ summary.outingCount }}</strong></div>
          <div class="st-kpi-item"><span>用餐</span><strong>{{ summary.diningCount }}</strong></div>
          <div class="st-kpi-item"><span>加班估算</span><strong>{{ summary.overtimeHours }}</strong></div>
        </div>
      </section>

      <section class="st-card">
        <h3 class="st-card-title">打卡记录</h3>
        <div class="st-table">
          <div class="st-row st-row-head">
            <div>状态</div>
            <div>地点</div>
            <div>时间</div>
          </div>
          <div v-for="record in records" :key="record.id" class="st-row">
            <div>{{ record.statusLabel }}</div>
            <div>{{ record.location }}</div>
            <div>{{ formatDateTime(record.occurredAt) }}</div>
          </div>
          <div v-if="records.length === 0" class="st-empty">暂无记录</div>
        </div>
      </section>
    </main>

    <AppBottomNav current="schedule" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import AppBottomNav from '../../components/AppBottomNav.vue'
import { getAttendanceList, getAttendanceSummary } from '../../api'
import { normalizeAttendanceRecord, PERIOD_OPTIONS, formatDateTime } from '../../data/models'

const props = defineProps({ period: { type: String, required: true } })
const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const records = ref([])
const summary = ref({ totalRecords: 0, outingCount: 0, diningCount: 0, overtimeHours: 0 })
const periodLabel = computed(() => PERIOD_OPTIONS.find((item) => item.key === props.period)?.label || props.period)

const loadData = async () => {
  const [listResponse, summaryResponse] = await Promise.all([
    getAttendanceList(user.id, props.period),
    getAttendanceSummary(user.id, props.period),
  ])
  records.value = (listResponse.data.records || []).map(normalizeAttendanceRecord)
  summary.value = summaryResponse.data || summary.value
}

onMounted(loadData)
</script>

<style scoped>
.st-screen-page { min-height: 100vh; background: #f8f9fa; color: #191c1d; }
.st-topbar { height: 64px; padding: 0 16px; border-bottom: 1px solid #c3c6d7; background: #fff; display: flex; justify-content: space-between; align-items: center; }
.st-topbar-left { display: flex; align-items: center; gap: 10px; }
.st-avatar-wrap { width: 32px; height: 32px; border: 1px solid #c3c6d7; border-radius: 999px; display:flex; align-items:center; justify-content:center; color:#434655; }
.st-brand { margin: 0; color: #004ac6; font-size: 30px; line-height: 38px; }
.st-icon-btn { width:40px; height:40px; border:0; background:transparent; color:#004ac6; border-radius:999px; }
.st-content { padding: 24px 16px 90px; display: flex; flex-direction: column; gap: 16px; }
.st-title { margin: 0; font-size: 30px; }
.st-sub { margin: 6px 0 0; color: #434655; }
.st-chip-row { display: flex; gap: 8px; }
.st-chip { padding: 6px 12px; border: 1px solid #c3c6d7; border-radius: 999px; color: #004ac6; text-decoration:none; font-size: 12px; }
.st-chip-active { background: #dbe1ff; border-color: #2563eb; }
.st-card { border: 1px solid #c3c6d7; border-radius: 8px; background: #fff; padding: 12px; }
.st-card-title { margin: 0 0 10px; font-size: 20px; }
.st-kpi-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 8px; }
.st-kpi-item { border: 1px solid #e1e3e4; border-radius: 8px; padding: 10px; display:flex; justify-content:space-between; font-size: 13px; }
.st-kpi-item strong { color: #004ac6; }
.st-table { border: 1px solid #e1e3e4; border-radius: 8px; overflow: hidden; }
.st-row { display:grid; grid-template-columns: 1fr 1.1fr 1.2fr; gap:8px; padding:10px; border-top:1px solid #e1e3e4; font-size:13px; }
.st-row:first-child { border-top:none; }
.st-row-head { background:#f8f9fa; font-size:12px; font-weight:600; color:#434655; }
.st-empty { padding: 12px; color:#64748b; font-size: 14px; }
</style>
