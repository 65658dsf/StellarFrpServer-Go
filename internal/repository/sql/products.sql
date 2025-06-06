CREATE TABLE IF NOT EXISTS `products` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `sku_id` varchar(255) NOT NULL COMMENT '商品的唯一标识符 (SKU)',
  `name` varchar(255) NOT NULL COMMENT '商品名称',
  `description` text COMMENT '商品描述',
  `price` decimal(10, 2) NOT NULL COMMENT '价格',
  `plan_id` varchar(255) DEFAULT NULL COMMENT '爱发电的套餐ID',
  `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否上架销售',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `reward_action` varchar(255) DEFAULT NULL COMMENT '奖励动作类型 (e.g., ADD_VERIFY_COUNT)',
  `reward_value` varchar(255) DEFAULT NULL COMMENT '奖励动作的值 (e.g., 1)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `sku_id` (`sku_id`),
  KEY `idx_sku_id` (`sku_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商品表';

-- 实名认证商品
INSERT INTO `products` (`sku_id`, `name`, `description`, `price`, `plan_id`, `is_active`, `reward_action`, `reward_value`)
VALUES ('088f8490bf5711efaa1652540025c377', '实名认证次数', '购买后可增加1次实名认证机会', 1.00, '0886a92ebf5711efa5c552540025c377', 1, 'ADD_VERIFY_COUNT', '1');

-- 白银会员商品
INSERT INTO `products` (`sku_id`, `name`, `description`, `price`, `plan_id`, `is_active`, `reward_action`, `reward_value`)
VALUES ('2389400c866711efa10552540025c377', '白银会员(1个月)', '购买后升级为白银会员，享受更多权益，有效期1个月', 15.00, '2380fa96866711efa15a52540025c377', 1, 'UPGRADE_GROUP', '3'); 