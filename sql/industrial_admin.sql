-- =====================================================================
-- 工业软件企业后台管理系统（数字化运营）· MySQL 8.0 建库建表脚本
-- 数据库名：industrial_admin
-- 字符集：utf8mb4    排序规则：utf8mb4_0900_ai_ci    存储引擎：InnoDB
-- 适用：实训阶段本地/服务器部署，可重复执行（先按依赖逆序删表再重建）
-- 说明：所有表均含时间戳；业务表使用 deleted_at 实现软删除；
--       口令仅存 bcrypt 加盐哈希，不存明文。
-- =====================================================================

SET NAMES utf8mb4;

CREATE DATABASE IF NOT EXISTS industrial_admin
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_0900_ai_ci;

USE industrial_admin;

-- ---------------------------------------------------------------------
-- 0. 清理（按外键依赖逆序删除，便于反复执行；仅影响本库同名表）
-- ---------------------------------------------------------------------
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS operation_log;
DROP TABLE IF EXISTS sys_config;
DROP TABLE IF EXISTS role_permission;
DROP TABLE IF EXISTS user_role;
DROP TABLE IF EXISTS permission;
DROP TABLE IF EXISTS `role`;
DROP TABLE IF EXISTS `user`;
DROP TABLE IF EXISTS dept;

SET FOREIGN_KEY_CHECKS = 1;

