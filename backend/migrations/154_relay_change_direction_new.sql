-- Migration: 154_relay_change_direction_new
-- 中转站监控：分组生命周期——新增分组事件。
-- relay_rate_changes.direction 增加 'new'（新增分组），与 up/down/(停用借用 down) 并列。

ALTER TABLE relay_rate_changes
    DROP CONSTRAINT IF EXISTS relay_rate_changes_direction_check;
ALTER TABLE relay_rate_changes
    ADD CONSTRAINT relay_rate_changes_direction_check
    CHECK (direction IN ('up', 'down', 'new'));
