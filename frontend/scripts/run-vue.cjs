const fs = require('fs')
const path = require('path')
const { spawnSync } = require('child_process')
const net = require('net')

const root = path.resolve(__dirname, '..')
const temporary = path.join(root, '.tmp')
fs.mkdirSync(temporary, { recursive: true })
const env = { ...process.env, TEMP: temporary, TMP: temporary, TMPDIR: temporary }
// Vue CLI 4 使用 webpack 4；新版本 Node 需要兼容其旧的摘要算法。
if (Number(process.versions.node.split('.')[0]) >= 17 &&
    !(env.NODE_OPTIONS || '').includes('--openssl-legacy-provider')) {
  env.NODE_OPTIONS = ((env.NODE_OPTIONS || '') + ' --openssl-legacy-provider').trim()
}
function reachable(host) {
  return new Promise(resolve => {
    const socket = net.createConnection({ host, port: 8081 })
    const finish = value => { socket.destroy(); resolve(value) }
    socket.setTimeout(800, () => finish(false))
    socket.once('connect', () => finish(true))
    socket.once('error', () => finish(false))
  })
}

async function main() {
  const configured = env.BACKEND_PROXY_TARGET ||
    ['.env', '.env.local', '.env.development', '.env.development.local'].some(name => {
      const file = path.join(root, name)
      return fs.existsSync(file) && /^\s*BACKEND_PROXY_TARGET\s*=/m.test(fs.readFileSync(file, 'utf8'))
    })
  if (process.argv[2] === 'serve' && process.platform === 'win32' && !configured &&
      !await reachable('127.0.0.1')) {
    const result = spawnSync('wsl.exe', ['-d', 'Ubuntu', '-e', 'hostname', '-I'], {
      windowsHide: true, timeout: 10000, encoding: 'utf8'
    })
    if (result.status === 0) {
      const candidates = result.stdout.trim().split(/\s+/).filter(value => /^\d+\.\d+\.\d+\.\d+$/.test(value))
      for (const address of candidates) {
        if (await reachable(address)) {
          env.BACKEND_PROXY_TARGET = 'http://' + address + ':8081'
          process.stdout.write('后端代理地址：' + env.BACKEND_PROXY_TARGET + '\n')
          break
        }
      }
    }
  }
  const result = spawnSync(process.execPath, [
    require.resolve('@vue/cli-service/bin/vue-cli-service.js'),
    ...process.argv.slice(2)
  ], { cwd: root, env, stdio: 'inherit' })
  if (result.error) {
    process.stderr.write(result.error.message + '\n')
    process.exit(1)
  }
  process.exit(result.status === null ? 1 : result.status)
}

main().catch(error => {
  process.stderr.write(error.message + '\n')
  process.exit(1)
})
