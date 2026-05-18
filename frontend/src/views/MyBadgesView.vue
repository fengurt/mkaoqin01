<template>
  <div class="page-shell">
    <header class="topbar">
      <button type="button" class="back-btn" @click="$router.push('/me')">返回</button>
      <h1>{{ $t('myBadges.title') }}</h1>
      <span />
    </header>
    <main class="content">
      <section v-if="streak" class="card streak">
        <p>{{ $t('myBadges.streakCheckIn') }}: <strong>{{ streak.checkInCurrent }}</strong>（最佳 {{ streak.checkInBest }}）</p>
        <p>{{ $t('myBadges.streakFullDay') }}: <strong>{{ streak.fullDayCurrent }}</strong>（最佳 {{ streak.fullDayBest }}）</p>
        <p>{{ $t('myBadges.memberTier') }}: <strong>{{ streak.memberTier }}</strong></p>
      </section>
      <van-empty v-if="!badges.length" :description="$t('myBadges.emptyBadges')" />
      <van-cell-group v-else inset>
        <van-cell v-for="b in badges" :key="b.id" :title="badgeTitle(b)" :label="b.code" />
      </van-cell-group>
    </main>
    <AppBottomNav />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { showFailToast } from 'vant'
import AppBottomNav from '../components/AppBottomNav.vue'
import { getRewardsMe } from '../api'

const { locale } = useI18n()
const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const streak = ref(null)
const badges = ref([])

const badgeTitle = (b) => {
  const ti = b.titleI18n
  if (ti && typeof ti === 'object') {
    const loc = locale.value === 'en' ? 'en' : 'zh-CN'
    return ti[loc] || ti['zh-CN'] || b.code
  }
  return b.code
}

const load = async () => {
  try {
    const { data } = await getRewardsMe(user.id || 1)
    streak.value = data.streak || null
    badges.value = data.badges || []
  } catch {
    showFailToast('加载失败')
  }
}

onMounted(load)
</script>

<style scoped>
.page-shell {
  min-height: 100vh;
  background: var(--brand-bg, #f1f5f9);
}
.topbar {
  display: grid;
  grid-template-columns: 64px 1fr 64px;
  align-items: center;
  height: 52px;
  padding: 0 8px;
  background: #fff;
  border-bottom: 1px solid var(--brand-border, #e2e8f0);
}
.topbar h1 {
  margin: 0;
  text-align: center;
  font-size: 17px;
}
.back-btn {
  border: 0;
  background: transparent;
  color: var(--brand-primary-mid, #2563eb);
  font-size: 14px;
}
.content {
  padding: 12px;
  padding-bottom: var(--app-nav-clearance);
}
.card.streak {
  background: #fff;
  border-radius: 12px;
  padding: 14px;
  margin-bottom: 12px;
  border: 1px solid var(--brand-border, #e2e8f0);
  font-size: 14px;
  line-height: 1.6;
  color: #334155;
}
.card.streak p {
  margin: 6px 0;
}
</style>
