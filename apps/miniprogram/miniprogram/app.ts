import { sessionStore } from './stores/session'

App({
  onLaunch() {
    void sessionStore.restore()
  },
})
