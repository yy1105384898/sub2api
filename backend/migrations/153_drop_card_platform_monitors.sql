-- Migration: 153_drop_card_platform_monitors
-- 移除发卡平台监控功能（链动小铺接口被阿里云反爬挡死，服务端不可行，功能下线）。
-- 删除相关表；子表已对父表 ON DELETE CASCADE，这里显式按依赖顺序删并加 CASCADE 兜底。

DROP TABLE IF EXISTS card_price_events CASCADE;
DROP TABLE IF EXISTS card_product_snapshots CASCADE;
DROP TABLE IF EXISTS card_platform_monitors CASCADE;
