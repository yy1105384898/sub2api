package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/relaymonitor"
	"github.com/Wei-Shaw/sub2api/ent/relayratechange"
	"github.com/Wei-Shaw/sub2api/ent/relayratesnapshot"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// relayMonitorRepository 实现 service.RelayMonitorRepository。
// CRUD 与查询走 ent；快照 upsert、历史裁剪走原生 SQL（避免 ent 在 ON CONFLICT 上的样板）。
type relayMonitorRepository struct {
	client *dbent.Client
	db     *sql.DB
}

// NewRelayMonitorRepository 创建仓储实例。
func NewRelayMonitorRepository(client *dbent.Client, db *sql.DB) service.RelayMonitorRepository {
	return &relayMonitorRepository{client: client, db: db}
}

// ---------- CRUD ----------

func (r *relayMonitorRepository) Create(ctx context.Context, m *service.RelayMonitor) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.RelayMonitor.Create().
		SetName(m.Name).
		SetSystem(relaymonitor.System(m.System)).
		SetBaseURL(m.BaseURL).
		SetVendor(m.Vendor).
		SetAuthAccount(m.AuthAccount).
		SetCredentialEncrypted(m.Credential). // 已是密文
		SetWatchedGroups(emptySliceIfNil(m.WatchedGroups)).
		SetEnabled(m.Enabled).
		SetIntervalSeconds(m.IntervalSeconds).
		SetCreatedBy(m.CreatedBy).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrRelayMonitorNotFound, nil)
	}
	m.ID = created.ID
	m.CreatedAt = created.CreatedAt
	m.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *relayMonitorRepository) GetByID(ctx context.Context, id int64) (*service.RelayMonitor, error) {
	row, err := r.client.RelayMonitor.Query().Where(relaymonitor.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrRelayMonitorNotFound, nil)
	}
	return entToServiceRelayMonitor(row), nil
}

func (r *relayMonitorRepository) Update(ctx context.Context, m *service.RelayMonitor) error {
	client := clientFromContext(ctx, r.client)
	updated, err := client.RelayMonitor.UpdateOneID(m.ID).
		SetName(m.Name).
		SetSystem(relaymonitor.System(m.System)).
		SetBaseURL(m.BaseURL).
		SetVendor(m.Vendor).
		SetAuthAccount(m.AuthAccount).
		SetCredentialEncrypted(m.Credential).
		SetWatchedGroups(emptySliceIfNil(m.WatchedGroups)).
		SetEnabled(m.Enabled).
		SetIntervalSeconds(m.IntervalSeconds).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrRelayMonitorNotFound, nil)
	}
	m.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *relayMonitorRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	if err := client.RelayMonitor.DeleteOneID(id).Exec(ctx); err != nil {
		return translatePersistenceError(err, service.ErrRelayMonitorNotFound, nil)
	}
	return nil
}

