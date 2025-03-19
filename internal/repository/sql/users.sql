CREATE TABLE
  `users` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '用户ID',
    `username` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户名',
    `password` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '密码（加密后）',
    `email` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '邮箱',
    `register_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
    `group_id` bigint(20) unsigned NOT NULL DEFAULT '1' COMMENT '权限组ID',
    `is_verified` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否实名认证（0：未认证，1：已认证）',
    `verify_info` text COLLATE utf8mb4_unicode_ci COMMENT '实名认证信息（加密后）',
    `verify_count` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '实名认证尝试次数',
    `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '用户状态（0：禁用，1：正常）',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `group_time` timestamp NULL DEFAULT NULL COMMENT '权限组时间',
    `token` text COLLATE utf8mb4_unicode_ci,
    PRIMARY KEY (`id`),
    UNIQUE KEY `username` (`username`),
    UNIQUE KEY `email` (`email`),
    KEY `idx_username` (`username`),
    KEY `idx_email` (`email`),
    KEY `idx_group_id` (`group_id`),
    KEY `idx_status` (`status`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '用户表'