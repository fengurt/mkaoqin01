<template>
  <div class="page-shell">
    <header class="topbar">
      <button type="button" class="back-btn" @click="$router.push('/leads')">{{ $t('lead.title') }}</button>
      <h1>{{ lead?.clientName || '…' }}</h1>
      <span />
    </header>
    <main v-if="lead" class="content">
      <section class="card card--hero">
        <div class="hero-chips">
          <span v-if="lead.leadSegment" class="chip chip--segment">{{ $t(`lead.segment.${lead.leadSegment}`) }}</span>
          <span v-if="lead.approxOriginRegion" class="chip">{{ $t(`lead.origin.${lead.approxOriginRegion}`) }}</span>
          <span v-if="lead.preferredVenue" class="chip chip--gold">{{ $t(`lead.venue.${lead.preferredVenue}`) }}</span>
        </div>
        <p class="status-pill">{{ lead.status }}</p>
        <p class="meta">{{ $t('home.leadCreated') }}: {{ (lead.createdAt || '').slice(0, 19) }}</p>
        <p v-if="lead.pickedUpAt" class="meta">{{ $t('lead.claimedAt') }}: {{ lead.pickedUpAt.slice(0, 19) }}</p>
        <p class="body">{{ lead.notes }}</p>
        <van-button
          v-if="lead.status === 'NEW' || lead.status === 'ASSIGNED'"
          type="primary"
          block
          round
          :loading="pickBusy"
          @click="onPickUp"
        >
          {{ $t('lead.pickUp') }}
        </van-button>
      </section>

      <section class="card card--radar">
        <div class="radar-head">
          <div>
            <h3>{{ $t('lead.radar.title') }}</h3>
            <p class="radar-sub">{{ $t('lead.radar.subtitle') }}</p>
          </div>
          <div class="composite-ring" role="status">
            <span class="composite-val">{{ valueRadar.composite }}</span>
            <span class="composite-lbl">{{ $t('lead.radar.compositeShort') }}</span>
          </div>
        </div>
        <LeadValueRadarChart
          :scores="radarScores"
          :labels="radarLabels"
          :diameter="252"
          :aria-label="$t('lead.radar.title')"
        />
        <ul class="radar-bars">
          <li v-for="key in LEAD_RADAR_AXIS_KEYS" :key="key">
            <span class="rb-name">{{ $t(`lead.radar.axis.${key}`) }}</span>
            <div class="rb-track" role="presentation">
              <i :style="{ width: (valueRadar.byAxis[key] ?? 0) + '%' }" />
            </div>
            <span class="rb-num">{{ valueRadar.byAxis[key] ?? 0 }}</span>
          </li>
        </ul>
      </section>

      <section class="card">
        <h3>{{ $t('lead.profileTitle') }}</h3>
        <dl class="kv">
          <template v-if="lead.languagePref">
            <dt>{{ $t('lead.field.languagePref') }}</dt>
            <dd>{{ lead.languagePref }}</dd>
          </template>
          <template v-if="lead.estimatedPartySize != null && lead.estimatedPartySize !== ''">
            <dt>{{ $t('lead.field.partySize') }}</dt>
            <dd>{{ lead.estimatedPartySize }}</dd>
          </template>
          <template v-if="lead.eventOccasion">
            <dt>{{ $t('lead.field.eventOccasion') }}</dt>
            <dd>{{ $t(`lead.occasion.${lead.eventOccasion}`) }}</dd>
          </template>
          <template v-if="lead.channelTouchpoint">
            <dt>{{ $t('lead.field.channelTouchpoint') }}</dt>
            <dd>{{ $t(`lead.channel.${lead.channelTouchpoint}`) }}</dd>
          </template>
          <template v-if="lead.intent">
            <dt>{{ $t('lead.field.intent') }}</dt>
            <dd>{{ lead.intent }}</dd>
          </template>
          <template v-if="lead.priority">
            <dt>{{ $t('lead.field.priority') }}</dt>
            <dd>{{ lead.priority }}</dd>
          </template>
        </dl>
      </section>

      <section v-if="showRefBlock" class="card card--ref">
        <h3>{{ $t('lead.refTitle') }}</h3>
        <dl class="kv">
          <template v-if="lead.refLastVisitAt">
            <dt>{{ $t('lead.field.refLastVisit') }}</dt>
            <dd>{{ lead.refLastVisitAt }}</dd>
          </template>
          <template v-if="lead.refLastProperty">
            <dt>{{ $t('lead.field.refLastProperty') }}</dt>
            <dd>{{ $te(`lead.venue.${lead.refLastProperty}`) ? $t(`lead.venue.${lead.refLastProperty}`) : lead.refLastProperty }}</dd>
          </template>
          <template v-if="lead.refLtvTier">
            <dt>{{ $t('lead.field.refLtvTier') }}</dt>
            <dd>{{ lead.refLtvTier }}</dd>
          </template>
          <template v-if="lead.refHostName">
            <dt>{{ $t('lead.field.refHostName') }}</dt>
            <dd>{{ lead.refHostName }}</dd>
          </template>
          <template v-if="lead.refMemberIdMasked">
            <dt>{{ $t('lead.field.refMemberIdMasked') }}</dt>
            <dd>{{ lead.refMemberIdMasked }}</dd>
          </template>
          <template v-if="lead.refNotes">
            <dt>{{ $t('lead.field.refNotes') }}</dt>
            <dd class="dd-multiline">{{ lead.refNotes }}</dd>
          </template>
        </dl>
      </section>

      <section v-if="canFollowUp" class="card">
        <h3>{{ $t('lead.followUp') }}</h3>
        <van-field v-model="note" type="textarea" rows="3" :placeholder="$t('lead.notePlaceholder')" />
        <van-button type="success" block round :loading="fuBusy" class="fu-btn" @click="onFollowUp">
          {{ $t('lead.followUp') }}
        </van-button>
      </section>
      <section v-else class="card card--hint">
        <p class="hint-text">{{ $t('lead.followUpLocked') }}</p>
      </section>
      <section class="card">
        <h3>{{ $t('lead.timeline') }}</h3>
        <ul class="timeline">
          <li v-for="ev in events" :key="ev.id">
            <time>{{ ev.occurredAt }}</time>
            <span class="ev-type">{{ ev.type }}</span>
            <pre class="ev-payload">{{ ev.payload }}</pre>
          </li>
        </ul>
      </section>
    </main>
    <van-loading v-else-if="loading" class="center-load" />
    <AppBottomNav />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showFailToast, showSuccessToast } from 'vant'
