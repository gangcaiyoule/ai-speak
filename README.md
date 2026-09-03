# ai-speak


最小本地运行骨架：Flutter 客户端、Go HTTP Server 和 PostgreSQL。

## 启动 PostgreSQL

```shell
docker compose up -d postgres
```

## 启动 Go Server

```shell
cd server
go run ./cmd/migrate
go run ./cmd/server
```

数据库迁移按 `server/migrations/NNNN_description.sql` 命名，并通过
`schema_migrations` 记录已执行版本。运行迁移命令前必须设置 `DATABASE_URL`。

健康检查：`http://127.0.0.1:8080/health`。

## 启动 Flutter

```shell
cd mobile
flutter pub get
flutter run
```

当前只提供接口和启动骨架，业务实现将在后续 Issue 中逐步增加。

## 身份接口

当前版本提供最小注册登录闭环，服务端使用线程安全内存仓库；重启服务会清空测试账号和会话。

```text
POST /v1/auth/register  {"email":"user@example.com","password":"password123"}
POST /v1/auth/login     {"email":"user@example.com","password":"password123"}
GET  /v1/me             Authorization: Bearer <token>
POST /v1/auth/logout    Authorization: Bearer <token>
```

密码以 Argon2id 哈希保存，会话只保存 Token 摘要。Flutter 端使用 `flutter_secure_storage` 保存当前会话 Token。
