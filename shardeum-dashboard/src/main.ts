import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useNetworkStore } from './stores/network'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Initialize network store
const networkStore = useNetworkStore()
networkStore.initializeNetwork()

app.mount('#app')