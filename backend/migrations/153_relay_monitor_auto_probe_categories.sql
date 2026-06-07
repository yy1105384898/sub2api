-- Migration: 153_relay_monitor_auto_probe_categories
-- 中转站监控：按模型类型自动探测并纳入新增分组。

ALTER TABLE relay_monitors
    ADD COLUMN IF NOT EXISTS auto_probe_categories JSONB NOT NULL DEFAULT '[]'::jsonb;
