CREATE TABLE IF NOT EXISTS `user_traffic_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '日志ID',
  `username` varchar(100) NOT NULL COMMENT '用户名',
  `total_traffic` bigint(20) NOT NULL DEFAULT '0' COMMENT '总消耗流量(字节)',
  `today_traffic` bigint(20) NOT NULL DEFAULT '0' COMMENT '今日流量(字节)',
  `history_traffic` text COMMENT '往期流量JSON格式',
  `traffic_quota` bigint(20) NOT NULL DEFAULT '0' COMMENT '用户总流量配额(字节)',
  `usage_percent` float NOT NULL DEFAULT '0' COMMENT '已使用流量百分比',
  `record_date` date NOT NULL COMMENT '记录日期',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username_date` (`username`, `record_date`),
  KEY `idx_record_date` (`record_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户隧道流量日志表'; 