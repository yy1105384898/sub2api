-- Migration: 149_add_relay_monitors
-- 中转站监控：周期性抓取外部中转站（sub2api / newapi）对外公布的分组倍率，
-- 记录涨/跌变化。与 channel_monitors（自有上游账号心跳）相互独立。
--
-- 表结构说明：
--   - relay_monitors        中转站配置表（一行 = 一个被监控站点）
--   - relay_rate_snapshots  当前倍率快照（每监控每被跟踪分组一行，用于对比涨跌）
--   - relay_rate_changes    倍率变化历史（一次变化 = 一行，即涨/跌公告）
--
-- 设计要点：
--   - credential_encrypted 存放 AES-256-GCM 密文（base64），sub2api 站点必填、newapi 可空。
--   - watched_groups 用 JSONB 存字符串数组，只跟踪用户勾选的分组。
--   - 两张子表通过 ON DELETE CASCADE 自动清理已删除监控的数据。
--   - snapshots 上 (monitor_id, group_name) 唯一索引服务 upsert。
--   - changes 上 (direction, detected_at) / (detected_at) 索引服务汇总卡与历史排序。

CREATE TABLE IF NOT EXISTS relay_monitors (
    id                   BIGSERIAL PRIMARY KEY,
    name                 VARCHAR(100) NOT NULL,
    system               VARCHAR(20)  NOT NULL,    -- sub2api / newapi
    base_url             VARCHAR(500) NOT NULL,    -- base origin
    vendor               VARCHAR(50)  NOT NULL DEFAULT '',
    credential_encrypted TEXT         NOT NULL DEFAULT '',  -- AES-256-GCM (base64)，newapi 可空
    watched_groups       JSONB        NOT NULL DEFAULT '[]'::jsonb,
    enabled              BOOLEAN      NOT NULL DEFAULT TRUE,
    interval_seconds     INT          NOT NULL DEFAULT 300,
    last_checked_at      TIMESTAMPTZ,
    last_error           VARCHAR(500) NOT NULL DEFAULT '',
    created_by           BIGINT       NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT relay_monitors_system_check CHECK (system IN ('sub2api', 'newapi')),
    CONSTRAINT relay_monitors_interval_check CHECK (interval_seconds BETWEEN 60 AND 86400)
);

CREATE INDEX IF NOT EXISTS idx_relay_monitors_enabled_last_checked
    ON relay_monitors (enabled, last_checked_at);
CREATE INDEX IF NOT EXISTS idx_relay_monitors_system
    ON relay_monitors (system);

CREATE TABLE IF NOT EXISTS relay_rate_snapshots (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  BIGINT           NOT NULL REFERENCES relay_monitors(id) ON DELETE CASCADE,
    group_name  VARCHAR(200)     NOT NULL,
    rate        DOUBLE PRECISION NOT NULL,
    updated_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_relay_rate_snapshots_monitor_group
    ON relay_rate_snapshots (monitor_id, group_name);

CREATE TABLE IF NOT EXISTS relay_rate_changes (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  BIGINT           NOT NULL REFERENCES relay_monitors(id) ON DELETE CASCADE,
    site        VARCHAR(100)     NOT NULL DEFAULT '',
    system      VARCHAR(20)      NOT NULL DEFAULT '',
    vendor      VARCHAR(50)      NOT NULL DEFAULT '',
    group_name  VARCHAR(200)     NOT NULL,
    old_rate    DOUBLE PRECISION NOT NULL,
    new_rate    DOUBLE PRECISION NOT NULL,
    direction   VARCHAR(10)      NOT NULL,
    content     VARCHAR(500)     NOT NULL DEFAULT '',
    detected_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    CONSTRAINT relay_rate_changes_direction_check CHECK (direction IN ('up', 'down'))
);

CREATE INDEX IF NOT EXISTS idx_relay_rate_changes_monitor_detected
    ON relay_rate_changes (monitor_id, detected_at);
CREATE INDEX IF NOT EXISTS idx_relay_rate_changes_direction_detected
    ON relay_rate_changes (direction, detected_at);
CREATE INDEX IF NOT EXISTS idx_relay_rate_changes_detected_at
    ON relay_rate_changes (detected_at);
