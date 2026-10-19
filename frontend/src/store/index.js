import Vue from 'vue'
import Vuex from 'vuex'
import { getSession, saveSession, clearSession, subscribeSession } from '../session'

Vue.use(Vuex)

function emptySession() {
  return { token: null, refresh_token: null, user_id: null, user_name: null }
}

const store = new Vuex.Store({
  state: {
    loginResult: getSession() || emptySession()
  },
  mutations: {
    init(state) {
      state.loginResult = getSession() || emptySession()
    },
    login(state, loginResult) {
      state.loginResult = saveSession(loginResult) || emptySession()
    },
    logout(state) {
      clearSession()
      state.loginResult = emptySession()
    },
    syncSession(state, loginResult) {
      state.loginResult = loginResult || emptySession()
    }
  },
  actions: {},
  getters: {
    isLogin: state => Boolean(state.loginResult.user_id && state.loginResult.token),
    userID: state => state.loginResult.user_id,
    username: state => state.loginResult.user_name,
    accessToken: state => state.loginResult.token
  }
})

subscribeSession(loginResult => store.commit('syncSession', loginResult))

export default store
