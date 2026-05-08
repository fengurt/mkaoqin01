<template>
  <div class="st-report-page">
    <header class="st-topbar">
      <h1 class="st-brand">Intervoice</h1>
      <button class="st-icon-btn" type="button"><span class="material-symbols-outlined">notifications</span></button>
    </header>

    <main class="st-content">
      <h2 class="st-title">详细考勤报告（管理员）</h2>
      <p class="st-sub">月度风险与工时汇总</p>

      <div class="st-card-grid">
        <div class="st-card"><span>迟到次数</span><strong>{{ report.lateCount }}</strong></div>
        <div class="st-card"><span>早退次数</span><strong>{{ report.earlyLeaveCount }}</strong></div>
        <div class="st-card"><span>风险预警</span><strong>{{ report.riskAlerts }}</strong></div>
        <div class="st-card"><span>总工时</span><strong>{{ report.totalHours }}</strong></div>
      </div>

      <div class="st-table-card">
        <div class="st-table-row"><span>数据来源</span><strong>admin/report</strong></div>
        <div class="st-table-row"><span>统计周期</span><strong>本月</strong></div>
        <div class="st-table-row"><span>最后更新时间</span><strong>{{ new Date().toLocaleString() }}</strong></div>
      </div>

      <div class="st-table-card">
        <div class="st-table-row"><span>近7日风险趋势</span><strong>{{ report.simulatedTrend?.alerts7d?.join(' / ') || '-' }}</strong></div>
        <div class="st-table-row"><span>近7日打卡趋势</span><strong>{{ report.simulatedTrend?.checkins7d?.join(' / ') || '-' }}</strong></div>
      </div>
    </main>

    <AppBottomNav current="me" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import AppBottomNav from '../../components/AppBottomNav.vue'
import { getAdminReport } from '../../api'

const report = ref({ lateCount: 0, earlyLeaveCount: 0, riskAlerts: 0, totalHours: 0 })

onMounted(async () => {
  const response = await getAdminReport()
  report.value = response.data || report.value
})
</script>

<style scoped>
.st-report-page { min-height: 100vh; background: #f8f9fa; }
.st-topbar { height: 64px; padding: 0 16px; background:#fff; border-bottom: 1px solid #c3c6d7; display:flex; align-items:center; justify-content:space-between; }
.st-brand { margin: 0; color:#004ac6; font-size:30px; }
.st-icon-btn { width:40px; height:40px; border:0; background:transparent; color:#004ac6; border-radius:999px; }
.st-content { padding: 24px 16px; display:flex; flex-direction:column; gap:16px; }
.st-title { margin:0; font-size:30px; color:#191c1d; }
.st-sub { margin:0; color:#434655; }
.st-card-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.st-card { background:#fff; border:1px solid #c3c6d7; border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:8px; }
.st-card span { font-size:12px; color:#434655; }
.st-card strong { font-size:30px; color:#004ac6; }
.st-table-card { background:#fff; border:1px solid #c3c6d7; border-radius:8px; }
.st-table-row { display:flex; justify-content:space-between; padding:12px; border-top:1px solid #e1e3e4; font-size:13px; }
.st-table-row:first-child { border-top:none; }
</style>
