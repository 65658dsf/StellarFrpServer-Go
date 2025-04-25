CREATE TABLE IF NOT EXISTS `user_checkin_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `user_id` bigint(20) unsigned NOT NULL COMMENT '用户ID',
  `username` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户名',
  `checkin_date` date NOT NULL COMMENT '签到日期',
  `reward_traffic` bigint(20) unsigned NOT NULL COMMENT '奖励流量(字节)',
  `continuity_days` int(10) unsigned NOT NULL DEFAULT '1' COMMENT '当时连续签到天数',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_date` (`user_id`, `checkin_date`),
  KEY `idx_username` (`username`),
  KEY `idx_checkin_date` (`checkin_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户签到记录表'; 