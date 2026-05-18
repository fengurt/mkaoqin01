import { createI18n } from 'vue-i18n'
import zhCN from '../locales/zh-CN.json'
import en from '../locales/en.json'

const storedLocale = typeof localStorage !== 'undefined' ? localStorage.getItem('locale') : ''
const browserZh =
  typeof navigator !== 'undefined' &&
  (navigator.language || '').toLowerCase().startsWith('zh')

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: storedLocale || (browserZh ? 'zh-CN' : 'en'),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    en,
  },
})

export function setLocale(next) {
  const value = next === 'en' ? 'en' : 'zh-CN'
  const g = i18n.global
  if (typeof g.locale === 'string') {
    g.locale = value
  } else if (g.locale && typeof g.locale === 'object' && 'value' in g.locale) {
    g.locale.value = value
  }
  localStorage.setItem('locale', value)
}
