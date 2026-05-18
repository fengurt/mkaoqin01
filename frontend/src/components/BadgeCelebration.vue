<template>
  <van-overlay :show="visible" class="badge-overlay" :z-index="3000">
    <div class="badge-overlay-inner" role="dialog" aria-modal="true" @click.stop>
      <div class="badge-card" :class="`badge-card--${tierClass}`">
        <p class="badge-kind">{{ kindLabel }}</p>
        <h2 class="badge-title">{{ titleText }}</h2>
        <p class="badge-sub">{{ $t('badges.celebrateSubtitle') }}</p>
        <van-button type="primary" block round class="badge-ok" @click="onAck">
          {{ $t('badges.gotIt') }}
        </van-button>
      </div>
    </div>
  </van-overlay>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { acknowledgeBadges } from '../api'

const { t, locale } = useI18n()

const visible = ref(false)
const queue = ref([])
const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')

const current = computed(() => queue.value[0] || null)

const tierClass = computed(() => String(current.value?.tier || 'BRONZE').toLowerCase())

const kindLabel = computed(() => {
  const k = current.value?.kind
  if (k === 'USER_MEDAL') return t('badges.kindUserMedal')
  if (k === 'ACHIEVEMENT_BADGE') return t('badges.kindAchievement')
  if (k === 'MEMBER_BADGE') return t('badges.kindMember')
  return t('badges.kindUserBadge')
})

const titleText = computed(() => {
  const ti = current.value?.titleI18n
  if (ti && typeof ti === 'object') {
    const loc = locale.value === 'en' ? 'en' : 'zh-CN'
    return ti[loc] || ti['zh-CN'] || current.value?.code || ''
  }
  return current.value?.code || ''
})

function burstConfetti() {
  import('canvas-confetti')
    .then((mod) => {
      const confetti = mod.default
      confetti({
        particleCount: 90,
        spread: 70,
        origin: { y: 0.35 },
        colors: ['#2563eb', '#f59e0b', '#10b981', '#a855f7', '#fff'],
      })
    })
    .catch(() => {})
}

async function play(badges) {
  if (!Array.isArray(badges) || badges.length === 0) return
  queue.value = [...badges]
  visible.value = true
  burstConfetti()
}

async function onAck() {
  const item = queue.value.shift()
  if (item?.id) {
    try {
      await acknowledgeBadges({ userId: user.id || 1, badgeIds: [item.id] })
    } catch {
      /* non-fatal */
    }
  }
  if (queue.value.length === 0) {
    visible.value = false
    return
  }
  burstConfetti()
}

defineExpose({ play })
</script>

<style scoped>
.badge-overlay :deep(.van-overlay__content) {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
.badge-overlay-inner {
  width: 100%;
  padding: 24px;
  display: flex;
  justify-content: center;
}
.badge-card {
  width: min(360px, 92vw);
  border-radius: 16px;
  padding: 22px 20px 18px;
  background: linear-gradient(160deg, #0f172a, #1e293b);
  color: #f8fafc;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.45);
  text-align: center;
}
.badge-card--gold {
  background: linear-gradient(160deg, #78350f, #ca8a04);
}
.badge-card--silver {
  background: linear-gradient(160deg, #334155, #94a3b8);
}
.badge-card--bronze {
  background: linear-gradient(160deg, #431407, #9a3412);
}
.badge-kind {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  opacity: 0.85;
}
.badge-title {
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 800;
  line-height: 1.25;
}
.badge-sub {
  margin: 0 0 18px;
  font-size: 13px;
  opacity: 0.88;
}
.badge-ok {
  font-weight: 700;
}
</style>
