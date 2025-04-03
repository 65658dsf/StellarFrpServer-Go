CREATE TABLE
  `proxy` (
    `id` int(10) NOT NULL AUTO_INCREMENT,
    `username` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户名',
    `proxy_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '隧道名称',
    `proxy_type` varchar(5) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '隧道类型',
    `local_ip` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '本地IP',
    `local_port` int(5) DEFAULT NULL COMMENT '本地端口',
    `use_encryption` varchar(5) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '是否使用加密（如yes/no）',
    `use_compression` varchar(5) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '是否使用压缩（如yes/no）',
    `domain` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '域名',
    `host_header_rewrite` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '重写主机头信息',
    `remote_port` varchar(5) COLLATE utf8mb4_unicode_ci DEFAULT '' COMMENT '远程端口号',
    `header_X-From-Where` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '' COMMENT '自定义HTTP头信息',
    `status` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '隧道状态（如active/inactive）',
    `lastupdate` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '最后更新时间',
    `node` int(10) NOT NULL COMMENT '所属节点',
    `runID` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '运行id',
    `traffic_quota` bigint(20) NOT NULL COMMENT '隧道流量',
    PRIMARY KEY (`id`) USING BTREE
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = DYNAMIC COMMENT = '隧道列表'