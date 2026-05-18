<template>
  <div class="page-shell">
    <header class="topbar">
      <h1>我的</h1>
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
        <van-cell title="客户线索" is-link @click="$router.push('/leads')" />
        <van-cell title="我的徽章" is-link @click="$router.push('/me/badges')" />
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
        <van-cell v-if="isAdmin" title="行程快捷配置（管理员）" is-link @click="$router.push('/me/schedule-quick-config')" />
        <van-cell v-if="isAdmin" title="班次与假期（管理员）" is-link @click="$router.push('/me/schedule-types-config')" />
        <van-cell v-if="isAdmin" title="线索管理（管理员）" is-link @click="$router.push('/me/leads-admin')" />
      </section>

      <section class="card">
        <h3>显示语言</h3>
        <van-cell :title="localeTitle" is-link :value="localeLabel" @click="openLocalePicker" />
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

    <AppBottomNav />

    <van-action-sheet
      v-model:show="showLocaleSheet"
      :title="localeTitle"
      :actions="localeActions"
      @select="onLocaleSelect"
    />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppBottomNav from '../components/AppBottomNav.vue'
import { setLocale } from '../i18n'

const { t, locale } = useI18n()

const user = JSON.parse(localStorage.getItem('user') || '{"id":1,"displayName":"演示用户","role":"employee"}')
const isAdmin = computed(() => user.role === 'admin')

const showLocaleSheet = ref(false)
const localeTitle = computed(() => t('locale.label'))
const localeLabel = computed(() => (locale.value === 'en' ? t('locale.en') : t('locale.zh')))
const localeActions = computed(() => [{ name: t('locale.zh') }, { name: t('locale.en') }])

const openLocalePicker = () => {
  showLocaleSheet.value = true
}

const onLocaleSelect = (_action, index) => {
  setLocale(index === 1 ? 'en' : 'zh-CN')
  showLocaleSheet.value = false
}
</script>

<style scoped>
.page-shell { min-height:100vh; background: var(--brand-bg, #f1f5f9); }
.topbar { height:64px; border-bottom:1px solid var(--brand-border, #e2e8f0); background:#fff; display:flex; align-items:center; justify-content:space-between; padding:0 16px; }
.topbar h1 { margin:0; font-size:22px; color: var(--brand-title, #0f172a); }
.content { padding:16px 16px var(--app-nav-clearance); display:flex; flex-direction:column; gap:12px; }
.card {
  background: var(--brand-card, #fff);
  border: 1px solid var(--brand-border, #e2e8f0);
  border-radius: 12px;
  box-shadow: 0 8px 18px rgba(15, 40, 120, 0.05);
}
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
