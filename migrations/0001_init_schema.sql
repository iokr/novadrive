-- ============================================================================
-- NovaDrive（星云盘）数据库表结构 —— 0001 初始版本
--
-- 适用范围：MySQL 5.7+ / 8.0+，字符集 utf8mb4。
-- 内容来源：与 internal/model 中的 GORM 定义逐字段核对后的完整 DDL
--           （字段类型、长度、可空性、默认值、索引名均与之保持一致）。
-- 执行方式：
--   mysql -uroot -p < migrations/0001_init_schema.sql
--   或在已选好库的客户端中直接执行本文件正文（跳过 CREATE DATABASE / USE）。
--
-- 说明：
--   1. 全部语句均为 IF NOT EXISTS，可重复执行；已有库执行不会破坏数据。
--   2. 服务启动时 internal/repository.AutoMigrate 仍会自动补全/同步结构，
--      本文件用于首次部署、DBA 审核与版本库留档。
--   3. 表之间只做逻辑关联（user_id / blob_id / file_id），不建物理外键，
--      引用完整性由服务层事务保证，避免分片与批量删除时的锁竞争。
--   4. 时间列统一为 datetime(3)，由 GORM 写入应用侧时间，故默认值为 NULL。
--   5. 后续变更请新增 0002_xxx.sql、0003_xxx.sql，不要直接改本文件。
-- ============================================================================

CREATE DATABASE IF NOT EXISTS `novadrive`
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_general_ci;

USE `novadrive`;

-- ---------------------------------------------------------------------------
-- users：用户表
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '用户主键ID',
  `username` varchar(32) NOT NULL COMMENT '用户名，登录账号，全局唯一',
  `email` varchar(128) NOT NULL COMMENT '邮箱，全局唯一，可用于登录',
  `password_hash` varchar(255) NOT NULL COMMENT '密码哈希（bcrypt）',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称，展示名称',
  `avatar` varchar(255) NOT NULL DEFAULT '' COMMENT '头像访问路径，空表示默认头像',
  `role` tinyint NOT NULL DEFAULT 0 COMMENT '角色：0-普通用户，1-管理员',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '账号状态：0-禁用，1-正常',
  `storage_used` bigint unsigned NOT NULL DEFAULT 0 COMMENT '已用逻辑存储（字节），按 files.size 累加',
  `storage_quota` bigint unsigned NOT NULL DEFAULT 10737418240 COMMENT '存储配额上限（字节），默认10GB',
  `last_login_at` datetime(3) DEFAULT NULL COMMENT '最后登录时间',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户表';