import AppBottomNav from '../components/AppBottomNav.vue'
import LeadValueRadarChart from '../components/LeadValueRadarChart.vue'
import { followUpLead, getLeadDetail, pickUpLead } from '../api'
import { LEAD_RADAR_AXIS_KEYS, computeLeadValueRadar, leadRadarScoresArray } from '../lib/leadValuePotential'

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const userIdNum = Number(user.id) || 1
const lead = ref(null)
const events = ref([])
const loading = ref(true)
const note = ref('')
const pickBusy = ref(false)
const fuBusy = ref(false)

const leadId = computed(() => route.query.leadId || '')

const canFollowUp = computed(() => {
  const l = lead.value
  if (!l) return false
  const assigned = Number(l.assignedUserId) || 0
  const picked = Number(l.pickedUpBy) || 0
  return picked === userIdNum || assigned === userIdNum
})

const showRefBlock = computed(() => {
  const l = lead.value
  if (!l) return false
  if (l.leadSegment === 'OLD_REACTIVATION') return true
  return Boolean(
    l.refLastVisitAt ||
      l.refLastProperty ||
      l.refLtvTier ||
      l.refHostName ||
      l.refMemberIdMasked ||
      l.refNotes,
  )
})

const valueRadar = computed(() => computeLeadValueRadar(lead.value || {}))

const radarScores = computed(() => leadRadarScoresArray(lead.value || {}))

const radarLabels = computed(() => LEAD_RADAR_AXIS_KEYS.map((k) => t(`lead.radar.axis.${k}`)))

const load = async () => {
  if (!leadId.value) {
    router.replace('/leads')
    return
  }
  loading.value = true
  try {
    const { data } = await getLeadDetail(userIdNum, leadId.value)
    lead.value = data.lead || null
    events.value = data.events || []
  } catch {
    showFailToast('加载失败')
    lead.value = null
  } finally {
    loading.value = false
  }
}

const onPickUp = async () => {
  pickBusy.value = true
  try {
    await pickUpLead({ leadId: Number(leadId.value), userId: userIdNum })
    showSuccessToast('已认领')
    await load()
  } catch {
    showFailToast('认领失败')
  } finally {
    pickBusy.value = false
  }
}

