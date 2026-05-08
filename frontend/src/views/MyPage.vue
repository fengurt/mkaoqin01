<template>
  <div class="page-shell">
    <header class="topbar">
      <h1>我的</h1>
      <span class="material-symbols-outlined">manage_accounts</span>
    </header>

    <main class="content">
      <section class="card profile">
        <van-image round width="64" height="64" src="https://fastly.jsdelivr.net/npm/@vant/assets/cat.jpeg" />
        <div>
          <h2>{{ user.displayName || '演示用户' }}</h2>
          <p>{{ user.role === 'admin' ? '管理员' : '员工' }}</p>
        </div>
      </section>

      <section class="card">
        <h3>关键入口</h3>
        <van-cell title="我的考勤（日）" is-link @click="$router.push('/my-attendance/day')" />
        <van-cell title="我的考勤（周）" is-link @click="$router.push('/my-attendance/week')" />
        <van-cell title="我的考勤（月）" is-link @click="$router.push('/my-attendance/month')" />
        <van-cell title="个人资料" is-link @click="$router.push('/employee/profile')" />
        <van-cell v-if="isAdmin" title="详细考勤报告" is-link @click="$router.push('/admin/report')" />
      </section>

      <section class="card">
        <h3>账号与安全</h3>
        <van-cell title="账号安全设置" is-link @click="$router.push('/me/security')" />
        <van-cell v-if="isAdmin" title="账号管理（管理员）" is-link @click="$router.push('/me/accounts')" />
      </section>

      <section v-if="isAdmin" class="team-spotlight" @click="$router.push('/team-attendance/day')">
        <div>
          <p class="team-tag">管理员入口</p>
          <h3>团队考勤</h3>
          <p class="team-sub">进入团队日历与两周汇总看板</p>
        </div>
        <span class="material-symbols-outlined">arrow_forward</span>
      </section>
    </main>

    <AppBottomNav current="me" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AppBottomNav from '../components/AppBottomNav.vue'

const user = JSON.parse(localStorage.getItem('user') || '{"id":1,"displayName":"演示用户","role":"employee"}')
const isAdmin = computed(() => user.role === 'admin')
</script>

<style scoped>
.page-shell { min-height:100vh; background:#f8f9fa; }
.topbar { height:64px; border-bottom:1px solid #c3c6d7; background:#fff; display:flex; align-items:center; justify-content:space-between; padding:0 16px; }
.topbar h1 { margin:0; font-size:22px; }
.content { padding:16px 16px 84px; display:flex; flex-direction:column; gap:10px; }
.card { background:#fff; border:1px solid #c3c6d7; border-radius:10px; }
.card h3 { margin:0; padding:12px 12px 0; font-size:15px; }
.profile { display:flex; align-items:center; gap:12px; padding:12px; }
.profile h2 { margin:0; font-size:18px; }
.profile p { margin:4px 0 0; color:#64748b; font-size:13px; }
.team-spotlight { border:1px solid #d7e1ff; background:linear-gradient(135deg,#2f5ed9,#7a97ff); color:#fff; border-radius:14px; padding:14px; display:flex; justify-content:space-between; align-items:center; box-shadow:0 14px 30px rgba(47,94,217,.28); }
.team-spotlight h3 { margin:0; font-size:20px; color:#fff; }
.team-tag { margin:0; font-size:12px; opacity:.92; }
.team-sub { margin:6px 0 0; font-size:12px; opacity:.88; }
.team-spotlight .material-symbols-outlined { font-size:22px; }
</style>
