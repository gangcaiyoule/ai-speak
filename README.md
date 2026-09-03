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
