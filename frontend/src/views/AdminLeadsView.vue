<template>
  <div class="page-shell">
    <header class="topbar">
      <button type="button" class="back-btn" @click="$router.push('/me')">返回</button>
      <h1>线索管理</h1>
      <span />
    </header>
    <main class="content">
      <van-loading v-if="loading" class="center-load" />
      <van-cell-group v-else inset>
        <van-cell v-for="item in items" :key="item.id" :title="item.clientName" :label="formatLead(item)" />
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
import { getAdminLeads } from '../api'

const { t } = useI18n()
const items = ref([])
const loading = ref(true)

const formatLead = (item) => {
  const seg = item.leadSegment ? t(`lead.segment.${item.leadSegment}`) : ''
  const venue = item.preferredVenue ? t(`lead.venue.${item.preferredVenue}`) : ''
  const origin = item.approxOriginRegion ? t(`lead.origin.${item.approxOriginRegion}`) : ''
  return [seg, venue, origin, item.status, item.updatedAt].filter(Boolean).join(' · ')
}

onMounted(async () => {
  try {
    const { data } = await getAdminLeads()
    items.value = data.items || []
  } catch {
    showFailToast('加载失败')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page-shell {
  min-height: 100vh;
  background: var(--brand-bg, #f3f0ea);
}
.topbar {
  display: grid;
  grid-template-columns: 64px 1fr 64px;
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
}
.content {
  padding: 12px 0 var(--app-nav-clearance);
}
.center-load {
  display: flex;
  justify-content: center;
  padding: 40px;
}
</style>
