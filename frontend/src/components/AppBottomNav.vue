<template>
  <nav class="app-bottom-nav" role="navigation" aria-label="主导航">
    <router-link
      class="app-nav-item"
      :class="{ active: activeTab === 'home' }"
      :aria-current="activeTab === 'home' ? 'page' : undefined"
      to="/home"
    >
      <span class="material-symbols-outlined" aria-hidden="true">home</span>
      <span>首页</span>
    </router-link>
    <router-link
      class="app-nav-item"
      :class="{ active: activeTab === 'schedule' }"
      :aria-current="activeTab === 'schedule' ? 'page' : undefined"
      to="/schedule"
    >
      <span class="material-symbols-outlined" aria-hidden="true">event_note</span>
      <span>行程</span>
    </router-link>
    <router-link
      class="app-nav-item"
      :class="{ active: activeTab === 'me' }"
      :aria-current="activeTab === 'me' ? 'page' : undefined"
      to="/me"
    >
      <span class="material-symbols-outlined" aria-hidden="true">person</span>
      <span>我的</span>
    </router-link>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const activeTab = computed(() => {
  const path = route.path
  if (path === '/home') return 'home'
  if (path.startsWith('/schedule') || path.startsWith('/my-attendance')) return 'schedule'
  return 'me'
})
</script>

<style scoped>
.app-bottom-nav {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 60;
  min-height: calc(64px + env(safe-area-inset-bottom, 0px));
  padding-bottom: env(safe-area-inset-bottom, 0px);
  border-top: 1px solid var(--brand-border, #d4cec2);
  background: rgba(253, 251, 247, 0.94);
  backdrop-filter: blur(10px);
  display: flex;
  align-items: stretch;
  justify-content: space-around;
  box-shadow: 0 -4px 20px rgba(15, 23, 42, 0.06);
}

.app-nav-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 8px 4px 10px;
  color: var(--brand-subtext, #64748b);
  text-decoration: none;
  font-size: 11px;
  font-weight: 500;
  -webkit-tap-highlight-color: transparent;
  transition: color 0.15s ease, transform 0.12s ease;
}

.app-nav-item:active {
  transform: scale(0.97);
}

.app-nav-item .material-symbols-outlined {
  font-size: 24px;
  opacity: 0.88;
}

.app-nav-item.active {
  color: var(--accent-gold, #b8954f);
  font-weight: 700;
}

.app-nav-item.active .material-symbols-outlined {
  opacity: 1;
  font-variation-settings: 'FILL' 0, 'wght' 600, 'GRAD' 0, 'opsz' 24;
}
</style>
