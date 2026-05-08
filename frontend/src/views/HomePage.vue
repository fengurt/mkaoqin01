<template>
  <div class="page-shell">
    <header class="topbar">
      <div>
        <p class="welcome">{{ greetingText }}</p>
        <h1>首页</h1>
      </div>
      <button class="icon-btn">
        <span class="material-symbols-outlined">notifications</span>
      </button>
    </header>

    <main class="content">
      <section class="hero-card">
        <div>
          <h2>今日状态概览</h2>
          <p>{{ todayTextDisplay }}</p>
        </div>
        <button class="primary-btn" @click="$router.push('/schedule')">
          <span class="material-symbols-outlined">add_task</span>
          快速去行程
        </button>
      </section>

      <section class="card">
        <h2 class="card-title">关键指标</h2>
        <div class="kpi-grid">
          <div class="kpi-item"><span>记录总数</span><strong>{{ records.length }}</strong></div>
          <div class="kpi-item"><span>最高频行为</span><strong>{{ topActionLabel }}</strong></div>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">快捷入口</h2>
        <div class="quick-grid">
          <button class="quick-btn" @click="$router.push('/schedule')"><span class="material-symbols-outlined">event_note</span><span>进入行程</span></button>
          <button class="quick-btn" @click="$router.push('/my-attendance/day')"><span class="material-symbols-outlined">calendar_month</span><span>我的考勤</span></button>
          <button class="quick-btn" @click="$router.push('/me')"><span class="material-symbols-outlined">person</span><span>我的主页</span></button>
          <button class="quick-btn" @click="$router.push('/employee/profile')"><span class="material-symbols-outlined">badge</span><span>个人资料</span></button>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">关键行为汇总（今日）</h2>
        <div v-if="sortedSummary.length" class="summary-list">
          <div v-for="item in sortedSummary" :key="item.key" class="summary-item">
            <span>{{ item.label }}</span>
            <strong>{{ item.count }}</strong>
          </div>
        </div>
        <div v-else class="empty">今天还没有行为记录，去行程新建第一条。</div>
      </section>

      <section class="card">
        <h2 class="card-title">最近动态</h2>
        <div v-if="records.length === 0" class="empty">暂无行为记录</div>
        <div v-else class="timeline-list">
          <div v-for="record in records" :key="record.id" class="timeline-item">
            <div class="timeline-time">{{ formatClock(record.occurredAt) }}</div>
            <div class="timeline-content">
              <div class="timeline-title">{{ record.statusLabel }}</div>
              <div class="timeline-desc">{{ record.location }} · {{ record.reason }}</div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <AppBottomNav current="home" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import AppBottomNav from '../components/AppBottomNav.vue'
import { getAttendanceByDate } from '../api'
import { formatClock, normalizeAttendanceRecord, STATUS_LABEL } from '../data/models'

const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const records = ref([])

const todayText = () => new Date().toISOString().slice(0, 10)
const todayTextDisplay = computed(() => todayText().replaceAll('-', '.'))
const greetingText = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return '早上好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

const loadRecords = async () => {
  const response = await getAttendanceByDate(user.id || 1, todayText())
  records.value = (response.data.records || []).map(normalizeAttendanceRecord)
}

const sortedSummary = computed(() => {
  const counter = {}
  records.value.forEach((row) => {
    counter[row.status] = (counter[row.status] || 0) + 1
  })
  return Object.entries(counter)
    .map(([status, count]) => ({ key: status, label: STATUS_LABEL[status] || status, count }))
    .sort((left, right) => right.count - left.count)
})
const topActionLabel = computed(() => sortedSummary.value[0]?.label || '无')

onMounted(loadRecords)

</script>

<style scoped>
.page-shell { min-height: 100vh; background:#f6f8ff; }
.topbar { height:72px; background:#fff; border-bottom:1px solid #d8e0f5; display:flex; align-items:center; justify-content:space-between; padding:0 16px; }
.welcome { margin:0; font-size:12px; color:#64748b; }
.topbar h1 { margin:0; font-size:22px; color:#0f172a; }
.icon-btn { width:38px; height:38px; border:1px solid #d8e0f5; border-radius:999px; background:#fff; color:#3156cb; }
.content { padding:16px 16px 84px; display:flex; flex-direction:column; gap:12px; }
.hero-card { background:linear-gradient(135deg,#2e58d6,#6b8dff); border-radius:14px; color:#fff; padding:14px; display:flex; align-items:center; justify-content:space-between; gap:10px; box-shadow:0 12px 26px rgba(46,88,214,0.3); }
.hero-card h2 { margin:0; font-size:16px; }
.hero-card p { margin:4px 0 0; font-size:12px; opacity:0.9; }
.primary-btn { border:0; background:#fff; color:#2e58d6; border-radius:10px; padding:8px 10px; display:flex; align-items:center; gap:4px; font-weight:700; font-size:12px; }
.card { background:#fff; border:1px solid #d8e0f5; border-radius:12px; padding:12px; box-shadow:0 8px 18px rgba(15,40,120,0.05); }
.card-title { margin:0 0 10px; font-size:16px; color:#0f172a; }
.kpi-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.kpi-item { border:1px solid #e4eafc; border-radius:10px; padding:10px; display:flex; flex-direction:column; gap:6px; }
.kpi-item span { color:#64748b; font-size:12px; }
.kpi-item strong { color:#3156cb; font-size:16px; }
.summary-list { display:flex; flex-direction:column; gap:8px; }
.summary-item { display:flex; justify-content:space-between; border:1px solid #e4eafc; border-radius:10px; padding:8px 10px; font-size:13px; }
.summary-item strong { color:#3156cb; }
.empty { color:#64748b; font-size:13px; }
.timeline-list { display:flex; flex-direction:column; gap:8px; }
.timeline-item { display:flex; gap:10px; border-bottom:1px dashed #e4eafc; padding:8px 0; }
.timeline-time { font-size:12px; color:#64748b; width:60px; }
.timeline-title { font-size:13px; font-weight:700; }
.timeline-desc { font-size:12px; color:#64748b; margin-top:2px; }
.quick-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.quick-btn { border:1px solid #dbe3ef; border-radius:10px; background:#fff; padding:10px; display:flex; flex-direction:column; align-items:flex-start; gap:6px; color:#0f172a; font-size:13px; transition:all .2s; }
.quick-btn:active { transform:translateY(1px) scale(0.99); background:#f1f5ff; }
.quick-btn .material-symbols-outlined { color:#3156cb; }
</style>
