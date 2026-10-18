# GoWeb Swagger 接口文档

使用 **swaggo/swag 从 Go 文档注释生成 Swagger 2.0**，覆盖当前注册的 10 个接口。本文介绍独立文档页面；运行中的后端也提供 `/swagger/index.html`，对应根目录 `docs` 包。

## 注释位置

- `main.go`：标题、版本、host、BasePath、schemes 和 ApiKeyAuth 定义。
- `controllers/user.go`：注册、登录、刷新令牌。
- `controllers/community.go`：社区列表/详情、创建帖子、帖子详情、两种帖子列表。
- `controllers/vote.go`：赞同、反对、取消投票。
- `controllers/response.go`：统一响应格式的字段说明。

没有新增业务结构体。请求和响应复用现有模型或使用注释中的内联对象描述。投票的字符串字段使用内联描述，避免 swag 把 `json:",string"` 与数字 `oneof` 组合成不一致的文档类型。

## 更新文档

在 **WSL Ubuntu** 的项目根目录运行：

```bash
sh docs/swagger/generate.sh
```

脚本执行的核心命令为：

```bash
swag init -g main.go -o docs/swagger --outputTypes json,yaml --exclude docs,tmp,logs,vendor
```

此脚本只更新独立页面使用的 JSON/YAML，不更新运行时导入的根目录 `docs/docs.go`。更新后端内置页面时，还应在项目根目录运行 `swag init -g main.go -o docs --exclude docs,tmp,logs,vendor`。

需要先安装 `swaggo/swag` CLI，例如执行 `go install github.com/swaggo/swag/cmd/swag@v1.16.6`，并将 Go 的 bin 目录加入 PATH。脚本也会尝试寻找 `$HOME/.local/bin/swag`。

## 浏览文档

先按原有方式启动 Go 后端（当前端口8081），再启动独立文档页面：

```bash
python3 -B docs/swagger/serve.py
```

Windows 浏览器打开 **http://localhost:8090/**。保持终端运行，`Ctrl+C` 关闭文档服务。后端未运行时仍可以浏览文档，调试接口则需要后端运行。

端口不同时：

```bash
python3 -B docs/swagger/serve.py --port 8091 --backend http://127.0.0.1:8081
```

文档服务只监听本机，使用 Python 标准库。Swagger UI 资源均在本地，无需 CDN。页面通过独立文档服务转发到后端，避免修改业务服务的路由或跨域配置。页面仅在内存中将生成规范的 host 替换为文档服务地址，下载的规范保留 `main.go` 中声明的 `localhost:8081`。

## 调试和认证

1. 打开 `POST /api/v1/login`，点击 Try it out，填写已有账号密码，再点击 Execute。
2. 复制响应中的 `data.accessTokne`，注意这是当前后端实际字段名。
3. 点击 Authorize，填入完整的头部值：**`Bearer 实际access_token`**。中间一个空格，不加双引号。当前是 Swagger 2.0 的 ApiKeyAuth，需要包含 Bearer 前缀。
4. 社区、帖子和投票接口会携带该 Authorization。刷新接口为根路径 `POST /refresh`，在JSON请求体中提交配对的 `access_token`、`refresh_token`，不要求 Authorization。
5. 刷新成功后，将返回的 `data.access_token` 重新填入 Authorize。访问令牌未过期时，刷新接口会返回原令牌对。

只有手动点击 Execute 才会调用接口；注册、发帖、投票会真实修改数据。文档服务不保存账号密码或令牌，页面不持久保存授权。

## 当前接口约定

- 业务成功 `code=1000`。正常业务错误也返回 HTTP 200，必须检查 JSON 的 code。
- 注册、登录、刷新公开，其他7个接口要求当前有效会话。
- 登录返回 `accessTokne`、`refreshToken`；刷新使用 `access_token`、`refresh_token`。
- 帖子和作者ID以字符串返回，社区ID为数字。
- 投票请求如 `{"post_id":"10000000000000001","direction":"1"}`，两个值都必须是JSON字符串；direction的 `"1"`、`"-1"`、`"0"` 分别为赞同、反对、取消，省略时当前按0处理。
- `/posts` 只处理page、size，按MySQL创建时间降序；`/posts2` 还支持order、community_id，省略社区或传0查询全部。
- `vote_num` 为赞同票数，0时省略；普通详情和 `/posts` 不查询票数。
- 空的 `/posts` 返回data为[]；`/posts2` 的Redis无匹配ID时data为null。注册、发帖、投票成功时省略data。

没有把尚未实现的业务校验写成已实现的约束。匿名请求的必填字段和JSON示例已写在接口描述中。

## 文件

- `swagger.json`、`swagger.yaml`：由源码注释生成的 Swagger 2.0 文档。
- `openapi.json`、`openapi.yaml`：兼容之前下载地址的副本，内容同样为 Swagger 2.0，不再是先前的手写快照。
- `generate.sh`：从注释更新文档和兼容副本。
- `index.html`、`init.js`、`vendor/`：本地 Swagger UI。
- `serve.py`：独立文档服务。

修改接口后同步维护函数上的注释，再运行生成脚本；无需手工编辑生成的JSON/YAML。

[swaggo 官方注释说明](https://github.com/swaggo/swag/blob/v1.16.6/README.md)
