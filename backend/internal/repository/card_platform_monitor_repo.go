package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type cardPlatformMonitorRepository struct {
	db *sql.DB
}

func NewCardPlatformMonitorRepository(_ *dbent.Client, db *sql.DB) service.CardPlatformMonitorRepository {
	return &cardPlatformMonitorRepository{db: db}
}

func (r *cardPlatformMonitorRepository) Create(ctx context.Context, m *service.CardPlatformMonitor) error {
	const q = `
		INSERT INTO card_platform_monitors
		    (name, platform_type, base_url, shop_url, auth_mode, credential_encrypted, enabled, interval_seconds, fetch_pages, note, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at, updated_at
	`
	if err := r.db.QueryRowContext(ctx, q, m.Name, m.PlatformType, m.BaseURL, m.ShopURL, m.AuthMode, m.Credential, m.Enabled, m.IntervalSeconds, m.FetchPages, m.Note, m.CreatedBy).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return fmt.Errorf("create card monitor: %w", err)
	}
	return nil
}

func (r *cardPlatformMonitorRepository) GetByID(ctx context.Context, id int64) (*service.CardPlatformMonitor, error) {
	const q = `
		SELECT id, name, platform_type, base_url, shop_url, auth_mode, credential_encrypted, enabled,
		       interval_seconds, fetch_pages, last_checked_at, last_error, note, created_by, created_at, updated_at
		FROM card_platform_monitors WHERE id=$1
	`
	m, err := scanCardMonitor(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrCardMonitorNotFound
		}
		return nil, fmt.Errorf("get card monitor: %w", err)
	}
	return m, nil
}

func (r *cardPlatformMonitorRepository) Update(ctx context.Context, m *service.CardPlatformMonitor) error {
	const q = `
		UPDATE card_platform_monitors
		SET name=$2, platform_type=$3, base_url=$4, shop_url=$5, auth_mode=$6, credential_encrypted=$7,
		    enabled=$8, interval_seconds=$9, fetch_pages=$10, note=$11, updated_at=NOW()
		WHERE id=$1
		RETURNING updated_at
	`
	if err := r.db.QueryRowContext(ctx, q, m.ID, m.Name, m.PlatformType, m.BaseURL, m.ShopURL, m.AuthMode, m.Credential, m.Enabled, m.IntervalSeconds, m.FetchPages, m.Note).
		Scan(&m.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return service.ErrCardMonitorNotFound
		}
		return fmt.Errorf("update card monitor: %w", err)
	}
	return nil
}

func (r *cardPlatformMonitorRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM card_platform_monitors WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete card monitor: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return service.ErrCardMonitorNotFound
	}
	return nil
}

func (r *cardPlatformMonitorRepository) List(ctx context.Context, params service.CardMonitorListParams) ([]*service.CardPlatformMonitor, int64, error) {
	where, args := cardMonitorWhere(params.PlatformType, params.Enabled, params.Search)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM card_platform_monitors `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count card monitors: %w", err)
	}
	page, pageSize := normalizeRepoPage(params.Page, params.PageSize)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, platform_type, base_url, shop_url, auth_mode, credential_encrypted, enabled,
		       interval_seconds, fetch_pages, last_checked_at, last_error, note, created_by, created_at, updated_at
		FROM card_platform_monitors `+where+`
		ORDER BY id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list card monitors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanCardMonitorRows(rows)
	return items, total, err
}

func (r *cardPlatformMonitorRepository) ListEnabled(ctx context.Context) ([]*service.CardPlatformMonitor, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, platform_type, base_url, shop_url, auth_mode, credential_encrypted, enabled,
		       interval_seconds, fetch_pages, last_checked_at, last_error, note, created_by, created_at, updated_at
		FROM card_platform_monitors WHERE enabled = TRUE ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled card monitors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanCardMonitorRows(rows)
}

func (r *cardPlatformMonitorRepository) MarkChecked(ctx context.Context, id int64, checkedAt time.Time, lastErr string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE card_platform_monitors SET last_checked_at=$2, last_error=$3, updated_at=NOW() WHERE id=$1
	`, id, checkedAt, trimForDB(lastErr, 500))
	if err != nil {
		return fmt.Errorf("mark card monitor checked: %w", err)
	}
	return nil
}

