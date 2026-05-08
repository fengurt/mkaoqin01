<template>
  <nav class="app-bottom-nav">
    <router-link class="app-nav-item" :class="{ active: current === 'home' }" to="/home">
      <span class="material-symbols-outlined">home</span>
      <span>首页</span>
    </router-link>
    <router-link class="app-nav-item" :class="{ active: current === 'schedule' }" to="/schedule">
      <span class="material-symbols-outlined">event_note</span>
      <span>行程</span>
    </router-link>
    <router-link class="app-nav-item" :class="{ active: current === 'me' }" to="/me">
      <span class="material-symbols-outlined">person</span>
      <span>我的</span>
    </router-link>
  </nav>
</template>

<script setup>
defineProps({ current: { type: String, required: true } })
</script>

<style scoped>
.app-bottom-nav {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: 64px;
  border-top: 1px solid #c3c6d7;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-around;
  z-index: 60;
}

.app-nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  color: #64748b;
  text-decoration: none;
  font-size: 12px;
  min-width: 70px;
}

.app-nav-item.active {
  color: #004ac6;
  font-weight: 700;
}

</style>
