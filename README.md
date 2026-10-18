# 一只菜鸟z的GoWeb项目

使用 Go、Gin、MySQL 和 Redis 实现的论坛后端练习项目。

## 功能范围

- 用户注册、登录，bcrypt 密码校验及旧密码格式登录后迁移。
- JWT 访问令牌与刷新令牌，Redis 当前会话校验。
- 社区列表与详情、帖子发布与查询、分页和排序。
- 帖子赞同、反对、取消投票。
- 请求日志、统一响应、限流、优雅退出和 Air 热重载。
- Swagger 接口文档。

## 开发环境

- Go：版本要求以 `go.mod` 为准，当前为 1.26.8。
- MySQL、Redis。
- 推荐在 WSL Ubuntu 或其他 Linux 环境运行。

## 启动

在项目根目录执行：

```bash
cp config.example.yaml config.yaml
```

编辑 `config.yaml`，填写 MySQL、Redis 的连接信息。先创建配置中指定的 MySQL 数据库，程序启动时会执行 `models/create_table.sql`，创建缺少的表；SQL 不包含社区等初始化数据，也不会自动升级已有表结构。

`secret_key` 必须填写至少 32 字节的随机密钥，可以使用以下命令生成后填入配置：

```bash
openssl rand -hex 32
```

`mysql.secret` 仅用于兼容历史密码格式。新数据库可以留空；已有旧账号迁移前，应保留其历史值。

如果 MySQL 安装在 Windows、后端运行在 WSL，请将 `mysql.host` 配置为 WSL 能访问到的 Windows 地址。

```bash
go mod download
go run .
```

示例配置使用端口 `8081`。必须从项目根目录启动，以便读取 `config.yaml` 和建表 SQL。

使用 Air 时，在项目根目录执行 `air`，项目已提供 `.air.toml`。

## 接口与认证

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `/api/v1/signup` | 注册 |
| POST | `/api/v1/login` | 登录 |
| POST | `/refresh` | 刷新令牌 |
| GET | `/api/v1/community` | 社区列表 |
| GET | `/api/v1/community/:id` | 社区详情 |
| POST | `/api/v1/post` | 创建帖子 |
| GET | `/api/v1/post/:id` | 帖子详情 |
| GET | `/api/v1/posts` | 帖子列表 |
| GET | `/api/v1/posts2` | 按时间、热度或社区查询帖子 |
| POST | `/api/v1/vote` | 投票 |

受保护接口在请求头中携带：

```text
Authorization: Bearer <access_token>
```

当前登录响应字段为 `data.accessTokne`、`data.refreshToken`。刷新接口提交 JSON 字段 `access_token`、`refresh_token`。实际 token 不包含外围引号。

正常业务响应使用 JSON 中的 `code` 判断结果，`1000` 表示成功。其他响应约定请查看接口文档。

## Swagger

后端启动后访问：

```text
http://localhost:8081/swagger/index.html
```

独立文档页面和生成说明见 [docs/swagger/README.md](docs/swagger/README.md)。如果修改接口注释，需要同步更新生成文档。运行时导入的是根目录 `docs` 包，可以使用：

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.6
swag init -g main.go -o docs --exclude docs,tmp,logs,vendor
```

## 检查

```bash
go test ./...
go vet ./...
go build -o /tmp/goweb-check .
```

密码相关测试使用内存数据库驱动，不会修改真实 MySQL 数据。

## 目录

```text
controllers/   HTTP 参数校验与响应
logic/         业务逻辑
dao/mysql/     MySQL 数据访问
dao/redis/     Redis 数据访问
middlewares/   JWT 鉴权与限流
models/        请求、数据模型与建表 SQL
pkg/           JWT 与雪花 ID
settings/      配置加载
logger/        日志
routes/        路由
docs/          Swagger 文档与静态页面
```

真实 `config.yaml`、环境变量文件、日志和编译产物均通过 `.gitignore` 排除。仓库只提供不含真实凭据的 `config.example.yaml`。