func (r *cardPlatformMonitorRepository) ListProducts(ctx context.Context, params service.CardProductSearchParams) ([]*service.CardProductSnapshot, int64, error) {
	where, args := cardProductWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM card_product_snapshots p JOIN card_platform_monitors m ON m.id=p.monitor_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count card products: %w", err)
	}
	page, pageSize := normalizeRepoPage(params.Page, params.PageSize)
	order := cardProductOrder(params.Sort)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.monitor_id, m.name, m.platform_type, p.external_product_id, p.title, p.merchant, p.category,
		       p.image_url, p.product_url, p.price, p.cost_price, p.stock, p.sales, p.status, p.lowest_price,
		       p.first_seen_at, p.last_seen_at, p.updated_at,
		       e.event_type, e.detected_at
		FROM card_product_snapshots p
		JOIN card_platform_monitors m ON m.id=p.monitor_id
		LEFT JOIN LATERAL (
		    SELECT event_type, detected_at FROM card_price_events ce
		    WHERE ce.monitor_id=p.monitor_id AND ce.product_id=p.id
		    ORDER BY ce.detected_at DESC, ce.id DESC LIMIT 1
		) e ON TRUE
		`+where+` `+order+` LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list card products: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items, err := scanCardProductRows(rows)
	return items, total, err
}

func (r *cardPlatformMonitorRepository) FindProduct(ctx context.Context, monitorID int64, externalID string) (*service.CardProductSnapshot, error) {
	const q = `
		SELECT p.id, p.monitor_id, m.name, m.platform_type, p.external_product_id, p.title, p.merchant, p.category,
		       p.image_url, p.product_url, p.price, p.cost_price, p.stock, p.sales, p.status, p.lowest_price,
		       p.first_seen_at, p.last_seen_at, p.updated_at, NULL::varchar, NULL::timestamptz
		FROM card_product_snapshots p JOIN card_platform_monitors m ON m.id=p.monitor_id
		WHERE p.monitor_id=$1 AND p.external_product_id=$2
	`
	rows, err := r.db.QueryContext(ctx, q, monitorID, externalID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanCardProductRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *cardPlatformMonitorRepository) UpsertProduct(ctx context.Context, p *service.CardProductSnapshot, rawJSON []byte, at time.Time) error {
	if len(rawJSON) == 0 {
		rawJSON = []byte(`{}`)
	}
	const q = `
		INSERT INTO card_product_snapshots
		    (monitor_id, external_product_id, title, merchant, category, image_url, product_url, price, cost_price,
		     stock, sales, status, lowest_price, raw_json, first_seen_at, last_seen_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$15,$15)
		ON CONFLICT (monitor_id, external_product_id) DO UPDATE SET
		    title=EXCLUDED.title,
		    merchant=EXCLUDED.merchant,
		    category=EXCLUDED.category,
		    image_url=EXCLUDED.image_url,
		    product_url=EXCLUDED.product_url,
		    price=EXCLUDED.price,
		    cost_price=EXCLUDED.cost_price,
		    stock=EXCLUDED.stock,
		    sales=EXCLUDED.sales,
		    status=EXCLUDED.status,
		    lowest_price=LEAST(COALESCE(card_product_snapshots.lowest_price, EXCLUDED.lowest_price), COALESCE(EXCLUDED.lowest_price, card_product_snapshots.lowest_price)),
		    raw_json=EXCLUDED.raw_json,
		    last_seen_at=EXCLUDED.last_seen_at,
		    updated_at=EXCLUDED.updated_at
		RETURNING id, first_seen_at, last_seen_at, updated_at, lowest_price
	`
	if err := r.db.QueryRowContext(ctx, q, p.MonitorID, p.ExternalProductID, p.Title, p.Merchant, p.Category, p.ImageURL, p.ProductURL,
		p.Price, p.CostPrice, p.Stock, p.Sales, p.Status, p.LowestPrice, string(rawJSON), at).
		Scan(&p.ID, &p.FirstSeenAt, &p.LastSeenAt, &p.UpdatedAt, &p.LowestPrice); err != nil {
		return fmt.Errorf("upsert card product: %w", err)
	}
	return nil
}

func (r *cardPlatformMonitorRepository) InsertEvents(ctx context.Context, rows []*service.CardPriceEvent) error {
	if len(rows) == 0 {
		return nil
	}
	const q = `
		INSERT INTO card_price_events (monitor_id, product_id, event_type, title, old_price, new_price, old_stock, new_stock, content, detected_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`
	for _, row := range rows {
		_, err := r.db.ExecContext(ctx, q, row.MonitorID, row.ProductID, row.EventType, row.Title, row.OldPrice, row.NewPrice, row.OldStock, row.NewStock, row.Content, row.DetectedAt)
		if err != nil {
			return fmt.Errorf("insert card event: %w", err)
		}
	}
	return nil
}

func (r *cardPlatformMonitorRepository) ListEvents(ctx context.Context, params service.CardEventListParams) ([]*service.CardPriceEvent, int64, error) {
	where, args := cardEventWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM card_price_events e JOIN card_platform_monitors m ON m.id=e.monitor_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count card events: %w", err)
	}
	page, pageSize := normalizeRepoPage(params.Page, params.PageSize)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.id, e.monitor_id, e.product_id, m.name, e.event_type, e.title, e.old_price, e.new_price,
		       e.old_stock, e.new_stock, e.content, e.detected_at
		FROM card_price_events e
		JOIN card_platform_monitors m ON m.id=e.monitor_id
		`+where+` ORDER BY e.detected_at DESC, e.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list card events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]*service.CardPriceEvent, 0)
	for rows.Next() {
		e := &service.CardPriceEvent{}
		var productID sql.NullInt64
		var oldPrice, newPrice sql.NullFloat64
		var oldStock, newStock sql.NullInt64
		if err := rows.Scan(&e.ID, &e.MonitorID, &productID, &e.Platform, &e.EventType, &e.Title, &oldPrice, &newPrice, &oldStock, &newStock, &e.Content, &e.DetectedAt); err != nil {
			return nil, 0, err
		}
		if productID.Valid {
			v := productID.Int64
			e.ProductID = &v
		}
		if oldPrice.Valid {
			v := oldPrice.Float64
			e.OldPrice = &v
		}
		if newPrice.Valid {
			v := newPrice.Float64
			e.NewPrice = &v
		}
		if oldStock.Valid {
			v := oldStock.Int64
			e.OldStock = &v
		}
		if newStock.Valid {
			v := newStock.Int64
			e.NewStock = &v
		}
		items = append(items, e)
	}
	return items, total, rows.Err()
}