func (r *relayMonitorRepository) List(ctx context.Context, params service.RelayMonitorListParams) ([]*service.RelayMonitor, int64, error) {
	q := r.client.RelayMonitor.Query()
	if params.System != "" {
		q = q.Where(relaymonitor.SystemEQ(relaymonitor.System(params.System)))
	}
	if params.Enabled != nil {
		q = q.Where(relaymonitor.EnabledEQ(*params.Enabled))
	}
	if s := strings.TrimSpace(params.Search); s != "" {
		q = q.Where(relaymonitor.Or(
			relaymonitor.NameContainsFold(s),
			relaymonitor.VendorContainsFold(s),
			relaymonitor.BaseURLContainsFold(s),
		))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count relay monitors: %w", err)
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}

	rows, err := q.
		Order(dbent.Desc(relaymonitor.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list relay monitors: %w", err)
	}
	out := make([]*service.RelayMonitor, 0, len(rows))
	for _, row := range rows {
		out = append(out, entToServiceRelayMonitor(row))
	}
	return out, int64(total), nil
}

// ---------- 调度器辅助 ----------

func (r *relayMonitorRepository) ListEnabled(ctx context.Context) ([]*service.RelayMonitor, error) {
	rows, err := r.client.RelayMonitor.Query().Where(relaymonitor.EnabledEQ(true)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled relay monitors: %w", err)
	}
	out := make([]*service.RelayMonitor, 0, len(rows))
	for _, row := range rows {
		out = append(out, entToServiceRelayMonitor(row))
	}
	return out, nil
}

func (r *relayMonitorRepository) MarkChecked(ctx context.Context, id int64, checkedAt time.Time, lastErr string) error {
	client := clientFromContext(ctx, r.client)
	if err := client.RelayMonitor.UpdateOneID(id).
		SetLastCheckedAt(checkedAt).
		SetLastError(lastErr).
		Exec(ctx); err != nil {
		return translatePersistenceError(err, service.ErrRelayMonitorNotFound, nil)
	}
	return nil
}

// ---------- 快照 ----------

func (r *relayMonitorRepository) ListSnapshots(ctx context.Context, monitorID int64) ([]*service.RelayRateSnapshot, error) {
	rows, err := r.client.RelayRateSnapshot.Query().
		Where(relayratesnapshot.MonitorIDEQ(monitorID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	out := make([]*service.RelayRateSnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, &service.RelayRateSnapshot{
			MonitorID: row.MonitorID,
			GroupName: row.GroupName,
			Rate:      row.Rate,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

// UpsertSnapshot 借助 (monitor_id, group_name) 唯一索引做 upsert。
func (r *relayMonitorRepository) UpsertSnapshot(ctx context.Context, monitorID int64, group string, rate float64, at time.Time) error {
	const q = `
		INSERT INTO relay_rate_snapshots (monitor_id, group_name, rate, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (monitor_id, group_name) DO UPDATE SET
		    rate = EXCLUDED.rate,
		    updated_at = EXCLUDED.updated_at
	`
	if _, err := r.db.ExecContext(ctx, q, monitorID, group, rate, at); err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}
	return nil
}

// DeleteSnapshotsNotIn 删除不在 groups 列表中的快照。groups 为空时删除该监控全部快照。
func (r *relayMonitorRepository) DeleteSnapshotsNotIn(ctx context.Context, monitorID int64, groups []string) error {
	client := clientFromContext(ctx, r.client)
	del := client.RelayRateSnapshot.Delete().Where(relayratesnapshot.MonitorIDEQ(monitorID))
	if len(groups) > 0 {
		del = del.Where(relayratesnapshot.GroupNameNotIn(groups...))
	}
	if _, err := del.Exec(ctx); err != nil {
		return fmt.Errorf("delete stale snapshots: %w", err)
	}
	return nil
}

// ListOverview 返回所有被跟踪分组的当前倍率 + 最近一次变化（LATERAL 取每组最近一条）。
// 变化过的排前面（changed_at 倒序），其余按 site/group。search 模糊匹配 site/vendor/group。
func (r *relayMonitorRepository) ListOverview(ctx context.Context, search string) ([]*service.RelayGroupOverview, error) {
	const q = `
		SELECT s.monitor_id, m.name AS site, m.base_url, m.system, m.vendor, s.group_name,
		       s.rate AS current_rate, s.updated_at, (s.rate < 0) AS removed,
		       c.old_rate, c.new_rate, c.direction, c.detected_at
		FROM relay_rate_snapshots s
		JOIN relay_monitors m ON m.id = s.monitor_id
		LEFT JOIN LATERAL (
		    SELECT old_rate, new_rate, direction, detected_at
		    FROM relay_rate_changes rc
		    WHERE rc.monitor_id = s.monitor_id AND rc.group_name = s.group_name
		    ORDER BY rc.detected_at DESC, rc.id DESC
		    LIMIT 1
		) c ON TRUE
		WHERE ($1 = '' OR m.name ILIKE '%'||$1||'%' OR m.vendor ILIKE '%'||$1||'%' OR s.group_name ILIKE '%'||$1||'%')
		ORDER BY (c.detected_at IS NOT NULL) DESC, c.detected_at DESC NULLS LAST, m.name, s.group_name
	`
	rows, err := r.db.QueryContext(ctx, q, search)
	if err != nil {
		return nil, fmt.Errorf("list overview: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]*service.RelayGroupOverview, 0)
	for rows.Next() {
		o := &service.RelayGroupOverview{}
		var oldRate, newRate sql.NullFloat64
		var direction sql.NullString
		var changedAt sql.NullTime
		if err := rows.Scan(&o.MonitorID, &o.Site, &o.BaseURL, &o.System, &o.Vendor, &o.GroupName,
			&o.CurrentRate, &o.UpdatedAt, &o.Removed, &oldRate, &newRate, &direction, &changedAt); err != nil {
			return nil, fmt.Errorf("scan overview row: %w", err)
		}
		if changedAt.Valid {
			o.HasChange = true
			o.OldRate = oldRate.Float64
			o.NewRate = newRate.Float64
			o.Direction = direction.String
			t := changedAt.Time
			o.ChangedAt = &t
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ---------- 变化历史 ----------

func (r *relayMonitorRepository) InsertChanges(ctx context.Context, rows []*service.RelayRateChange) error {
	if len(rows) == 0 {
		return nil
	}
	client := clientFromContext(ctx, r.client)
	bulk := make([]*dbent.RelayRateChangeCreate, 0, len(rows))
	for _, row := range rows {
		bulk = append(bulk, client.RelayRateChange.Create().
			SetMonitorID(row.MonitorID).
			SetSite(row.Site).
			SetSystem(row.System).
			SetVendor(row.Vendor).
			SetGroupName(row.GroupName).
			SetOldRate(row.OldRate).
			SetNewRate(row.NewRate).
			SetDirection(relayratechange.Direction(row.Direction)).
			SetContent(row.Content).
			SetDetectedAt(row.DetectedAt))
	}
	if _, err := client.RelayRateChange.CreateBulk(bulk...).Save(ctx); err != nil {
		return fmt.Errorf("insert changes: %w", err)
	}
	return nil
}

func (r *relayMonitorRepository) ListChanges(ctx context.Context, params service.RelayRateChangeListParams) ([]*service.RelayRateChange, int64, error) {
	q := r.buildChangeQuery(params)

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count changes: %w", err)
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}

	rows, err := q.
		Order(dbent.Desc(relayratechange.FieldDetectedAt), dbent.Desc(relayratechange.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list changes: %w", err)
	}
	out := make([]*service.RelayRateChange, 0, len(rows))
	for _, row := range rows {
		out = append(out, entToServiceRelayChange(row))
	}
	return out, int64(total), nil
}

func (r *relayMonitorRepository) SummarizeChanges(ctx context.Context, params service.RelayRateChangeListParams) (service.RelayChangeSummary, error) {
	var summary service.RelayChangeSummary
	upParams := params
	upParams.Direction = service.RelayDirectionUp
	up, err := r.buildChangeQuery(upParams).Count(ctx)
	if err != nil {
		return summary, fmt.Errorf("count up changes: %w", err)
	}
	downParams := params
	downParams.Direction = service.RelayDirectionDown
	down, err := r.buildChangeQuery(downParams).Count(ctx)
	if err != nil {
		return summary, fmt.Errorf("count down changes: %w", err)
	}
	summary.UpCount = int64(up)
	summary.DownCount = int64(down)
	return summary, nil
}

func (r *relayMonitorRepository) DeleteChange(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	if err := client.RelayRateChange.DeleteOneID(id).Exec(ctx); err != nil {
		return translatePersistenceError(err, service.ErrRelayMonitorNotFound, nil)
	}
	return nil
}

// buildChangeQuery 根据 params 组装变化历史查询（monitor/direction/search 过滤）。
func (r *relayMonitorRepository) buildChangeQuery(params service.RelayRateChangeListParams) *dbent.RelayRateChangeQuery {
	q := r.client.RelayRateChange.Query()
	if params.MonitorID > 0 {
		q = q.Where(relayratechange.MonitorIDEQ(params.MonitorID))
	}
	if params.Direction != "" {
		q = q.Where(relayratechange.DirectionEQ(relayratechange.Direction(params.Direction)))
	}
	if s := strings.TrimSpace(params.Search); s != "" {
		q = q.Where(relayratechange.Or(
			relayratechange.SiteContainsFold(s),
			relayratechange.VendorContainsFold(s),
			relayratechange.GroupNameContainsFold(s),
		))
	}
	return q
}

// PruneChangesForMonitor 仅保留某监控最新 keep 条变化，物理删除更早的。
func (r *relayMonitorRepository) PruneChangesForMonitor(ctx context.Context, monitorID int64, keep int) error {
	if keep <= 0 {
		return nil
	}
	const q = `
		DELETE FROM relay_rate_changes
		WHERE monitor_id = $1
		  AND id NOT IN (
		      SELECT id FROM relay_rate_changes
		      WHERE monitor_id = $1
		      ORDER BY detected_at DESC, id DESC
		      LIMIT $2
		  )
	`
	if _, err := r.db.ExecContext(ctx, q, monitorID, keep); err != nil {
		return fmt.Errorf("prune changes: %w", err)
	}
	return nil
}

// ---------- helpers ----------

func entToServiceRelayMonitor(row *dbent.RelayMonitor) *service.RelayMonitor {
	if row == nil {
		return nil
	}
	groups := row.WatchedGroups
	if groups == nil {
		groups = []string{}
	}
	return &service.RelayMonitor{
		ID:              row.ID,
		Name:            row.Name,
		System:          string(row.System),
		BaseURL:         row.BaseURL,
		Vendor:          row.Vendor,
		AuthAccount:     row.AuthAccount,
		Credential:      row.CredentialEncrypted, // 仍为密文，service 层解密
		WatchedGroups:   groups,
		Enabled:         row.Enabled,
		IntervalSeconds: row.IntervalSeconds,
		LastCheckedAt:   row.LastCheckedAt,
		LastError:       row.LastError,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func entToServiceRelayChange(row *dbent.RelayRateChange) *service.RelayRateChange {
	return &service.RelayRateChange{
		ID:         row.ID,
		MonitorID:  row.MonitorID,
		Site:       row.Site,
		System:     row.System,
		Vendor:     row.Vendor,
		GroupName:  row.GroupName,
		OldRate:    row.OldRate,
		NewRate:    row.NewRate,
		Direction:  string(row.Direction),
		Content:    row.Content,
		DetectedAt: row.DetectedAt,
	}
}
