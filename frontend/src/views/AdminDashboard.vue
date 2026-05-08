<template>
  <div class="st-admin-page">
    <header class="st-topbar">
      <div class="st-topbar-left">
        <div class="st-avatar-wrap">
          <span class="material-symbols-outlined">person</span>
        </div>
        <h1 class="st-brand">Intervoice</h1>
      </div>
      <button class="st-icon-btn" type="button">
        <span class="material-symbols-outlined">notifications</span>
      </button>
    </header>

    <main class="st-admin-main">
      <div>
        <div class="st-title-wrap">
          <span class="material-symbols-outlined st-title-icon">analytics</span>
          <h2 class="st-title">团队状态</h2>
        </div>
        <p class="st-sub">全员实时考勤与在岗概览</p>
      </div>

      <div class="st-kpi-grid">
        <div class="st-kpi-card">
          <div class="st-kpi-head"><span class="material-symbols-outlined">groups</span><span>员工总数</span></div>
          <div class="st-kpi-value">{{ summary.totalEmployees }}</div>
        </div>
        <div class="st-kpi-card">
          <div class="st-kpi-head"><span class="material-symbols-outlined">check_circle</span><span>总记录</span></div>
          <div class="st-kpi-value">{{ summary.totalRecords }}</div>
        </div>
        <div class="st-kpi-card">
          <div class="st-kpi-head"><span class="material-symbols-outlined">commute</span><span>外勤次数</span></div>
          <div class="st-kpi-value">{{ summary.outingCount }}</div>
        </div>
        <div class="st-kpi-card st-kpi-alert">
          <div class="st-kpi-head"><span class="material-symbols-outlined">warning</span><span>离线人数</span></div>
          <div class="st-kpi-value">{{ summary.offlineCount }}</div>
        </div>
      </div>

      <div class="st-grid-wrap">
        <section class="st-roster">
          <div class="st-roster-head">
            <h3>实时花名册</h3>
            <button class="st-refresh" type="button" @click="loadAll">刷新</button>
          </div>
          <div class="st-table">
            <div class="st-row st-row-header">
              <div>员工</div>
              <div>状态</div>
              <div>地点</div>
            </div>
            <div v-for="item in items" :key="item.userId" class="st-row">
              <div class="st-emp-col">
                <div class="st-badge">{{ initials(item.userName) }}</div>
                <span>{{ item.userName }}</span>
              </div>
              <div>
                <span class="st-status-pill" :class="statusClass(item.status)">{{ statusMap[item.status] || item.status }}</span>
              </div>
              <div class="st-location">{{ item.location || '未上报地点' }}</div>
            </div>
          </div>
        </section>

        <aside class="st-side">
          <div class="st-side-card">
            <h3>详细考勤报告</h3>
            <div class="st-side-item"><span>迟到次数</span><strong>{{ report.lateCount }}</strong></div>
            <div class="st-side-item"><span>早退次数</span><strong>{{ report.earlyLeaveCount }}</strong></div>
            <div class="st-side-item"><span>风险预警</span><strong>{{ report.riskAlerts }}</strong></div>
            <div class="st-side-item"><span>总工时</span><strong>{{ report.totalHours }}</strong></div>
          </div>
        </aside>
      </div>

      <div class="card" style="margin-bottom: 90px">
        <div class="section-title">管理视图导航</div>
        <van-space wrap>
          <van-button size="small" @click="$router.push('/team-attendance/day')">团队考勤-日</van-button>
          <van-button size="small" @click="$router.push('/team-attendance/week')">团队考勤-周</van-button>
          <van-button size="small" @click="$router.push('/team-attendance/month')">团队考勤-月</van-button>
          <van-button size="small" @click="$router.push('/admin/report')">详细报告</van-button>
        </van-space>
      </div>
    </main>

    <nav class="st-bottom-nav">
      <a class="st-nav-item" href="#"><span class="material-symbols-outlined">dashboard</span><span>首页</span></a>
      <a class="st-nav-item" href="#"><span class="material-symbols-outlined">mic</span><span>语音</span></a>
      <a class="st-nav-item st-nav-active" href="#"><span class="material-symbols-outlined">analytics</span><span>团队</span></a>
      <a class="st-nav-item" href="#"><span class="material-symbols-outlined">person</span><span>我的</span></a>
    </nav>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { getAdminBoard, getAdminReport, getAdminSummary } from '../api'

const items = ref([])
const summary = ref({ totalEmployees: 0, totalRecords: 0, outingCount: 0, offlineCount: 0 })
const report = ref({ lateCount: 0, earlyLeaveCount: 0, riskAlerts: 0, totalHours: 0 })

const statusMap = {
  CHECK_IN: '签到',
  OFFICE: '在岗',
  OUTING: '外出',
  DINING: '用餐',
  BUSINESS_TRIP: '出差',
  CHECK_OUT: '签退',
  OFFLINE: '离线',
}

const initials = (name) => (name || '员工').slice(0, 1)

