import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Vant from 'vant'
import 'vant/lib/index.css'
import 'material-symbols/outlined.css'
import './style.css'
import App from './App.vue'
import router from './router'
import { i18n } from './i18n'

createApp(App).use(i18n).use(createPinia()).use(router).use(Vant).mount('#app')