-- ---------------------------------------------------------------------
-- 1. 部门表 dept（自关联树，根部门 parent_id 为 NULL）
-- ---------------------------------------------------------------------
CREATE TABLE dept (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门主键',
  parent_id  BIGINT UNSIGNED NULL DEFAULT NULL     COMMENT '上级部门ID，NULL 表示根部门',
  dept_name  VARCHAR(64) NOT NULL                  COMMENT '部门名称，同级唯一（由业务校验）',
  dept_code  VARCHAR(64) NOT NULL                  COMMENT '部门编码，全局唯一',
  leader_id  BIGINT UNSIGNED NULL DEFAULT NULL     COMMENT '部门负责人，逻辑指向 user.id',
  sort_no    INT NOT NULL DEFAULT 0                COMMENT '同级排序号，升序展示',
  status     TINYINT NOT NULL DEFAULT 1            COMMENT '状态：1 启用，0 停用',
  remark     VARCHAR(255) NULL DEFAULT NULL        COMMENT '备注',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL DEFAULT NULL            COMMENT '软删除时间，NULL 表示未删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_dept_code (dept_code),
  KEY idx_dept_parent (parent_id),
  KEY idx_dept_status (status),
  CONSTRAINT fk_dept_parent FOREIGN KEY (parent_id) REFERENCES dept (id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '部门表：组织架构树';

-- ---------------------------------------------------------------------
-- 2. 用户表 user
-- ---------------------------------------------------------------------
CREATE TABLE `user` (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户主键',
  username         VARCHAR(50) NOT NULL            COMMENT '登录账号，全局唯一',
  password_hash    VARCHAR(100) NOT NULL           COMMENT 'bcrypt 加盐哈希（60 字符），禁止存明文',
  real_name        VARCHAR(50) NOT NULL            COMMENT '姓名',
  email            VARCHAR(100) NULL DEFAULT NULL  COMMENT '邮箱',
  phone            VARCHAR(20) NULL DEFAULT NULL   COMMENT '手机号',
  dept_id          BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '所属部门ID',
  status           TINYINT NOT NULL DEFAULT 1      COMMENT '状态：1 启用，0 停用',
  login_fail_count INT NOT NULL DEFAULT 0          COMMENT '连续登录失败次数',
  lock_until       DATETIME NULL DEFAULT NULL      COMMENT '锁定截止时间，NULL 表示未锁定',
  last_login_at    DATETIME NULL DEFAULT NULL      COMMENT '最近登录时间',
  last_login_ip    VARCHAR(45) NULL DEFAULT NULL   COMMENT '最近登录IP（兼容 IPv6）',
  remark           VARCHAR(255) NULL DEFAULT NULL  COMMENT '备注',
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at       DATETIME NULL DEFAULT NULL      COMMENT '软删除时间，NULL 表示未删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_username (username),
  KEY idx_user_dept (dept_id),
  KEY idx_user_status (status),
  KEY idx_user_created (created_at),
  KEY idx_user_last_login (last_login_at),
  CONSTRAINT fk_user_dept FOREIGN KEY (dept_id) REFERENCES dept (id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '用户表：账号与身份载体';

-- ---------------------------------------------------------------------
-- 3. 角色表 role（注意：role 是 MySQL 8.0 保留字，必须加反引号）
-- ---------------------------------------------------------------------
CREATE TABLE `role` (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '角色主键',
  role_name  VARCHAR(50) NOT NULL                  COMMENT '角色名称',
  role_code  VARCHAR(50) NOT NULL                  COMMENT '角色编码，全局唯一',
  data_scope TINYINT NOT NULL DEFAULT 1            COMMENT '数据范围：1 全部，2 本部门，3 仅本人',
  is_builtin TINYINT NOT NULL DEFAULT 0            COMMENT '是否内置角色：1 是（不可删除），0 否',
  sort_no    INT NOT NULL DEFAULT 0                COMMENT '排序号',
  status     TINYINT NOT NULL DEFAULT 1            COMMENT '状态：1 启用，0 停用',
  remark     VARCHAR(255) NULL DEFAULT NULL        COMMENT '备注',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL DEFAULT NULL            COMMENT '软删除时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_role_code (role_code),
  UNIQUE KEY uk_role_name (role_name),
  KEY idx_role_status (status)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '角色表：授权管理单元';

-- ---------------------------------------------------------------------
-- 4. 权限表 permission（菜单/按钮/接口 三类权限点，树形）
-- ---------------------------------------------------------------------
CREATE TABLE permission (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '权限点主键',
  parent_id   BIGINT UNSIGNED NULL DEFAULT NULL    COMMENT '上级权限点ID，NULL 表示顶层',
  perm_name   VARCHAR(64) NOT NULL                 COMMENT '权限点名称',
  perm_code   VARCHAR(100) NOT NULL                COMMENT '权限点编码（模块:动作），全局唯一',
  perm_type   TINYINT NOT NULL DEFAULT 1           COMMENT '类型：1 菜单，2 按钮，3 接口',
  route_path  VARCHAR(200) NULL DEFAULT NULL       COMMENT '前端路由路径（菜单类）',
  api_path    VARCHAR(200) NULL DEFAULT NULL       COMMENT '后端接口路径（接口类）',
  http_method VARCHAR(10) NULL DEFAULT NULL        COMMENT 'HTTP 方法：GET/POST/PUT/DELETE',
  icon        VARCHAR(64) NULL DEFAULT NULL        COMMENT '菜单图标',
  sort_no     INT NOT NULL DEFAULT 0               COMMENT '排序号',
  status      TINYINT NOT NULL DEFAULT 1           COMMENT '状态：1 启用，0 停用',
  remark      VARCHAR(255) NULL DEFAULT NULL       COMMENT '备注',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at  DATETIME NULL DEFAULT NULL           COMMENT '软删除时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_perm_code (perm_code),
  KEY idx_perm_parent (parent_id),
  KEY idx_perm_type (perm_type),
  CONSTRAINT fk_perm_parent FOREIGN KEY (parent_id) REFERENCES permission (id)
    ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '权限点表：RBAC 授权最小粒度';

-- ---------------------------------------------------------------------
-- 5. 用户角色关联表 user_role（多对多）
-- ---------------------------------------------------------------------
CREATE TABLE user_role (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id    BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  role_id    BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授予时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_role (user_id, role_id),
  KEY idx_ur_role (role_id),
  CONSTRAINT fk_ur_user FOREIGN KEY (user_id) REFERENCES `user` (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_ur_role FOREIGN KEY (role_id) REFERENCES `role` (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '用户角色关联表';

-- ---------------------------------------------------------------------
-- 6. 角色权限关联表 role_permission（多对多，RBAC 的关键一环）
-- ---------------------------------------------------------------------
CREATE TABLE role_permission (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  role_id    BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  perm_id    BIGINT UNSIGNED NOT NULL COMMENT '权限点ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授权时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_role_perm (role_id, perm_id),
  KEY idx_rp_perm (perm_id),
  CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES `role` (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_rp_perm FOREIGN KEY (perm_id) REFERENCES permission (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '角色权限关联表';

-- ---------------------------------------------------------------------
-- 7. 系统配置表 sys_config
-- ---------------------------------------------------------------------
CREATE TABLE sys_config (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  config_key   VARCHAR(100) NOT NULL               COMMENT '参数键（分类.名称），全局唯一',
  config_name  VARCHAR(100) NOT NULL               COMMENT '参数名称',
  config_value VARCHAR(2000) NULL DEFAULT NULL     COMMENT '参数值，按 value_type 校验格式',
  value_type   TINYINT NOT NULL DEFAULT 1          COMMENT '值类型：1 字符串，2 整数，3 布尔，4 JSON',
  category     VARCHAR(32) NOT NULL DEFAULT 'base' COMMENT '分类：base/security/business/dashboard',
  is_system    TINYINT NOT NULL DEFAULT 0          COMMENT '是否系统内置：1 是',
  status       TINYINT NOT NULL DEFAULT 1          COMMENT '状态：1 启用，0 停用',
  remark       VARCHAR(255) NULL DEFAULT NULL      COMMENT '说明',
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at   DATETIME NULL DEFAULT NULL          COMMENT '软删除时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_config_key (config_key),
  KEY idx_config_category (category),
  KEY idx_config_status (status)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '系统参数配置表';

-- ---------------------------------------------------------------------
-- 8. 操作日志表 operation_log（仅追加，不设软删除与更新字段）
-- ---------------------------------------------------------------------
CREATE TABLE operation_log (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id    BIGINT UNSIGNED NULL DEFAULT NULL     COMMENT '操作人ID，仅记录不设外键',
  username   VARCHAR(50) NULL DEFAULT NULL         COMMENT '操作账号快照，用户删除后仍可追溯',
  module     VARCHAR(50) NOT NULL                  COMMENT '业务模块：用户/角色/部门/配置/看板/登录',
  action     VARCHAR(50) NOT NULL                  COMMENT '操作动作：新增/修改/删除/授权/登录等',
  method     VARCHAR(10) NULL DEFAULT NULL         COMMENT 'HTTP 方法',
  url        VARCHAR(255) NULL DEFAULT NULL        COMMENT '请求地址',
  ip         VARCHAR(45) NULL DEFAULT NULL         COMMENT '客户端IP',
  param      TEXT NULL DEFAULT NULL                COMMENT '关键参数（白名单、已脱敏）',
  result     TINYINT NOT NULL DEFAULT 1            COMMENT '结果：1 成功，0 失败',
  error_msg  VARCHAR(500) NULL DEFAULT NULL        COMMENT '失败原因摘要',
  cost_ms    INT NOT NULL DEFAULT 0                COMMENT '接口耗时（毫秒）',
  user_agent VARCHAR(255) NULL DEFAULT NULL        COMMENT '客户端标识',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
  PRIMARY KEY (id),
  KEY idx_log_created (created_at),
  KEY idx_log_user (user_id),
  KEY idx_log_module_time (module, created_at),
  KEY idx_log_result (result)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci
  COMMENT = '操作日志表：审计留痕';

-- ---------------------------------------------------------------------
-- 9. 补充循环依赖外键：部门负责人 → 用户（在 user 建成后追加）
-- ---------------------------------------------------------------------
ALTER TABLE dept
  ADD CONSTRAINT fk_dept_leader FOREIGN KEY (leader_id) REFERENCES `user` (id)
  ON DELETE SET NULL ON UPDATE CASCADE;

-- =====================================================================
-- 初始化测试数据
-- 说明：口令均为 bcrypt(rounds=10) 加盐哈希，salt 已内嵌在哈希串中。
--   种子账号 admin（超级管理员）、test01（普通员工），初始口令见部署说明，交付前应修改。
-- =====================================================================

-- 9.1 部门（三级结构：总公司 → 技术部 → 研发一组）
INSERT INTO dept (id, parent_id, dept_name, dept_code, leader_id, sort_no, status, remark) VALUES
(1, NULL, '总公司',   'HQ',        NULL, 1, 1, '根部门，不可删除'),
(2, 1,    '技术部',   'TECH',      NULL, 2, 1, '负责平台研发与运维'),
(3, 1,    '运营部',   'OPS',       NULL, 3, 1, '负责用户运营与数据分析'),
(4, 1,    '审计部',   'AUDIT',     NULL, 4, 1, '负责操作审计与合规检查'),
(5, 2,    '研发一组', 'TECH-RD1',  NULL, 5, 1, '负责后端与前端开发');

-- 9.2 用户
INSERT INTO `user` (id, username, password_hash, real_name, email, phone, dept_id, status, remark) VALUES
(1, 'admin',  '$2b$10$tffW2ASo4EysCyb78FjhA.amKGd2eLuWv/olRJ6ZWuJds7koBUX2S',
    '系统管理员', 'admin@example.com',  '13800000001', 1, 1, '内置超级管理员账号'),
(2, 'test01', '$2b$10$aRk6Qv6LYYm9nfB0o5Rfi.H0ntAb3NZg1Rs8e2ju5yQDJs/mvPXvy',
    '测试员工', 'test01@example.com', '13800000002', 3, 1, '普通员工测试账号');

-- 9.3 角色
INSERT INTO `role` (id, role_name, role_code, data_scope, is_builtin, sort_no, status, remark) VALUES
(1, '超级管理员', 'super_admin',   1, 1, 1, 1, '内置角色，拥有全部权限，不可删除'),
(2, '系统管理员', 'sys_admin',     1, 0, 2, 1, '负责用户、部门、参数日常维护'),
(3, '部门主管',   'dept_manager',  2, 0, 3, 1, '管辖本部门成员，数据范围为本部门'),
(4, '运营专员',   'operator',      1, 0, 4, 1, '查看运营指标'),
(5, '审计员',     'auditor',       1, 0, 5, 1, '查询操作日志，追溯变更'),
(6, '普通员工',   'employee',      3, 0, 6, 1, '默认角色，仅本人数据');

-- 9.4 权限点（菜单 / 按钮 / 接口）
INSERT INTO permission (id, parent_id, perm_name, perm_code, perm_type, route_path, api_path, http_method, sort_no) VALUES
(1,  NULL, '系统管理',     'sys',              1, '/system',          NULL, NULL, 1),
(2,  1,    '用户管理',     'sys:user',         1, '/system/user',     NULL, NULL, 1),
(3,  1,    '角色管理',     'sys:role',         1, '/system/role',     NULL, NULL, 2),
(4,  1,    '部门管理',     'sys:dept',         1, '/system/dept',     NULL, NULL, 3),
(5,  1,    '参数配置',     'sys:config',       1, '/system/config',   NULL, NULL, 4),
(6,  NULL, '数据运营',     'data',             1, '/dashboard',       NULL, NULL, 2),
(7,  6,    '运营看板',     'data:dashboard',   1, '/dashboard/home',  NULL, NULL, 1),
(8,  NULL, '日志审计',     'audit',            1, '/audit',           NULL, NULL, 3),
(9,  8,    '操作日志',     'audit:log',        1, '/audit/log',       NULL, NULL, 1),
(10, 2,    '用户新增',     'user:create',      2, NULL, NULL, NULL, 1),
(11, 2,    '用户编辑',     'user:update',      2, NULL, NULL, NULL, 2),
(12, 2,    '用户删除',     'user:delete',      2, NULL, NULL, NULL, 3),
(13, 2,    '口令重置',     'user:resetpwd',    2, NULL, NULL, NULL, 4),
(14, 3,    '角色授权',     'role:grant',       2, NULL, NULL, NULL, 1),
(15, 3,    '权限点维护',   'perm:edit',        2, NULL, NULL, NULL, 2),
(16, 4,    '部门新增',     'dept:create',      2, NULL, NULL, NULL, 1),
(17, 4,    '部门删除',     'dept:delete',      2, NULL, NULL, NULL, 2),
(18, 5,    '参数修改',     'config:update',    2, NULL, NULL, NULL, 1),
(19, NULL, '个人中心',     'profile',          1, '/profile',        NULL, NULL, 4),
(20, 2,    '用户列表接口', 'api:user:list',    3, NULL, '/api/v1/users',  'GET',    1),
(21, NULL, '登录接口',     'api:auth:login',   3, NULL, '/api/v1/auth/login', 'POST', 1);

-- 9.5 用户角色关联
INSERT INTO user_role (user_id, role_id) VALUES
(1, 1),
(2, 6);

-- 9.6 角色权限关联（超管拥有全部权限，其余按职责授予）
INSERT INTO role_permission (role_id, perm_id)
SELECT 1, id FROM permission WHERE deleted_at IS NULL;

INSERT INTO role_permission (role_id, perm_id) VALUES
(2, 1), (2, 2), (2, 3), (2, 4), (2, 5), (2, 6), (2, 7), (2, 8), (2, 9), (2, 19),
(2, 10), (2, 11), (2, 12), (2, 13), (2, 14), (2, 16), (2, 18), (2, 20),
(3, 6), (3, 7), (3, 8), (3, 9), (3, 19), (3, 16),
(4, 6), (4, 7), (4, 19),
(5, 8), (5, 9), (5, 19),
(6, 19);

-- 9.7 系统参数
INSERT INTO sys_config (config_key, config_name, config_value, value_type, category, is_system, status, remark) VALUES
('base.system.name',           '系统名称',             '工业软件企业后台管理系统', 1, 'base',      1, 1, '展示在登录页与首页顶部'),
('base.page.size.default',     '列表默认每页条数',     '20',  2, 'base',      1, 1, '分页默认值，上限 50'),
('base.dept.max.depth',        '部门最大层级',         '4',   2, 'base',      1, 1, '防止部门树过深'),
('security.password.min.length','口令最小长度',        '8',   2, 'security',  1, 1, '同时要求包含字母与数字'),
('security.login.max.fail',    '登录失败锁定次数',     '5',   2, 'security',  1, 1, '连续失败达到该值锁定账号'),
('security.login.lock.minutes','锁定时长（分钟）',     '10',  2, 'security',  1, 1, '锁定期内禁止登录'),
('security.token.expire.hours','令牌有效期（小时）',   '2',   2, 'security',  1, 1, 'JWT 过期时间'),
('business.log.keep.days',     '操作日志保留天数',     '180', 2, 'business',  1, 1, '超期由清理脚本处理'),
('dashboard.default.days',     '看板默认时间范围（天）','7',  2, 'dashboard', 1, 1, '可切换为 30 天'),
('dashboard.enable.cache',     '看板启用缓存',         'true', 3, 'dashboard', 0, 1, '布尔值，演示环境可关闭');

-- 9.8 回填部门负责人
UPDATE dept SET leader_id = 1 WHERE id = 2;
UPDATE dept SET leader_id = 2 WHERE id = 3;

-- =====================================================================
-- 执行后自检（可选，逐条核对初始化结果）
-- =====================================================================
-- SELECT COUNT(*) FROM dept;              -- 期望 5
-- SELECT COUNT(*) FROM `user`;            -- 期望 2
-- SELECT COUNT(*) FROM `role`;            -- 期望 6
-- SELECT COUNT(*) FROM permission;        -- 期望 21
-- SELECT COUNT(*) FROM user_role;         -- 期望 2
-- SELECT COUNT(*) FROM role_permission;   -- 期望 21 + 18 + 6 + 3 + 3 + 1 = 52
-- SELECT COUNT(*) FROM sys_config;        -- 期望 10
-- 查看某用户的有效权限（多角色取并集）
-- SELECT u.username, r.role_name, p.perm_code, p.perm_name
--   FROM `user` u
--   JOIN user_role ur ON ur.user_id = u.id
--   JOIN `role` r     ON r.id = ur.role_id
--   JOIN role_permission rp ON rp.role_id = r.id
--   JOIN permission p ON p.id = rp.perm_id
--  WHERE u.username = 'admin' AND p.deleted_at IS NULL
--  ORDER BY p.id;
