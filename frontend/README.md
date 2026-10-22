# Vue 前端

这里保存与当前 Go 后端配套的 Vue 源码。页面保留帖子列表、详情、发帖、投票、注册和登录。

## 开发与打包

在已安装 Node.js/npm 的环境中执行：

```bash
cd frontend
npm ci
npm run serve
```

开发页面默认使用 8080 端口，`/api/v1` 和 `/refresh` 代理到 `http://127.0.0.1:8081`。后端地址不同时，可以设置 `BACKEND_PROXY_TARGET`。Windows 启动脚本会在必要时尝试定位 WSL 后端。

```bash
npm run lint
npm run build:backend
```

`build:backend` 将产物写入后端的 `templates/index.html` 和 `static/`。Docker 运行镜像会复制这些已生成文件，所以更新前端源码后应先重新打包，再从项目根目录执行：

```bash
docker compose up -d --build app
```

页面使用 hash 路由，例如 `/#/login`、`/#/publish`，不需要后端为每个前端页面增加路由。

## 接口约定

- 登录：`POST /api/v1/login`；读取 `userID`、`username`、`accessTokne`、`refreshToken`，保留后端现有字段拼写。
- 注册：确认密码字段为 `repassword`。
- 刷新：`POST /refresh`，发送 `access_token` 和 `refresh_token`。前端合并并发刷新请求，并只重试一次。
- 被其他设备登录替换的会话退出登录，不尝试刷新恢复。
- 帖子和用户 ID 保持字符串；`community_id` 按后端接口要求使用数字。
- 详情页社区数据读取 `post.community`，不使用不存在的顶层 `community_name`。

生产构建不发布 source map。实际密码、JWT 签名密钥与后端配置文件都不应放入前端。
