<template>
  <div class="page-shell">
    <header class="topbar">
      <button type="button" class="back-btn" @click="$router.push('/home')">{{ $t('lead.backHome') }}</button>
      <h1>{{ $t('lead.title') }}</h1>
      <span />
    </header>
    <main class="content">
      <van-loading v-if="loading" class="center-load" />
      <van-empty v-else-if="!items.length" :description="$t('home.leadTileEmpty')" />
      <div v-else class="lead-deck">
        <article
          v-for="item in items"
          :key="item.id"
          class="lead-card"
          role="button"
          tabindex="0"
          @click="openDetail(item.id)"
          @keydown.enter.prevent="openDetail(item.id)"
        >
          <div class="lead-card-body">
            <header class="lead-card-head">
              <h2 class="lead-card-title">{{ item.clientName }}</h2>
              <div class="lead-card-score" :title="$t('lead.radar.composite')">
                <span class="lead-card-score-val">{{ potentialFor(item) }}</span>
                <span class="lead-card-score-lbl">{{ $t('lead.radar.compositeShort') }}</span>
              </div>
            </header>
            <div class="lead-chips">
              <span v-if="item.leadSegment" class="lc">{{ t(`lead.segment.${item.leadSegment}`) }}</span>
              <span v-if="item.preferredVenue" class="lc lc--gold">{{ t(`lead.venue.${item.preferredVenue}`) }}</span>
              <span v-if="item.priority" class="lc lc--pri">{{ item.priority }}</span>
            </div>
            <p class="lead-meta">{{ formatLeadMeta(item) }}</p>
          </div>
          <div class="lead-card-viz" aria-hidden="true">
            <LeadValueRadarChart
              :scores="scoresFor(item)"
              :labels="[]"
              compact
              :diameter="80"
            />
          </div>
          <span class="material-symbols-outlined lead-card-chev">chevron_right</span>
        </article>
      </div>
    </main>
    <AppBottomNav />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showFailToast } from 'vant'
import AppBottomNav from '../components/AppBottomNav.vue'
import LeadValueRadarChart from '../components/LeadValueRadarChart.vue'
import { getLeadFeed } from '../api'
import { computeLeadValueRadar, leadRadarScoresArray } from '../lib/leadValuePotential'

const { t } = useI18n()
const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const items = ref([])
const loading = ref(true)

const formatLeadMeta = (item) => {
  const origin = item.approxOriginRegion ? t(`lead.origin.${item.approxOriginRegion}`) : ''
  const created = item.createdAt || ''
  const tail = [origin, `${t('home.leadCreated')}: ${created.slice(0, 16)}`, item.status].filter(Boolean)
  return tail.join(' · ')
}

const potentialFor = (item) => computeLeadValueRadar(item).composite

const scoresFor = (item) => leadRadarScoresArray(item)

const load = async () => {
  loading.value = true
  try {
    const { data } = await getLeadFeed(user.id || 1)
    items.value = data.items || []
  } catch {
    showFailToast('加载失败')
    items.value = []
  } finally {
    loading.value = false
  }
}

const openDetail = (id) => {
  router.push({ path: '/leads/detail', query: { leadId: String(id) } })
}

onMounted(load)
</script>

<style scoped>
.page-shell {
  min-height: 100vh;
  background: var(--brand-bg, #f3f0ea);
}
.topbar {
  display: grid;
  grid-template-columns: 88px 1fr 88px;
  align-items: center;
  height: 52px;
  padding: 0 8px;
  background: var(--brand-card, #fdfbf7);
  border-bottom: 2px solid var(--accent-gold, #b8954f);
}
.topbar h1 {
  margin: 0;
  text-align: center;
  font-size: 17px;
}
.back-btn {
  border: 0;
  background: transparent;
  color: var(--brand-primary-mid, #6b5a32);
  font-size: 14px;
  justify-self: start;
}
.content {
  padding: 12px 12px var(--app-nav-clearance);
}
.lead-deck {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.lead-card {
  display: grid;
  grid-template-columns: 1fr 88px 28px;
  align-items: center;
  gap: 4px 8px;
  padding: 12px 10px 12px 14px;
  background: var(--brand-card, #fdfbf7);
  border-radius: 16px;
  border: 1px solid var(--brand-border, #d4cec2);
  box-shadow: 0 8px 22px rgba(20, 24, 33, 0.06);
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}
.lead-card:active {
  filter: brightness(0.98);
}
.lead-card-body {
  min-width: 0;
}
.lead-card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}
.lead-card-title {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
  color: var(--brand-title, #141821);
  font-family: 'Noto Serif SC', 'Songti SC', serif;
  line-height: 1.25;
}
.lead-card-score {
  flex-shrink: 0;
  text-align: right;
  padding: 4px 8px;
  border-radius: 10px;
  background: var(--accent-gold-soft, rgba(184, 149, 79, 0.14));
  border: 1px solid rgba(184, 149, 79, 0.35);
}
.lead-card-score-val {
  display: block;
  font-size: 17px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--brand-title, #141821);
  line-height: 1.1;
}
.lead-card-score-lbl {
  font-size: 9px;
  font-weight: 700;
  color: var(--brand-subtext, #6b6560);
  letter-spacing: 0.03em;
}
.lead-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.lead-chips .lc {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--brand-primary-soft, #f0e8d8);
  color: var(--brand-primary-mid, #6b5a32);
  border: 1px solid var(--brand-border, #d4cec2);
}
.lead-chips .lc--gold {
  text-transform: none;
  letter-spacing: 0;
  background: var(--accent-gold-soft, rgba(184, 149, 79, 0.14));
  border-color: rgba(184, 149, 79, 0.4);
}
.lead-chips .lc--pri {
  text-transform: none;
  background: rgba(26, 34, 48, 0.06);
  border-color: rgba(26, 34, 48, 0.12);
}
.lead-meta {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--brand-subtext, #6b6560);
}
.lead-card-viz {
  display: flex;
  justify-content: center;
  align-items: center;
}
.lead-card-chev {
  font-size: 22px;
  color: rgba(107, 90, 50, 0.45);
  justify-self: center;
}
.center-load {
  display: flex;
  justify-content: center;
  padding: 40px;
}
</style>