func (r *cardPlatformMonitorRepository) Summary(ctx context.Context, search string) (service.CardSummary, error) {
	var s service.CardSummary
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE last_error <> '') FROM card_platform_monitors`).Scan(&s.PlatformCount, &s.ErrorCount); err != nil {
		return s, err
	}
	productWhere, args := cardProductWhere(service.CardProductSearchParams{Search: strings.TrimSpace(search)})
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM card_product_snapshots p JOIN card_platform_monitors m ON m.id=p.monitor_id `+productWhere, args...).Scan(&s.ProductCount); err != nil {
		return s, err
	}
	eventWhere, eventArgs := cardEventWhere(service.CardEventListParams{Search: strings.TrimSpace(search)})
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE e.event_type IN ('price_down','new_low')),
		       COUNT(*) FILTER (WHERE e.event_type='restock')
		FROM card_price_events e JOIN card_platform_monitors m ON m.id=e.monitor_id `+eventWhere, eventArgs...).Scan(&s.PriceDown, &s.Restock); err != nil {
		return s, err
	}
	return s, nil
}

func (r *cardPlatformMonitorRepository) PruneEventsForMonitor(ctx context.Context, monitorID int64, keep int) error {
	if keep <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM card_price_events
		WHERE monitor_id=$1 AND id NOT IN (
		    SELECT id FROM card_price_events WHERE monitor_id=$1 ORDER BY detected_at DESC, id DESC LIMIT $2
		)
	`, monitorID, keep)
	return err
}

type scannerRow interface{ Scan(dest ...any) error }

