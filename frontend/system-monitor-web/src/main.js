import { createApp } from 'vue'
import App from './App.vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './assets/global.css' // 可选：你自定义全局样式

const app = createApp(App)
app.use(ElementPlus)
app.mount('#app')
