const backend = process.env.BACKEND_PROXY_TARGET || 'http://127.0.0.1:8081'

module.exports = {
  publicPath: '/',
  assetsDir: 'static',
  productionSourceMap: false,
  parallel: false,
  pages: {
    index: { entry: 'src/main.js', title: 'bluebell · GoWeb 社区' }
  },
  devServer: {
    host: '127.0.0.1',
    port: 8080,
    proxy: {
      '/api/v1': { target: backend, changeOrigin: true },
      '/refresh': { target: backend, changeOrigin: true }
    }
  }
}