-- ---------------------------------------------------------------------------
-- blobs：物理文件表（按 MD5 全局去重，支撑秒传）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `blobs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '物理文件主键ID',
  `hash_md5` char(32) NOT NULL COMMENT '文件MD5哈希，全局唯一，用于秒传与去重',
  `size` bigint unsigned NOT NULL COMMENT '物理文件大小（字节）',
  `ext` varchar(32) NOT NULL DEFAULT '' COMMENT '主扩展名（小写，不含点）',
  `mime_type` varchar(128) NOT NULL DEFAULT '' COMMENT 'MIME类型',
  `storage_path` varchar(512) NOT NULL COMMENT '物理存储相对路径，按 hash 前两位分片',
  `thumb_path` varchar(512) NOT NULL DEFAULT '' COMMENT '缩略图相对路径，空表示无缩略图',
  `ref_count` int unsigned NOT NULL DEFAULT 1 COMMENT '引用计数，为0时物理文件可被GC清理',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_hash_md5` (`hash_md5`),
  KEY `idx_ref_count` (`ref_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='物理文件表';

-- ---------------------------------------------------------------------------
-- files：文件 / 目录表（逻辑条目，内容通过 blob_id 引用 blobs）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `files` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '文件/目录主键ID',
  `user_id` bigint unsigned NOT NULL COMMENT '所属用户ID，关联 users.id',
  `parent_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父目录ID，0表示根目录',
  `name` varchar(255) NOT NULL COMMENT '文件或目录名称（含扩展名）',
  `is_dir` tinyint NOT NULL DEFAULT 0 COMMENT '类型：0-文件，1-目录',
  `ext` varchar(32) NOT NULL DEFAULT '' COMMENT '扩展名（小写，不含点），目录为空',
  `size` bigint unsigned NOT NULL DEFAULT 0 COMMENT '文件大小（字节），目录为0',
  `mime_type` varchar(128) NOT NULL DEFAULT '' COMMENT 'MIME类型，用于前端预览判断',
  `blob_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '物理文件ID，关联 blobs.id，目录为0',
  `is_favorite` tinyint NOT NULL DEFAULT 0 COMMENT '是否收藏：0-否，1-是',
  `last_access_at` datetime(3) DEFAULT NULL COMMENT '最后访问时间，用于最近访问排序',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态：0-回收站，1-正常',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT '进入回收站的时间，NULL表示未删除',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_parent_status` (`user_id`, `parent_id`, `status`),
  KEY `idx_user_name` (`user_id`, `name`),
  KEY `idx_user_favorite` (`user_id`, `is_favorite`),
  KEY `idx_user_last_access` (`user_id`, `last_access_at`),
  KEY `idx_blob_id` (`blob_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='文件与目录表';

-- ---------------------------------------------------------------------------
-- upload_sessions：分片上传会话表（分片本体存放在存储目录，不入库）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `upload_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '上传会话主键ID',
  `upload_id` char(36) NOT NULL COMMENT '上传会话唯一标识（UUID），客户端持有',
  `user_id` bigint unsigned NOT NULL COMMENT '所属用户ID，关联 users.id',
  `parent_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '目标父目录ID，0表示根目录',
  `file_name` varchar(255) NOT NULL COMMENT '上传文件名（含扩展名）',
  `file_size` bigint unsigned NOT NULL COMMENT '文件总大小（字节）',
  `hash_md5` char(32) NOT NULL DEFAULT '' COMMENT '文件MD5，前端可选提供，用于秒传与合并校验',
  `chunk_size` int unsigned NOT NULL COMMENT '分片大小（字节），由客户端在init时提交',
  `total_chunks` int unsigned NOT NULL COMMENT '分片总数',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '会话状态：0-上传中，1-已合并，2-已取消，3-已过期',
  `chunk_dir` varchar(512) NOT NULL COMMENT '分片临时目录相对路径',
  `expire_at` datetime(3) NOT NULL COMMENT '会话过期时间，超时由GC清理',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_upload_id` (`upload_id`),
  KEY `idx_user_status` (`user_id`, `status`),
  KEY `idx_status_expire` (`status`, `expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='分片上传会话表';

-- ---------------------------------------------------------------------------
-- shares：分享表
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `shares` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '分享主键ID',
  `user_id` bigint unsigned NOT NULL COMMENT '创建分享的用户ID，关联 users.id',
  `file_id` bigint unsigned NOT NULL COMMENT '被分享的文件或目录ID，关联 files.id',
  `token` varchar(32) NOT NULL COMMENT '分享访问码，URL中唯一标识',
  `type` tinyint NOT NULL DEFAULT 0 COMMENT '分享类型：0-文件分享，1-目录分享',
  `password` varchar(255) NOT NULL DEFAULT '' COMMENT '提取密码（bcrypt哈希），空表示无密码',
  `expire_at` datetime(3) DEFAULT NULL COMMENT '过期时间，NULL表示永久有效',
  `max_downloads` bigint NOT NULL DEFAULT 0 COMMENT '最大下载次数，0表示不限制',
  `download_count` bigint NOT NULL DEFAULT 0 COMMENT '已下载次数',
  `view_count` bigint NOT NULL DEFAULT 0 COMMENT '访问次数',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态：0-已取消，1-正常，2-已过期',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token` (`token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_file_id` (`file_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='分享表';
