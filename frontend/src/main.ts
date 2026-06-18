import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './assets/main.css'
import { useRegistry } from './core/registry'
import { serialModule } from './serial'
import { Events } from '@wailsio/runtime'

const registry = useRegistry()
registry.register(serialModule)

// Listen for backend-emitted module contributions (ApplicationStarted → Emit).
Events.On('app:modules', (event: { data: unknown }) => {
  registry.mergeBackendContributions(event.data ?? [])
})

const app = createApp(App)
app.use(createPinia())
app.mount('#app')
