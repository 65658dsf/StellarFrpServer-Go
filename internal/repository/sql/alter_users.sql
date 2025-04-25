-- 修改用户表，添加签到相关字段
ALTER TABLE `users` 
ADD COLUMN `last_checkin` date DEFAULT NULL COMMENT '最后签到日期',
ADD COLUMN `checkin_count` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '签到总次数',
ADD COLUMN `continuity_checkin` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '连续签到天数';

-- 创建索引以提高查询性能
CREATE INDEX `idx_last_checkin` ON `users` (`last_checkin`); 