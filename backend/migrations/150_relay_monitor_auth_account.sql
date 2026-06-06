-- Migration: 150_relay_monitor_auth_account
-- 中转站监控：sub2api 目标站只能用「邮箱+密码」登录（别人的站不开放 API 密钥），
-- 探测时先登录拿 JWT 再抓分组。新增 auth_account 列存登录邮箱；
-- credential_encrypted 改存登录密码（加密）。

ALTER TABLE relay_monitors
    ADD COLUMN IF NOT EXISTS auth_account VARCHAR(200) NOT NULL DEFAULT '';
