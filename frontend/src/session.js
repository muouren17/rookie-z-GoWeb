// 登录字段在这里统一，页面不直接解析或写入令牌。
const STORAGE_KEY = 'loginResult'
const listeners = []
let sessionVersion = 0

function getStorage() {
  try {
    if (typeof window !== 'undefined') return window.localStorage
    if (typeof localStorage !== 'undefined') return localStorage
  } catch (error) {
    // 浏览器可能禁止持久化；当前页面仍可使用内存中的会话。
  }
  return null
}

function text(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function tokenText(value) {
  const result = text(value)
  return result === 'undefined' || result === 'null' ? '' : result
}

export function normalizeSession(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const rawID = value.userID !== undefined ? value.userID : value.user_id
  let userID = ''
  if (typeof rawID === 'string') userID = rawID
  else if (Number.isSafeInteger(rawID) && rawID > 0) userID = String(rawID)
  if (!/^\d+$/.test(userID) || !/[1-9]/.test(userID)) return null

  const userName = text(value.username !== undefined ? value.username : value.user_name)
  const token = tokenText(value.accessTokne !== undefined ? value.accessTokne : value.token)
  const refreshToken = tokenText(value.refreshToken !== undefined ? value.refreshToken : value.refresh_token)
  if (!userName || !token || !refreshToken) return null
  return { user_id: userID, user_name: userName, token, refresh_token: refreshToken }
}

function readStoredSession() {
  const storage = getStorage()
  if (!storage) return null
  try {
    const raw = storage.getItem(STORAGE_KEY)
    if (!raw) return null
    const result = normalizeSession(JSON.parse(raw))
    if (!result) storage.removeItem(STORAGE_KEY)
    return result
  } catch (error) {
    try { storage.removeItem(STORAGE_KEY) } catch (ignored) { /* 持久化不可用 */ }
    return null
  }
}

let currentSession = readStoredSession()

export function getSession() {
  return currentSession ? { ...currentSession } : null
}

export function hasSession() {
  return currentSession !== null
}

export function getSessionVersion() {
  return sessionVersion
}

function notify() {
  listeners.slice().forEach(listener => listener(getSession()))
}

function persist() {
  const storage = getStorage()
  if (!storage) return
  try {
    if (currentSession) storage.setItem(STORAGE_KEY, JSON.stringify(currentSession))
    else storage.removeItem(STORAGE_KEY)
  } catch (error) {
    // 持久化失败不会把一次成功登录变成页面异常。
  }
}

export function saveSession(value) {
  currentSession = normalizeSession(value)
  sessionVersion += 1
  persist()
  notify()
  return getSession()
}

export function clearSession() {
  currentSession = null
  sessionVersion += 1
  persist()
  notify()
}

export function updateSessionTokens(tokens, expectedVersion, expectedAccessToken) {
  if (!currentSession || sessionVersion !== expectedVersion ||
      currentSession.token !== expectedAccessToken) return false
  const token = tokenText(tokens && tokens.access_token)
  const refreshToken = tokenText(tokens && tokens.refresh_token)
  if (!token || !refreshToken) return false
  currentSession = { ...currentSession, token, refresh_token: refreshToken }
  persist()
  notify()
  return true
}

export function subscribeSession(listener) {
  listeners.push(listener)
  return () => {
    const index = listeners.indexOf(listener)
    if (index !== -1) listeners.splice(index, 1)
  }
}

if (typeof window !== 'undefined' && window.addEventListener) {
  window.addEventListener('storage', event => {
    if (event.key !== STORAGE_KEY && event.key !== null) return
    currentSession = readStoredSession()
    sessionVersion += 1
    notify()
  })
}
