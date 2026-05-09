<template>
  <div class="st-profile-page">
    <header class="st-topbar">
      <h1 class="st-brand">Intervoice</h1>
      <button class="st-icon-btn" type="button"><span class="material-symbols-outlined">settings</span></button>
    </header>

    <main class="st-content">
      <div class="st-profile-card">
        <van-image round width="72" height="72" src="https://fastly.jsdelivr.net/npm/@vant/assets/cat.jpeg" />
        <div>
          <h2>{{ user.displayName || '演示员工' }}</h2>
          <p>员工编号：{{ user.id || 1 }}</p>
        </div>
      </div>

      <div class="st-summary-card">
        <h3>月度摘要</h3>
        <div class="st-item"><span>打卡记录</span><strong>{{ summary.totalRecords }}</strong></div>
        <div class="st-item"><span>外勤次数</span><strong>{{ summary.outingCount }}</strong></div>
        <div class="st-item"><span>商务用餐</span><strong>{{ summary.diningCount }}</strong></div>
        <div class="st-item"><span>估算加班</span><strong>{{ summary.overtimeHours }}</strong></div>
      </div>
    </main>

    <AppBottomNav />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { showFailToast } from 'vant'
import AppBottomNav from '../../components/AppBottomNav.vue'
import { getAttendanceSummary } from '../../api'

const user = JSON.parse(localStorage.getItem('user') || '{"id":1,"displayName":"演示员工"}')
const summary = ref({ totalRecords: 0, outingCount: 0, diningCount: 0, overtimeHours: 0 })

onMounted(async () => {
  try {
    const response = await getAttendanceSummary(user.id || 1, 'month')
    summary.value = response.data || summary.value
  } catch (error) {
    showFailToast(error?.response?.data?.error || '摘要加载失败')
  }
})
</script>

<style scoped>
.st-profile-page { min-height: 100vh; background:#f8f9fa; }
.st-topbar { height:64px; padding:0 16px; background:#fff; border-bottom:1px solid #c3c6d7; display:flex; align-items:center; justify-content:space-between; }
.st-brand { margin:0; color:#004ac6; font-size:30px; }
.st-icon-btn { width:40px; height:40px; border:0; background:transparent; color:#004ac6; border-radius:999px; }
.st-content { padding:24px 16px var(--app-nav-clearance); display:flex; flex-direction:column; gap:16px; }
.st-profile-card { background:#fff; border:1px solid #c3c6d7; border-radius:8px; padding:16px; display:flex; align-items:center; gap:12px; }
.st-profile-card h2 { margin:0; color:#191c1d; }
.st-profile-card p { margin:4px 0 0; color:#434655; font-size:13px; }
.st-summary-card { background:#fff; border:1px solid #c3c6d7; border-radius:8px; padding:12px; }
.st-summary-card h3 { margin:0 0 8px; font-size:20px; }
.st-item { display:flex; justify-content:space-between; padding:10px 0; border-top:1px dashed #e1e3e4; }
.st-item:first-of-type { border-top:none; }
.st-item span { color:#434655; font-size:13px; }
.st-item strong { color:#004ac6; }
</style>
