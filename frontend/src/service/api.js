import axios from 'axios'
import {
  getSession,
  getSessionVersion,
  clearSession,
  updateSessionTokens
} from '../session'

// 刷新客户端不使用业务响应拦截器，避免 /refresh 再次触发自身。
const api = axios.create({
  baseURL: '/api/v1/',
  headers: { 'Content-Type': 'application/json' }
})
const refreshClient = axios.create({ baseURL: '/' })
let refreshFlight = null

function isAccountRequest(config) {
  return /\/(login|signup)(?:\?|$)/.test(config.url || '')
}

function sessionMatches(config) {
  const session = getSession()
  return session && config._sessionVersion === getSessionVersion() &&
    config._accessToken === session.token
}

function clearRequestSession(config) {
  if (sessionMatches(config)) clearSession()
}

function invalidResponse() {
  return new Error('服务响应异常，请稍后重试')
}

function readPayload(response) {
  const payload = response && response.data
  if (!payload || typeof payload !== 'object' || typeof payload.code !== 'number') {
    throw invalidResponse()
  }
  return payload
}

function refreshSession(session, version) {
  if (refreshFlight && refreshFlight.version === version &&
      refreshFlight.token === session.token) return refreshFlight.promise

  const flight = { version, token: session.token }
  flight.promise = refreshClient.post('/refresh', {
    access_token: session.token,
    refresh_token: session.refresh_token
  }).then(response => {
    const payload = readPayload(response)
    if (payload.code !== 1000) {
      const error = new Error(typeof payload.msg === 'string' ? payload.msg : '刷新登录状态失败')
      error.businessCode = payload.code
      throw error
    }
    if (!updateSessionTokens(payload.data, version, session.token)) {
      const error = new Error('登录状态已变更，请重试')
      error.sessionChanged = true
      throw error
    }
  }).finally(() => {
    if (refreshFlight === flight) refreshFlight = null
  })
  refreshFlight = flight
  return flight.promise
}

api.interceptors.request.use(config => {
  const session = getSession()
  config.headers = config.headers || {}
  delete config.headers.Authorization
  delete config.headers.authorization
  config._accessToken = null
  config._sessionVersion = getSessionVersion()
  if (session && !isAccountRequest(config)) {
    config.headers.Authorization = 'Bearer ' + session.token
    config._accessToken = session.token
  }
  return config
}, error => Promise.reject(error))

api.interceptors.response.use(async response => {
  const payload = readPayload(response)
  const config = response.config || {}

  if (payload.code === 1009 || payload.code === 1007) {
    // 新设备已登录时，不能尝试用旧刷新令牌恢复会话。
    clearRequestSession(config)
    return payload
  }
  if (payload.code !== 1006 || isAccountRequest(config)) return payload

  const session = getSession()
  if (!session || !config._accessToken ||
      config._sessionVersion !== getSessionVersion()) return payload

  if (config._authRetried) {
    clearRequestSession(config)
    return payload
  }
  const retryConfig = { ...config, _authRetried: true }

  // 另一请求已经刷新了同一会话，直接复用新令牌。
  if (config._accessToken !== session.token) return api.request(retryConfig)

  try {
    await refreshSession(session, config._sessionVersion)
  } catch (error) {
    if (error.businessCode === 1006 || error.businessCode === 1007 ||
        error.businessCode === 1009) {
      clearRequestSession(config)
      return { ...payload, msg: '登录已失效，请重新登录' }
    }
    // 网络故障、服务繁忙或用户已切换账号，均不能删除新的登录状态。
    return Promise.reject(error)
  }
  if (!getSession() || config._sessionVersion !== getSessionVersion()) return payload
  return api.request(retryConfig)
}, error => Promise.reject(error))

export default api
