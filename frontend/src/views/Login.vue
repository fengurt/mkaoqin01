<template>
  <div class="st-page-login">
    <main class="st-login-main">
      <header class="st-login-header">
        <div class="st-login-logo-wrap">
          <span class="material-symbols-outlined st-login-logo">graphic_eq</span>
        </div>
        <div>
          <h1 class="st-login-title">Intervoice</h1>
          <p class="st-login-subtitle">智能考勤与状态门户</p>
        </div>
      </header>

      <form class="st-login-form" @submit.prevent="handleLogin">
        <div class="st-login-field-wrap">
          <label class="st-login-label" for="account">账号</label>
          <input id="account" v-model="account" class="st-login-input" type="text" placeholder="请输入工号或邮箱" />
        </div>

        <div class="st-login-field-wrap">
          <label class="st-login-label" for="password">密码</label>
          <input id="password" v-model="password" class="st-login-input" type="password" placeholder="请输入密码" />
        </div>

        <button class="st-login-btn-primary" type="submit">
          登录
          <span class="material-symbols-outlined st-login-btn-icon">arrow_forward</span>
        </button>

      </form>

    </main>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { login } from '../api'

const router = useRouter()
const account = ref('132369')
const password = ref('123456a')

const routeByRole = (role) => {
  void role
  router.push('/home')
}

const storeSession = (data) => {
  localStorage.setItem('token', data.token)
  localStorage.setItem('user', JSON.stringify(data.user))
}

const handleLogin = async () => {
  try {
    const response = await login({ account: account.value, password: password.value })
    storeSession(response.data)
    routeByRole(response.data.user.role)
  } catch {
    showToast('登录失败，请检查账号密码')
  }
}
</script>

<style scoped>
.st-page-login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  padding: 24px;
}

.st-login-main {
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.st-login-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 10px;
}

.st-login-logo-wrap {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  border: 1px solid #c3c6d7;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
}

.st-login-logo { font-size: 40px; color: #2563eb; }
.st-login-title { margin: 0; font-size: 36px; line-height: 44px; color: #191c1d; }
.st-login-subtitle { margin: 4px 0 0; font-size: 16px; color: #434655; }

.st-login-form { display: flex; flex-direction: column; gap: 16px; }
.st-login-field-wrap { display: flex; flex-direction: column; gap: 4px; }
.st-login-label { font-size: 12px; font-weight: 600; color: #191c1d; }
.st-login-input {
  height: 48px;
  border-radius: 8px;
  border: 1px solid #c3c6d7;
  padding: 0 12px;
  font-size: 16px;
}

.st-login-btn-primary {
  height: 48px;
  border-radius: 8px;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 20px;
  font-weight: 600;
}

.st-login-btn-primary { background: #2563eb; color: #fff; }
.st-login-btn-icon { font-size: 20px; }

</style>
