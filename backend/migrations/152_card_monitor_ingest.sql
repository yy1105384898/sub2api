-- Migration: 152_card_monitor_ingest
-- 发卡平台监控：新增「浏览器推送」模式。
-- 链动小铺等站点接口被阿里云 ESA(acw_sc__v2) 反爬挑战保护，服务器直连拿到的是
-- 挑战网页而非 JSON，无法服务端抓取。改由浏览器油猴脚本(天然过挑战)抓取后用
-- ingest_key 推送到本服务。本迁移加推送密钥列与 'push' 认证模式。

ALTER TABLE card_platform_monitors
    ADD COLUMN IF NOT EXISTS ingest_key VARCHAR(64) NOT NULL DEFAULT '';

-- 放宽 auth_mode 约束，加入 push。
ALTER TABLE card_platform_monitors
    DROP CONSTRAINT IF EXISTS card_platform_monitors_auth_mode_check;
ALTER TABLE card_platform_monitors
    ADD CONSTRAINT card_platform_monitors_auth_mode_check
    CHECK (auth_mode IN ('public', 'token', 'cookie', 'push'));

-- 给既有行补一个随机推送密钥(md5 无需 pgcrypto 扩展)。
UPDATE card_platform_monitors
SET ingest_key = md5(random()::text || clock_timestamp()::text || id::text)
WHERE ingest_key = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_card_platform_monitors_ingest_key
    ON card_platform_monitors (ingest_key) WHERE ingest_key <> '';
