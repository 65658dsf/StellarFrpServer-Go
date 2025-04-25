-- 修改用户组表，添加签到相关字段
ALTER TABLE `groups` 
ADD COLUMN `checkin_min_traffic` bigint(20) NOT NULL DEFAULT '1073741824' COMMENT '签到最小流量(字节，默认1GB)',
ADD COLUMN `checkin_max_traffic` bigint(20) NOT NULL DEFAULT '3221225472' COMMENT '签到最大流量(字节，默认3GB)';

-- 更新默认权限组的签到流量范围
UPDATE `groups` SET 
  `checkin_min_traffic` = 1073741824,  -- 1GB = 1024*1024*1024 字节
  `checkin_max_traffic` = 3221225472   -- 3GB = 3*1024*1024*1024 字节
WHERE id = 1;  -- 未实名/普通用户组

-- 假设id=2是VIP组
UPDATE `groups` SET 
  `checkin_min_traffic` = 3221225472,  -- 3GB
  `checkin_max_traffic` = 10737418240  -- 10GB
WHERE id = 2;

-- 假设id=3是SVIP组
UPDATE `groups` SET 
  `checkin_min_traffic` = 5368709120,  -- 5GB
  `checkin_max_traffic` = 16106127360  -- 15GB
WHERE id = 3; 