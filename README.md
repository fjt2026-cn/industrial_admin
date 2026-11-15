# industrial_admin

基于 Gin + GORM 的企业后台管理系统后端，MVC 分层，适配 RBAC 权限模型。

## 环境要求

- Go 1.25 及以上
- MySQL 8.0（库名 industrial_admin，字符集 utf8mb4）
- Node.js 16 及以上（仅前端）

## 启动步骤

```bash
# 1. 建库建表并初始化数据
mysql -u root -p --default-character-set=utf8mb4 < industrial_admin.sql

# 2. 配置环境变量（PowerShell 示例）
$env:DB_PASSWORD="你的口令"
$env:JWT_SECRET="换成随机长串"

# 3. 拉取依赖并启动
go mod tidy
go run main.go
```

服务默认监听 8080，健康入口为 `POST /api/v1/auth/login`。

## 目录说明

| 目录 | 作用 |
| --- | --- |
| config | 读取环境变量，产出全局配置 |
| db | GORM 连接、连接池、自动迁移 |
| model | 与库表一一对应的结构体 |
| controller | 解析参数、调用服务、封装响应 |
| service | 业务规则、事务边界、统计聚合 |
| middleware | 鉴权、权限、跨域、异常、日志 |
| routes | 路由分组与权限点挂载 |
| response | 统一响应结构与错误码 |
| utils | 口令哈希、JWT、分页、树构造、错误类型 |

## 统一响应

```json
{"code":0,"message":"ok","data":null,"timestamp":1794528000000}
```

## 安全约定

- 口令使用 bcrypt 加盐哈希存储，接口与日志均不输出
- 令牌 HS256 签名，密钥来自环境变量
- 所有查询走 GORM 参数化，排序字段白名单
- 关键操作由中间件自动留痕，写库失败不阻断业务
