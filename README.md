# NovaDrive · 星云盘

基于 Go 的私有云盘服务，单二进制同时对外提供 REST API 与前端 SPA 静态资源。
支持大文件分片上传、断点续传、MD5 秒传去重、在线预览、目录 / 批量 zip 下载、
分享（提取码 + 访问签名）、收藏 / 最近访问 / 回收站与管理员后台。

## 技术栈

| 层 | 选型 |
|---|---|
| Web 框架 | Gin v1.10 |
| ORM | GORM v2 + MySQL 8 |
| 配置 | Viper（YAML 为唯一配置源） |
| 认证 | golang-jwt/v5（HS256）+ bcrypt |
| 日志 | zap（控制台 + 文件双输出） |
| 前端 | Vue 3.5 + Vite 5 + Pinia + Vue Router 4 + Element Plus 2.8（深/浅色双主题） |
| 前端托管 | `embed` 嵌入 `web/dist`，history 路由回退 |
| 图片处理 | `image` + `golang.org/x/image`（缩放、webp/bmp 解码） |

## 核心设计

- **逻辑文件与物理文件分离**：`files` 保存用户可见条目，`blobs` 按 `hash_md5` 全局唯一，
  通过 `ref_count` 引用计数支撑秒传、复制零拷贝与安全 GC。
- **Blob 落盘路径**：`{storage.local.root}/{md5[0:2]}/{md5}`（无扩展名，扩展名与 MIME 存库）。
- **分片上传**：分片不入库，落盘 `{storage.local.upload_root}/{upload_id}/{index}.part`，
  先写 `.part.tmp` 再原子重命名，保证重传幂等；合并时流式计算 MD5 并校验。
- **统一响应**：HTTP 状态码恒为 200，业务状态由 `code` 表达（`0` 成功，其余见下表）。
- **回收站**：删除仅递归置 `status=0` 并写 `deleted_at`，不改变 `ref_count`；
  彻底删除 / 清空回收站才在事务内扣减引用并回收配额。
- **分享**：`token` 随机生成；提取码 bcrypt 存储；
  校验通过后返回 `access_key = HMAC-SHA256(secret, "share:"+token)`，后续请求带 `X-Share-Key` 头。

## 快速开始

```bash
# 1. 初始化数据库（完整表结构见 migrations/0001_init_schema.sql，服务启动时也会 AutoMigrate 补全）
mysql -uroot -p < migrations/0001_init_schema.sql

# 2. 修改配置（数据库账号、jwt.secret、bootstrap 管理员等）
vi config/config.yaml

# 3. 运行
go run ./cmd/novadrive -c config/config.yaml
```

默认监听 `http://localhost:8080`，首次启动会按 `bootstrap` 配置创建管理员
（默认 `admin / admin123`，**请首次登录后立即修改密码**）。

健康检查：`GET /api/v1/health`。

### 前端开发

```bash
cd frontend
npm install
npm run dev           # http://localhost:5173，/api 已代理到 http://localhost:8080
```

页面与路由职责见下文「前端页面」。

### 编译与部署

```bash
# 1. 构建前端（产物落到 web/dist，由 go:embed 内嵌）
cd frontend && npm ci && npm run build && cd ..

# 2. 编译后端单二进制
make build            # 产物 bin/novadrive

# 或一键完成前后端（等价于 make web + make build）
make build-all

# 交叉编译（Linux 服务器，会先构建前端）
GOOS=linux GOARCH=amd64 scripts/build.sh
```

`go:embed all:dist` 要求 `web/dist` 目录存在；未构建前端时目录为空，
访问 `/` 会返回提示页面、API 仍可正常使用，按上述步骤构建后重新编译即可随二进制一起发布。

## 配置说明（config/config.yaml）