const statusClass = (status) => {
  if (status === 'OFFICE' || status === 'CHECK_IN') return 'st-status-ok'
  if (status === 'OUTING' || status === 'BUSINESS_TRIP') return 'st-status-warn'
  if (status === 'OFFLINE') return 'st-status-error'
  return ''
}

const loadAll = async () => {
  const [boardResponse, summaryResponse, reportResponse] = await Promise.all([
    getAdminBoard(),
    getAdminSummary('week'),
    getAdminReport(),
  ])
  items.value = boardResponse.data.items || []
  summary.value = summaryResponse.data || summary.value
  report.value = reportResponse.data || report.value
}

onMounted(async () => {
  await loadAll()
  setInterval(loadAll, 5000)
})
</script>

<style scoped>
.st-admin-page { min-height: 100vh; background: #f8f9fa; color: #191c1d; }
.st-topbar {
  height: 64px; padding: 0 16px; border-bottom: 1px solid #c3c6d7; background: #fff;
  display: flex; justify-content: space-between; align-items: center; position: sticky; top: 0; z-index: 20;
}
.st-topbar-left { display: flex; align-items: center; gap: 10px; }
.st-avatar-wrap {
  width: 32px; height: 32px; border-radius: 999px; border: 1px solid #c3c6d7;
  display: flex; align-items: center; justify-content: center; color: #434655;
}
.st-brand { margin: 0; color: #004ac6; font-size: 30px; line-height: 38px; }
.st-icon-btn { width: 40px; height: 40px; border-radius: 999px; border: none; background: transparent; color: #004ac6; }

.st-admin-main { max-width: 1280px; margin: 0 auto; padding: 24px 16px 90px; display: flex; flex-direction: column; gap: 24px; }
.st-title-wrap { display: flex; align-items: center; gap: 10px; }
.st-title-icon { color: #004ac6; font-size: 28px; }
.st-title { margin: 0; font-size: 36px; line-height: 44px; }
.st-sub { margin: 8px 0 0; color: #434655; }

.st-kpi-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.st-kpi-card { background: #fff; border: 1px solid #c3c6d7; border-radius: 8px; padding: 12px; }
.st-kpi-head { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #434655; margin-bottom: 10px; }
.st-kpi-value { font-size: 36px; line-height: 44px; color: #004ac6; font-weight: 700; }
.st-kpi-alert .st-kpi-head, .st-kpi-alert .st-kpi-value { color: #ba1a1a; }

.st-grid-wrap { display: grid; grid-template-columns: 1fr; gap: 16px; }
.st-roster, .st-side-card { background: #fff; border: 1px solid #c3c6d7; border-radius: 8px; }
.st-roster { padding: 12px; }
.st-roster-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.st-roster-head h3 { margin: 0; font-size: 24px; }
.st-refresh { border: 1px solid #c3c6d7; background: #fff; border-radius: 6px; padding: 6px 12px; color: #004ac6; font-weight: 600; }

.st-table { border: 1px solid #c3c6d7; border-radius: 8px; overflow: hidden; }
.st-row { display: grid; grid-template-columns: 1.2fr 0.9fr 1fr; gap: 8px; padding: 10px 12px; align-items: center; border-top: 1px solid #e1e3e4; }
.st-row:first-child { border-top: none; }
.st-row-header { background: #f8f9fa; font-size: 12px; font-weight: 600; color: #434655; }
.st-emp-col { display: flex; align-items: center; gap: 8px; }
.st-badge { width: 28px; height: 28px; border-radius: 999px; background: #d0e1fb; display: flex; align-items: center; justify-content: center; font-size: 12px; font-weight: 700; }
.st-status-pill { font-size: 12px; padding: 4px 8px; border-radius: 999px; background: #e1e3e4; }
.st-status-ok { background: #e6f4ea; color: #137333; }
.st-status-warn { background: #fef7e0; color: #b06000; }
.st-status-error { background: #fce8e6; color: #c5221f; }
.st-location { font-size: 13px; color: #434655; }

.st-side-card { padding: 12px; }
.st-side-card h3 { margin: 0 0 10px; font-size: 20px; }
.st-side-item { display: flex; justify-content: space-between; border-top: 1px dashed #c3c6d7; padding: 10px 0; font-size: 14px; }
.st-side-item:first-of-type { border-top: none; }

.st-bottom-nav {
  position: fixed; left: 0; right: 0; bottom: 0; height: 64px; background: #fff; border-top: 1px solid #c3c6d7;
  display: flex; align-items: center; justify-content: space-around; z-index: 50;
}
.st-nav-item { display: flex; flex-direction: column; align-items: center; gap: 2px; color: #434655; text-decoration: none; font-size: 12px; }
.st-nav-active { color: #004ac6; font-weight: 700; }

@media (min-width: 900px) {
  .st-grid-wrap { grid-template-columns: 2fr 1fr; }
  .st-kpi-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
}
</style>
