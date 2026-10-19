const fs = require('fs')
const path = require('path')

const frontend = path.resolve(__dirname, '..')
const backend = path.resolve(frontend, '..')
const dist = path.join(frontend, 'dist')
if (!fs.existsSync(path.join(backend, 'go.mod'))) {
  throw new Error('Run this script from the frontend directory inside the Go project.')
}
if (!fs.existsSync(path.join(dist, 'index.html')) || !fs.existsSync(path.join(dist, 'static'))) {
  throw new Error('Build the frontend before publishing: npm run build')
}
fs.mkdirSync(path.join(backend, 'templates'), { recursive: true })
fs.mkdirSync(path.join(backend, 'static'), { recursive: true })
fs.cpSync(path.join(dist, 'static'), path.join(backend, 'static'), { recursive: true })
fs.copyFileSync(path.join(dist, 'index.html'), path.join(backend, 'templates', 'index.html'))
process.stdout.write('Updated backend templates/index.html and static/.\n')