| 配置段 | 关键项 | 说明 |
|---|---|---|
| `server` | `port`、`mode`、`base_url` | `mode=debug` 开启 Gin 调试与 GORM 明细日志；`base_url` 用于拼接分享链接 |
| `database` | `host/port/user/password/name` | MySQL 连接与连接池 |
| `jwt` | `secret`、`expire_hours` | JWT 密钥与有效期 |
| `storage.local` | `root`、`thumb_root`、`upload_root` | 物理文件 / 缩略图 / 分片目录 |
| `storage.upload` | `max_file_size`、`max_batch_*`、`allowed_ext` | 上传限制与扩展名白名单 |
| `storage.chunk` | `threshold`、`default_chunk_size`、`max_chunk_size` | 分片参数 |
| `storage.session` | `expire_hours`、`gc_interval_minutes` | 上传会话有效期与 GC 周期 |
| `download` | `zip_buffer_size`、`zip_max_entries` | zip 流缓冲与条目上限 |
| `thumbnail` | `max_width/max_height/quality` | 缩略图尺寸与 JPEG 质量 |
| `share` | `token_length`、`default_expire_days` | 分享码长度与默认有效期 |
| `blob_gc` | `enabled`、`interval_minutes` | 零引用物理文件回收 |
| `bootstrap` | `admin_username/admin_email/admin_password` | 初始管理员 |
| `log` | `level`、`path` | 日志级别与文件目录 |

## 接口一览（前缀 `/api/v1`）

**认证**

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/register` | 注册 |
| POST | `/auth/login` | 登录，返回 token |
| GET / PUT | `/auth/profile` | 当前用户信息 / 修改昵称头像 |
| PUT | `/auth/password` | 修改密码 |

**文件**

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/files?parent_id=&keyword=&sort=&order=` | 列表 / 搜索 |
| GET | `/files/breadcrumb`、`/files/recent`、`/files/favorites`、`/files/stats` | 面包屑 / 最近 / 收藏 / 空间统计 |
| POST | `/files/mkdir` | 新建目录 |
| POST | `/files/upload`、`/files/upload/batch` | 单文件 / 多文件与文件夹上传 |
| POST | `/files/check` | 秒传预检 |
| POST | `/files/upload/init` | 分片上传初始化（命中秒传直接入库） |
| PUT | `/files/upload/:upload_id/chunk?index=N` | 上传分片 |
| GET | `/files/upload/:upload_id/status` | 查询进度（断点续传） |
| POST | `/files/upload/:upload_id/complete` | 合并入库 |
| DELETE | `/files/upload/:upload_id` | 取消会话 |
| GET | `/files/:id` | 详情（刷新最近访问） |
| GET | `/files/:id/thumb`、`/files/:id/raw` | 缩略图 / 内联预览（Range、ETag） |
| PUT | `/files/:id/rename`、`/files/:id/move`、`/files/:id/favorite` | 重命名 / 移动 / 收藏 |
| POST | `/files/:id/copy` | 复制（目录递归） |
| GET | `/files/:id/download`、`POST /files/download/batch` | 单条 / 批量下载（目录与批量走 zip 流） |
| DELETE | `/files/:id` | 移入回收站 |
| GET / DELETE | `/files/recycle` | 回收站列表 / 清空 |
| POST | `/files/:id/restore`、`DELETE /files/:id/force` | 还原 / 彻底删除 |

**分享**

| 方法 | 路径 | 说明 |
|---|---|---|
| POST / GET | `/shares` | 创建分享 / 我的分享 |
| DELETE | `/shares/:id` | 取消分享 |
| GET | `/shares/public/:token` | 分享基本信息 |
| POST | `/shares/public/:token/verify` | 校验提取码，返回 `access_key` |
| GET | `/shares/public/:token/files` | 浏览分享目录 |
| GET | `/shares/public/:token/preview/:file_id` | 预览 |
| GET | `/shares/public/:token/download?file_id=` | 下载（目录自动打包 zip） |

