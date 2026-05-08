<template>
  <div class="screen-page">
    <header class="topbar">
      <button class="icon-btn" type="button" @click="$router.push('/me')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>账号管理</h1>
      <span class="material-symbols-outlined">manage_accounts</span>
    </header>

    <main class="content">
      <section class="card">
        <div class="head-row">
          <h3>系统账号列表</h3>
          <span class="meta">{{ users.length }} 个账号</span>
        </div>
        <van-field v-model="keyword" placeholder="搜索账号或昵称" clearable />
        <div v-if="loadUsersFailed" class="error">账号列表加载失败，请重新登录后再试</div>
        <div v-else-if="filteredUsers.length === 0" class="empty">暂无匹配账号</div>
        <div v-else class="list-wrap">
          <div v-for="item in filteredUsers" :key="item.id" class="user-row">
            <div>
              <div class="user-name">{{ item.displayName }}（{{ item.role === 'admin' ? '管理员' : '员工' }}）</div>
              <div class="user-account">账号：{{ item.account }}</div>
            </div>
            <van-button size="small" type="primary" plain @click="openResetPassword(item)">重置密码</van-button>
          </div>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showResetPassword" position="bottom" round :style="{ height: '42%' }">
      <div class="popup-body">
        <h3>重置用户密码</h3>
        <p class="popup-sub">{{ resetTarget.displayName || '-' }}（{{ resetTarget.account || '-' }}）</p>
        <van-field v-model="resetPasswordValue" label="新密码" type="password" placeholder="至少6位" />
        <van-button type="primary" block @click="submitResetPassword">确认重置</van-button>
      </div>
    </van-popup>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { showFailToast, showSuccessToast } from 'vant'
import { getAuthUsers, resetUserPassword } from '../../api'

const users = ref([])
const keyword = ref('')
const loadUsersFailed = ref(false)
const showResetPassword = ref(false)
const resetTarget = ref({ id: 0, account: '', displayName: '' })
const resetPasswordValue = ref('')

const filteredUsers = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  if (!key) return users.value
  return users.value.filter((item) => {
    const accountText = String(item.account || '').toLowerCase()
    const displayNameText = String(item.displayName || '').toLowerCase()
    return accountText.includes(key) || displayNameText.includes(key)
  })
})

const loadUsers = async () => {
  try {
    const response = await getAuthUsers()
    users.value = response.data.items || []
    loadUsersFailed.value = false
  } catch {
    loadUsersFailed.value = true
    users.value = []
  }
}

const openResetPassword = (targetUser) => {
  resetTarget.value = targetUser
  resetPasswordValue.value = ''
  showResetPassword.value = true
}

const submitResetPassword = async () => {
  if (!resetTarget.value.id || resetPasswordValue.value.length < 6) {
    showFailToast('请输入至少6位的新密码')
    return
  }
  try {
    await resetUserPassword({ userId: resetTarget.value.id, newPassword: resetPasswordValue.value })
    showSuccessToast('用户密码已重置')
    showResetPassword.value = false
  } catch (error) {
    showFailToast(error?.response?.data?.error || '重置密码失败')
  }
}

onMounted(loadUsers)
</script>

<style scoped>
.screen-page { min-height: 100vh; background: #f6f8ff; }
.topbar { position: sticky; top: 0; z-index: 20; height: 64px; border-bottom: 1px solid #d8e0f5; background: #fff; display: flex; align-items: center; justify-content: space-between; padding: 0 12px; }
.topbar h1 { margin: 0; font-size: 20px; color: #102a5c; }
.icon-btn { width: 36px; height: 36px; border: 1px solid #d8e0f5; border-radius: 999px; background: #fff; color: #3156cb; display: flex; align-items: center; justify-content: center; }
.content { padding: 14px; }
.card { border: 1px solid #d8e0f5; border-radius: 12px; background: #fff; box-shadow: 0 8px 18px rgba(15, 40, 120, 0.05); padding: 12px; display: flex; flex-direction: column; gap: 10px; }
.head-row { display: flex; align-items: center; justify-content: space-between; }
.head-row h3 { margin: 0; font-size: 15px; color: #102a5c; }
.meta { font-size: 12px; color: #64748b; }
.error { color: #dc2626; font-size: 13px; }
.empty { color: #64748b; font-size: 13px; }
.list-wrap { display: flex; flex-direction: column; gap: 8px; }
.user-row { display: flex; justify-content: space-between; align-items: center; gap: 10px; border: 1px solid #eef2f7; border-radius: 10px; padding: 10px; }
.user-name { font-size: 13px; font-weight: 700; color: #0f172a; }
.user-account { font-size: 12px; color: #64748b; margin-top: 4px; }
.popup-body { padding: 16px; display: flex; flex-direction: column; gap: 10px; }
.popup-body h3 { margin: 0; }
.popup-sub { margin: 0; color: #64748b; font-size: 12px; }
</style>
