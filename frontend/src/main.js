import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import './styles/index.scss'

const app = createApp(App)

// 全量注册 Element Plus 图标，模板中可直接使用 <el-icon><Folder /></el-icon>
for (const [name, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(name, component)
}

app.use(createPinia())
app.use(router)
// 基础 z-index 抬高，保证预览层（2100）之上的弹窗、消息与提示仍可见
app.use(ElementPlus, { locale: zhCn, zIndex: 3000 })

app.mount('#app')
