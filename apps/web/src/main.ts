import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import '@formtally/design-tokens/tokens.css'
import './styles/main.css'

createApp(App).use(router).mount('#app')
