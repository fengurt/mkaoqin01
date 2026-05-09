import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import HomePage from '../views/HomePage.vue'
import SchedulePage from '../views/SchedulePage.vue'
import MyPage from '../views/MyPage.vue'
import MyAttendanceView from '../views/screens/MyAttendanceView.vue'
import TeamAttendanceView from '../views/screens/TeamAttendanceView.vue'
import DetailedAttendanceReport from '../views/screens/DetailedAttendanceReport.vue'
import EmployeeProfile from '../views/screens/EmployeeProfile.vue'
import AccountSecurityView from '../views/screens/AccountSecurityView.vue'
import AdminAccountManagementView from '../views/screens/AdminAccountManagementView.vue'
import AdminScheduleQuickConfigView from '../views/screens/AdminScheduleQuickConfigView.vue'
import AdminScheduleTypesView from '../views/screens/AdminScheduleTypesView.vue'
import TeamMemberHistoryView from '../views/screens/TeamMemberHistoryView.vue'

const routes = [
  { path: '/', component: Login },
  { path: '/home', component: HomePage },
  { path: '/schedule', component: SchedulePage },
  { path: '/me', component: MyPage },
  { path: '/employee', redirect: '/home' },
  { path: '/admin', redirect: '/home' },
  { path: '/my-attendance/day', component: MyAttendanceView, props: { period: 'day' } },
  { path: '/my-attendance/week', component: MyAttendanceView, props: { period: 'week' } },
  { path: '/my-attendance/month', component: MyAttendanceView, props: { period: 'month' } },
  { path: '/team-attendance/day', component: TeamAttendanceView, props: { period: 'day' } },
  { path: '/team-attendance/week', component: TeamAttendanceView, props: { period: 'week' } },
  { path: '/team-attendance/month', component: TeamAttendanceView, props: { period: 'month' } },
  { path: '/admin/report', component: DetailedAttendanceReport },
  { path: '/employee/profile', component: EmployeeProfile },
  { path: '/me/security', component: AccountSecurityView },
  { path: '/me/accounts', component: AdminAccountManagementView },
  { path: '/me/schedule-quick-config', component: AdminScheduleQuickConfigView },
  { path: '/me/schedule-types-config', component: AdminScheduleTypesView },
  { path: '/team-member/:userId', component: TeamMemberHistoryView },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0, left: 0 }
  },
})

router.beforeEach((to) => {
  if (to.path === '/') return true
  const token = localStorage.getItem('token')
  if (!token) return '/'
  if (to.path.startsWith('/team-attendance') || to.path === '/admin/report' || to.path === '/me/accounts' || to.path === '/me/schedule-quick-config' || to.path === '/me/schedule-types-config' || to.path.startsWith('/team-member/')) {
    const user = JSON.parse(localStorage.getItem('user') || '{}')
    if (user.role !== 'admin') return '/me'
  }
  return true
})

export default router
