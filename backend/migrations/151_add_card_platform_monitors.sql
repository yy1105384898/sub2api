-- Migration: 151_add_card_platform_monitors
-- 发卡平台监控：定时扫描外部发卡站商品，保存当前快照并记录价格/库存/上下架变化。

CREATE TABLE IF NOT EXISTS card_platform_monitors (
    id                   BIGSERIAL PRIMARY KEY,
    name                 VARCHAR(100) NOT NULL,
    platform_type        VARCHAR(30)  NOT NULL DEFAULT 'ldxp',
    base_url             VARCHAR(500) NOT NULL,
    shop_url             VARCHAR(500) NOT NULL DEFAULT '',
    auth_mode            VARCHAR(20)  NOT NULL DEFAULT 'token',
    credential_encrypted TEXT         NOT NULL DEFAULT '',
    enabled              BOOLEAN      NOT NULL DEFAULT TRUE,
    interval_seconds     INT          NOT NULL DEFAULT 300,
    fetch_pages          INT          NOT NULL DEFAULT 5,
    last_checked_at      TIMESTAMPTZ,
    last_error           VARCHAR(500) NOT NULL DEFAULT '',
    note                 VARCHAR(500) NOT NULL DEFAULT '',
    created_by           BIGINT       NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT card_platform_monitors_platform_check CHECK (platform_type IN ('ldxp')),
    CONSTRAINT card_platform_monitors_auth_mode_check CHECK (auth_mode IN ('public', 'token', 'cookie')),
    CONSTRAINT card_platform_monitors_interval_check CHECK (interval_seconds BETWEEN 60 AND 86400),
    CONSTRAINT card_platform_monitors_fetch_pages_check CHECK (fetch_pages BETWEEN 1 AND 500)
);

CREATE INDEX IF NOT EXISTS idx_card_platform_monitors_enabled_last_checked
    ON card_platform_monitors (enabled, last_checked_at);
CREATE INDEX IF NOT EXISTS idx_card_platform_monitors_platform
    ON card_platform_monitors (platform_type);

CREATE TABLE IF NOT EXISTS card_product_snapshots (
    id                  BIGSERIAL PRIMARY KEY,
    monitor_id          BIGINT        NOT NULL REFERENCES card_platform_monitors(id) ON DELETE CASCADE,
    external_product_id VARCHAR(120)  NOT NULL,
    title               VARCHAR(500)  NOT NULL DEFAULT '',
    merchant            VARCHAR(200)  NOT NULL DEFAULT '',
    category            VARCHAR(200)  NOT NULL DEFAULT '',
    image_url           TEXT          NOT NULL DEFAULT '',
    product_url         TEXT          NOT NULL DEFAULT '',
    price               NUMERIC(18,6),
    cost_price          NUMERIC(18,6),
    stock               BIGINT,
    sales               BIGINT,
    status              VARCHAR(30)   NOT NULL DEFAULT 'unknown',
    lowest_price        NUMERIC(18,6),
    raw_json            JSONB         NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_card_product_snapshots_monitor_external
    ON card_product_snapshots (monitor_id, external_product_id);
CREATE INDEX IF NOT EXISTS idx_card_product_snapshots_search
    ON card_product_snapshots USING GIN (
        to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(merchant, '') || ' ' || coalesce(category, ''))
    );
CREATE INDEX IF NOT EXISTS idx_card_product_snapshots_monitor_updated
    ON card_product_snapshots (monitor_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_card_product_snapshots_price
    ON card_product_snapshots (cost_price, price);

CREATE TABLE IF NOT EXISTS card_price_events (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  BIGINT       NOT NULL REFERENCES card_platform_monitors(id) ON DELETE CASCADE,
    product_id  BIGINT       REFERENCES card_product_snapshots(id) ON DELETE SET NULL,
    event_type  VARCHAR(30)  NOT NULL,
    title       VARCHAR(500) NOT NULL DEFAULT '',
    old_price   NUMERIC(18,6),
    new_price   NUMERIC(18,6),
    old_stock   BIGINT,
    new_stock   BIGINT,
    content     VARCHAR(500) NOT NULL DEFAULT '',
    detected_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT card_price_events_type_check CHECK (
        event_type IN ('new_product', 'price_down', 'price_up', 'new_low', 'restock', 'sold_out', 'offline', 'online', 'changed')
    )
);

CREATE INDEX IF NOT EXISTS idx_card_price_events_monitor_detected
    ON card_price_events (monitor_id, detected_at);
CREATE INDEX IF NOT EXISTS idx_card_price_events_type_detected
    ON card_price_events (event_type, detected_at);
CREATE INDEX IF NOT EXISTS idx_card_price_events_detected
    ON card_price_events (detected_at);
