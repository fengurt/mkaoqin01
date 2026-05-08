<template>
  <div class="history-page">
    <header class="topbar">
      <button class="icon-btn" type="button" @click="$router.push('/team-attendance/day')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>{{ memberName || '员工' }}历史汇总</h1>
      <span class="material-symbols-outlined">insights</span>
    </header>

    <main class="content">
      <section class="card">
        <div class="range-row">
          <strong>统计区间</strong>
          <span>{{ periodRangeLabel }}</span>
        </div>
        <div class="period-row">
          <button
            v-for="item in periodOptions"
            :key="item.value"
            class="period-btn"
            :class="{ active: selectedPeriod === item.value }"
            @click="changePeriod(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
      </section>

      <section class="card">
        <h3>行为统计</h3>
        <div class="kpi-grid">
          <div class="kpi-item"><span>记录总数</span><strong>{{ summary.totalRecords || 0 }}</strong></div>
          <div class="kpi-item"><span>外出次数</span><strong>{{ summary.outingCount || 0 }}</strong></div>
          <div class="kpi-item"><span>在岗天数</span><strong>{{ summary.officeDays || 0 }}</strong></div>
          <div class="kpi-item"><span>总工时</span><strong>{{ summary.totalHours || 0 }}</strong></div>
        </div>
      </section>

      <section class="card">
        <h3>行为时间线</h3>
        <div v-if="records.length === 0" class="empty">该周期暂无行为记录</div>
        <div v-else class="timeline">
          <div v-for="record in records" :key="record.id" class="item">
            <span class="time">{{ formatDateTime(record.occurredAt) }}</span>
            <span class="status">{{ record.statusLabel }}</span>
            <span class="desc">{{ record.reason }} · {{ record.location }}</span>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getAttendanceList, getAttendanceSummary } from '../../api'
import { formatDateTime, normalizeAttendanceRecord } from '../../data/models'

const route = useRoute()
const memberId = computed(() => Number(route.params.userId || 0))
const memberName = computed(() => String(route.query.name || ''))
const selectedPeriod = ref('week')
const records = ref([])
const summary = ref({ totalRecords: 0, outingCount: 0, officeDays: 0, totalHours: 0 })

const periodOptions = [
  { value: 'day', label: '日' },
  { value: 'week', label: '周' },
  { value: 'month', label: '月' },
]

const periodRangeLabel = computed(() => {
  const today = new Date()
  const end = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  let start = new Date(end)
  if (selectedPeriod.value === 'day') {
    start = new Date(end)
  } else if (selectedPeriod.value === 'week') {
    start.setDate(end.getDate() - 6)
  } else {
    start = new Date(end.getFullYear(), end.getMonth(), 1)
  }
  const fmt = (date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  return `${fmt(start)} 至 ${fmt(end)}`
})

const loadData = async () => {
  if (!memberId.value) return
  const [recordsResponse, summaryResponse] = await Promise.all([
    getAttendanceList(memberId.value, selectedPeriod.value),
    getAttendanceSummary(memberId.value, selectedPeriod.value),
  ])
  records.value = (recordsResponse.data.items || []).map(normalizeAttendanceRecord)
  summary.value = summaryResponse.data || summary.value
}

const changePeriod = async (period) => {
  selectedPeriod.value = period
  await loadData()
}

onMounted(loadData)
</script>

<style scoped>
.history-page { min-height: 100vh; background: #f6f8ff; }
.topbar { position: sticky; top: 0; z-index: 20; height: 64px; border-bottom: 1px solid #d8e0f5; background: #fff; display: flex; align-items: center; justify-content: space-between; padding: 0 12px; }
.topbar h1 { margin: 0; font-size: 17px; color: #102a5c; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.icon-btn { width: 36px; height: 36px; border: 1px solid #d8e0f5; border-radius: 999px; background: #fff; color: #3156cb; display: flex; align-items: center; justify-content: center; }
.content { padding: 14px; display: flex; flex-direction: column; gap: 12px; }
.card { border: 1px solid #d8e0f5; border-radius: 12px; background: #fff; box-shadow: 0 8px 18px rgba(15, 40, 120, 0.05); padding: 12px; }
.card h3 { margin: 0 0 10px; font-size: 15px; color: #102a5c; }
.range-row { display:flex; align-items:center; justify-content:space-between; padding-bottom:10px; margin-bottom:10px; border-bottom:1px solid #edf1fd; font-size:12px; color:#51607e; }
.range-row strong { color:#102a5c; }
.period-row { display: flex; gap: 8px; }
.period-btn { border: 1px solid #d8e0f5; background: #fff; color: #64748b; border-radius: 999px; padding: 6px 14px; font-size: 12px; }
.period-btn.active { border-color: #2c5ee8; background: #edf2ff; color: #2c5ee8; font-weight: 700; }
.kpi-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.kpi-item { border: 1px solid #e4eafc; border-radius: 10px; padding: 8px; display: flex; justify-content: space-between; font-size: 12px; color: #51607e; }
.kpi-item strong { color: #2c5ee8; font-size: 15px; }
.empty { color: #64748b; font-size: 13px; }
.timeline { display: flex; flex-direction: column; gap: 8px; max-height: 420px; overflow: auto; }
.item { border: 1px solid #e4eafc; border-radius: 10px; padding: 8px; display: grid; grid-template-columns: 1fr auto; gap: 4px 8px; }
.time { grid-column: 1 / 3; color: #64748b; font-size: 12px; }
.status { color: #2c5ee8; font-size: 13px; font-weight: 700; }
.desc { color: #475569; font-size: 12px; text-align: right; }
</style>