**管理员（`role=1`）**

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/admin/users` | 用户列表 |
| PUT | `/admin/users/:id/status`、`/admin/users/:id/quota` | 启用禁用 / 调整配额 |
| GET | `/admin/stats` | 系统统计 |
| GET | `/admin/upload-sessions` | 上传会话监控 |

## 前端页面

| 路由 | 页面 | 主要能力 |
|---|---|---|
| `/` | 我的文件 | 目录浏览、上传 / 新建目录、排序与视图切换、批量下载与删除、在线预览 |
| `/favorites` | 我的收藏 | 收藏列表、批量下载、取消收藏 |
| `/recent` | 最近访问 | 最近 50 条访问记录 |
| `/recycle` | 回收站 | 还原、彻底删除、清空回收站 |
| `/shares` | 我的分享 | 创建分享、状态筛选、二维码与提取码、取消分享 |
| `/profile` | 个人中心 | 空间分析、资料与头像、密码修改 |
| `/admin`、`/admin/users` | 管理后台 | 系统统计、上传会话监控、用户启停与配额（`role=1`） |
| `/login`、`/register` | 登录 / 注册 | 账号密码登录、注册后自动登录 |
| `/s/:token` | 分享访问页 | 提取码校验、目录浏览、在线预览与下载 |

在线预览覆盖图片（缩放 / 切换）、PDF（连续滚动与缩放）、视频 / 音频（播放与倍速）、
文本（行号与换行切换），其余类型先提示再下载；全局搜索支持 `Ctrl + K` 唤起。

## 业务错误码

| code | 含义 |
|---|---|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 未登录或 token 失效 |
| 1003 | 无权限 |
| 1004 | 非管理员 |
| 2001 | 用户名 / 邮箱已存在 |
| 2002 | 用户名或密码错误 |
| 3001 | 文件不存在 |
| 3002 | 存储空间不足 |
| 3003 | 同名文件已存在 |
| 3004 | 上传数量或大小超限 |
| 3005 | 压缩包条目数超限 |
| 3006 | 上传会话不存在或已失效 |
| 3007 | 分片索引越界或分片大小不匹配 |
| 3008 | 分片不完整，无法合并 |
| 3009 | 合并后 MD5 与声明不一致 |
| 4001 | 分享不存在或已失效 |
| 4002 | 提取密码错误 |
| 5000 | 服务器内部错误 |

## 目录结构

```
cmd/novadrive/        服务入口（配置 → 日志 → DB → 依赖装配 → 优雅退出）
config/               config.yaml 示例配置
internal/
  config/             配置加载与校验
  constants/          枚举、限制与固定取值
  model/              GORM 模型（users / blobs / files / upload_sessions / shares）
  repository/         数据访问层
  storage/            物理存储抽象与本地实现（blob / 分片 / 缩略图）
  service/            业务层（认证、文件、上传、分享、缩略图、管理后台）
  dto/                接口出入参
  handler/            HTTP 处理器与流式响应
  middleware/         认证、权限、CORS、日志、异常恢复
  router/             路由装配与 SPA 托管
  jobs/               上传会话 GC、物理文件 GC
pkg/                  通用工具（resp / errs / jwt / hash / random / fileutil / ziputil / logger）
scripts/              构建与发布脚本
migrations/           数据库建表 / 迁移脚本（0001_init_schema.sql 为完整表结构）
frontend/             Vue 3 前端工程（构建产物输出到 web/dist）
web/                  前端产物目录与 embed 声明
```

## 运维要点

- 定期备份 `data/blobs` 与 `data/thumbs`，以及 MySQL 库；删除前请先备份。
- 物理文件仅在 `ref_count = 0` 时由 GC 回收（默认每 30 分钟一轮），
  顺序为「条件删除记录 → 删除磁盘文件」，避免并发秒传复活后被误删。
- 分片会话默认 24 小时过期，GC 会清理分片目录并置会话为已过期。
- 生产环境务必修改 `jwt.secret` 与 `bootstrap` 默认密码，并把 `server.mode` 设为 `release`。
