<template>
  <div class="screen-page">
    <header class="topbar">
      <button class="icon-btn" type="button" @click="$router.push('/me')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>账号安全</h1>
      <span class="material-symbols-outlined">security</span>
    </header>

    <main class="content">
      <section class="card">
        <h3>修改密码</h3>
        <van-field v-model="passwordForm.currentPassword" label="当前密码" type="password" placeholder="请输入当前密码" />
        <van-field v-model="passwordForm.newPassword" label="新密码" type="password" placeholder="至少6位" />
        <van-button type="primary" block @click="submitPasswordChange">确认修改</van-button>
      </section>

      <section class="card">
        <h3>登录会话</h3>
        <van-button type="danger" plain block @click="logout">退出登录</van-button>
      </section>
    </main>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { showFailToast, showSuccessToast } from 'vant'
import { useRouter } from 'vue-router'
import { changePassword } from '../../api'

const router = useRouter()
const passwordForm = reactive({ currentPassword: '', newPassword: '' })

const submitPasswordChange = async () => {
  if (!passwordForm.currentPassword || !passwordForm.newPassword) {
    showFailToast('请完整填写密码信息')
    return
  }
  try {
    await changePassword({
      currentPassword: passwordForm.currentPassword,
      newPassword: passwordForm.newPassword,
    })
    showSuccessToast('密码修改成功，请重新登录')
    logout()
  } catch (error) {
    showFailToast(error?.response?.data?.error || '修改密码失败')
  }
}

const logout = async () => {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  await router.replace('/')
}
</script>

<style scoped>
.screen-page { min-height: 100vh; background: #f6f8ff; }
.topbar { position: sticky; top: 0; z-index: 20; height: 64px; border-bottom: 1px solid #d8e0f5; background: #fff; display: flex; align-items: center; justify-content: space-between; padding: 0 12px; }
.topbar h1 { margin: 0; font-size: 20px; color: #102a5c; }
.icon-btn { width: 36px; height: 36px; border: 1px solid #d8e0f5; border-radius: 999px; background: #fff; color: #3156cb; display: flex; align-items: center; justify-content: center; }
.content { padding: 14px; display: flex; flex-direction: column; gap: 12px; }
.card { border: 1px solid #d8e0f5; border-radius: 12px; background: #fff; box-shadow: 0 8px 18px rgba(15, 40, 120, 0.05); padding: 12px; display: flex; flex-direction: column; gap: 10px; }
.card h3 { margin: 0; font-size: 15px; color: #102a5c; }
</style>