func scanCardMonitor(row scannerRow) (*service.CardPlatformMonitor, error) {
	m := &service.CardPlatformMonitor{}
	var lastChecked sql.NullTime
	if err := row.Scan(&m.ID, &m.Name, &m.PlatformType, &m.BaseURL, &m.ShopURL, &m.AuthMode, &m.Credential,
		&m.Enabled, &m.IntervalSeconds, &m.FetchPages, &lastChecked, &m.LastError, &m.Note, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	if lastChecked.Valid {
		t := lastChecked.Time
		m.LastCheckedAt = &t
	}
	return m, nil
}

func scanCardMonitorRows(rows *sql.Rows) ([]*service.CardPlatformMonitor, error) {
	out := make([]*service.CardPlatformMonitor, 0)
	for rows.Next() {
		m, err := scanCardMonitor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanCardProductRows(rows *sql.Rows) ([]*service.CardProductSnapshot, error) {
	out := make([]*service.CardProductSnapshot, 0)
	for rows.Next() {
		p := &service.CardProductSnapshot{}
		var price, cost, lowest sql.NullFloat64
		var stock, sales sql.NullInt64
		var eventType sql.NullString
		var eventAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.MonitorID, &p.PlatformName, &p.PlatformType, &p.ExternalProductID, &p.Title, &p.Merchant, &p.Category,
			&p.ImageURL, &p.ProductURL, &price, &cost, &stock, &sales, &p.Status, &lowest, &p.FirstSeenAt, &p.LastSeenAt, &p.UpdatedAt, &eventType, &eventAt); err != nil {
			return nil, err
		}
		if price.Valid {
			v := price.Float64
			p.Price = &v
		}
		if cost.Valid {
			v := cost.Float64
			p.CostPrice = &v
		}
		if stock.Valid {
			v := stock.Int64
			p.Stock = &v
		}
		if sales.Valid {
			v := sales.Int64
			p.Sales = &v
		}
		if lowest.Valid {
			v := lowest.Float64
			p.LowestPrice = &v
		}
		if eventType.Valid {
			p.LastEventType = eventType.String
		}
		if eventAt.Valid {
			t := eventAt.Time
			p.LastEventAt = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func cardMonitorWhere(platform string, enabled *bool, search string) (string, []any) {
	parts := []string{"1=1"}
	args := []any{}
	if platform = strings.TrimSpace(platform); platform != "" {
		args = append(args, platform)
		parts = append(parts, fmt.Sprintf("platform_type=$%d", len(args)))
	}
	if enabled != nil {
		args = append(args, *enabled)
		parts = append(parts, fmt.Sprintf("enabled=$%d", len(args)))
	}
	if search = strings.TrimSpace(search); search != "" {
		args = append(args, search)
		parts = append(parts, fmt.Sprintf("(name ILIKE '%%'||$%d||'%%' OR base_url ILIKE '%%'||$%d||'%%' OR shop_url ILIKE '%%'||$%d||'%%' OR note ILIKE '%%'||$%d||'%%')", len(args), len(args), len(args), len(args)))
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func cardProductWhere(params service.CardProductSearchParams) (string, []any) {
	parts := []string{"1=1"}
	args := []any{}
	if params.MonitorID > 0 {
		args = append(args, params.MonitorID)
		parts = append(parts, fmt.Sprintf("p.monitor_id=$%d", len(args)))
	}
	if params.PlatformType != "" {
		args = append(args, params.PlatformType)
		parts = append(parts, fmt.Sprintf("m.platform_type=$%d", len(args)))
	}
	if params.Status != "" {
		args = append(args, params.Status)
		parts = append(parts, fmt.Sprintf("p.status=$%d", len(args)))
	}
	if params.InStockOnly {
		parts = append(parts, "COALESCE(p.stock, 0) > 0")
	}
	if s := strings.TrimSpace(params.Search); s != "" {
		args = append(args, s)
		parts = append(parts, fmt.Sprintf("(p.title ILIKE '%%'||$%d||'%%' OR p.merchant ILIKE '%%'||$%d||'%%' OR p.category ILIKE '%%'||$%d||'%%' OR m.name ILIKE '%%'||$%d||'%%')", len(args), len(args), len(args), len(args)))
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func cardEventWhere(params service.CardEventListParams) (string, []any) {
	parts := []string{"1=1"}
	args := []any{}
	if params.MonitorID > 0 {
		args = append(args, params.MonitorID)
		parts = append(parts, fmt.Sprintf("e.monitor_id=$%d", len(args)))
	}
	if params.EventType != "" {
		args = append(args, params.EventType)
		parts = append(parts, fmt.Sprintf("e.event_type=$%d", len(args)))
	}
	if s := strings.TrimSpace(params.Search); s != "" {
		args = append(args, s)
		parts = append(parts, fmt.Sprintf("(e.title ILIKE '%%'||$%d||'%%' OR e.content ILIKE '%%'||$%d||'%%' OR m.name ILIKE '%%'||$%d||'%%')", len(args), len(args), len(args)))
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func cardProductOrder(sort string) string {
	switch sort {
	case "priceAsc":
		return "ORDER BY COALESCE(p.cost_price, p.price, 999999999) ASC, p.updated_at DESC"
	case "priceDesc":
		return "ORDER BY COALESCE(p.cost_price, p.price, -1) DESC, p.updated_at DESC"
	case "stockDesc":
		return "ORDER BY COALESCE(p.stock, -1) DESC, p.updated_at DESC"
	case "newest":
		return "ORDER BY p.first_seen_at DESC, p.id DESC"
	default:
		return "ORDER BY p.updated_at DESC, p.id DESC"
	}
}

func normalizeRepoPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func trimForDB(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max]
	}
	return s
}

func mustJSONBytes(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