const onFollowUp = async () => {
  if (note.value.trim().length < 2) {
    showFailToast('请填写跟进内容')
    return
  }
  fuBusy.value = true
  try {
    await followUpLead({
      leadId: Number(leadId.value),
      userId: userIdNum,
      note: note.value.trim(),
      statusTo: 'IN_FOLLOW_UP',
    })
    showSuccessToast('已记录')
    note.value = ''
    await load()
  } catch {
    showFailToast('提交失败')
  } finally {
    fuBusy.value = false
  }
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
  grid-template-columns: 72px 1fr 72px;
  align-items: center;
  min-height: 52px;
  padding: 0 8px;
  background: var(--brand-card, #fdfbf7);
  border-bottom: 2px solid var(--accent-gold, #b8954f);
  box-shadow: 0 6px 18px rgba(20, 24, 33, 0.06);
}
.topbar h1 {
  margin: 0;
  text-align: center;
  font-size: 16px;
  font-family: 'Noto Serif SC', 'Songti SC', serif;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--brand-title, #141821);
}
.back-btn {
  border: 0;
  background: transparent;
  color: var(--brand-primary-mid, #6b5a32);
  font-size: 13px;
  justify-self: start;
}
.content {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-bottom: var(--app-nav-clearance);
}
.card {
  background: var(--brand-card, #fdfbf7);
  border-radius: 16px;
  padding: 14px;
  border: 1px solid var(--brand-border, #d4cec2);
  box-shadow: 0 6px 20px rgba(20, 24, 33, 0.05);
}
.card--hero {
  border-color: rgba(184, 149, 79, 0.35);
}
.card--ref {
  background: linear-gradient(165deg, #fdfbf7 0%, #f5efe4 100%);
}
.card h3 {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--brand-subtext, #6b6560);
}
.hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 10px;
}
.chip {
  font-size: 11px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--brand-primary-soft, #f0e8d8);
  color: var(--brand-primary-mid, #6b5a32);
  border: 1px solid var(--brand-border, #d4cec2);
}
.chip--segment {
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.chip--gold {
  background: var(--accent-gold-soft, rgba(184, 149, 79, 0.14));
  border-color: rgba(184, 149, 79, 0.45);
}
.status-pill {
  display: inline-block;
  margin: 0 0 8px;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--accent-gold-soft, rgba(184, 149, 79, 0.14));
  color: var(--brand-primary-mid, #6b5a32);
  font-size: 12px;
  font-weight: 700;
}
.meta {
  margin: 4px 0;
  font-size: 12px;
  color: var(--brand-subtext, #6b6560);
}
.body {
  margin: 10px 0 14px;
  font-size: 14px;
  line-height: 1.55;
  color: var(--brand-title, #141821);
}
.kv {
  display: grid;
  grid-template-columns: 108px 1fr;
  gap: 8px 12px;
  margin: 0;
  font-size: 13px;
}
.kv dt {
  margin: 0;
  color: var(--brand-subtext, #6b6560);
  font-weight: 600;
}
.kv dd {
  margin: 0;
  color: var(--brand-title, #141821);
  font-weight: 600;
}
.dd-multiline {
  white-space: pre-wrap;
  font-weight: 500;
  line-height: 1.45;
}
.fu-btn {
  margin-top: 10px;
}
.card--hint {
  background: rgba(240, 232, 216, 0.45);
}
.hint-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: var(--brand-subtext, #6b6560);
}
.timeline {
  list-style: none;
  margin: 0;
  padding: 0;
}
.timeline li {
  border-bottom: 1px solid rgba(212, 206, 194, 0.65);
  padding: 10px 0;
}
.timeline time {
  font-size: 11px;
  color: #8a847c;
}
.ev-type {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 700;
  color: #3d3a34;
}
.ev-payload {
  margin: 6px 0 0;
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
  color: #6b6560;
}
.center-load {
  display: flex;
  justify-content: center;
  padding: 48px;
}
.card--radar h3 {
  text-transform: none;
  letter-spacing: 0.02em;
  font-size: 15px;
  color: var(--brand-title, #141821);
}
.radar-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 6px;
}
.radar-sub {
  margin: 4px 0 0;
  font-size: 11px;
  line-height: 1.45;
  color: var(--brand-subtext, #6b6560);
  font-weight: 500;
}
.composite-ring {
  flex-shrink: 0;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: conic-gradient(
    rgba(184, 149, 79, 0.35) 0deg,
    rgba(184, 149, 79, 0.12) 120deg,
    rgba(26, 34, 48, 0.08) 240deg,
    rgba(184, 149, 79, 0.28) 360deg
  );
  border: 1px solid rgba(184, 149, 79, 0.45);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.composite-val {
  font-size: 20px;
  font-weight: 800;
  color: var(--brand-title, #141821);
  font-variant-numeric: tabular-nums;
  line-height: 1;
}
.composite-lbl {
  font-size: 9px;
  font-weight: 700;
  color: var(--brand-subtext, #6b6560);
  margin-top: 2px;
  letter-spacing: 0.04em;
}
.radar-bars {
  list-style: none;
  margin: 12px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.radar-bars li {
  display: grid;
  grid-template-columns: 72px 1fr 28px;
  align-items: center;
  gap: 8px;
  font-size: 11px;
}
.rb-name {
  color: var(--brand-subtext, #6b6560);
  font-weight: 600;
}
.rb-track {
  height: 6px;
  border-radius: 999px;
  background: rgba(26, 34, 48, 0.06);
  overflow: hidden;
}
.rb-track i {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, rgba(107, 90, 50, 0.35), rgba(184, 149, 79, 0.85));
}
.rb-num {
  text-align: right;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--brand-title, #141821);
}
</style>
