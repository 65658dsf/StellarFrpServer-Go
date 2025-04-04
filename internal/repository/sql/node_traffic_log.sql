CREATE TABLE IF NOT EXISTS `node_traffic_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '日志ID',
  `node_name` varchar(100) NOT NULL COMMENT '节点名称',
  `traffic_in` bigint(20) NOT NULL DEFAULT '0' COMMENT '入站流量(字节)',
  `traffic_out` bigint(20) NOT NULL DEFAULT '0' COMMENT '出站流量(字节)',
  `online_count` int(11) NOT NULL DEFAULT '0' COMMENT '在线连接数',
  `record_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录时间',
  `record_date` date NOT NULL COMMENT '记录日期',
  `is_increment` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为增量记录(1:增量,0:总计)',
  PRIMARY KEY (`id`),
  KEY `idx_node_name` (`node_name`),
  KEY `idx_record_date` (`record_date`),
  KEY `idx_is_increment` (`is_increment`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='节点流量日志表'; 