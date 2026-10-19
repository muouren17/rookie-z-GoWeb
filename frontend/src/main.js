import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import axios from './service/api'

Vue.prototype.$axios = axios
Vue.config.productionTip = false

store.commit('init')

router.beforeEach((to, from, next) => {
  if (to.meta.requireAuth && !store.getters.isLogin) {
    return next({ name: 'Login', query: { redirect: to.fullPath } })
  }
  if (to.name === 'Login' && store.getters.isLogin) {
    return next({ name: 'Home' })
  }
  return next()
})

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
