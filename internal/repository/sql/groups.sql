CREATE TABLE IF NOT EXISTS `groups` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '组ID',
  `name` varchar(50) NOT NULL COMMENT '组名称',
  `tunnel_limit` int(11) NOT NULL DEFAULT 1 COMMENT '可创建隧道数量',
  `bandwidth_limit` int(11) NOT NULL DEFAULT 1 COMMENT '带宽限制(Mbps)',
  `traffic_quota` bigint(20) NOT NULL DEFAULT 1073741824 COMMENT '默认流量配额(GB)',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户组表';

-- 插入默认用户组
INSERT INTO `groups` (`id`, `name`, `tunnel_limit`, `bandwidth_limit`, `traffic_quota`) VALUES
(1, '免费用户', 2, 1, 30),
(2, '普通会员', 5, 5, 50),
(3, '高级会员', 10, 10, 100)
ON DUPLICATE KEY UPDATE `id`=`id`;