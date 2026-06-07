package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	CardPlatformLDXP = "ldxp"

	CardAuthModePublic = "public"
	CardAuthModeToken  = "token"
	CardAuthModeCookie = "cookie"

	CardEventNewProduct = "new_product"
	CardEventPriceDown  = "price_down"
	CardEventPriceUp    = "price_up"
	CardEventNewLow     = "new_low"
	CardEventRestock    = "restock"
	CardEventSoldOut    = "sold_out"
	CardEventOffline    = "offline"
	CardEventOnline     = "online"

	cardMinIntervalSeconds       = 60
	cardMaxIntervalSeconds       = 86400
	cardDefaultIntervalSeconds   = 300
	cardDefaultFetchPages        = 5
	cardMaxFetchPages            = 500
	cardProbeTimeout             = 35 * time.Second
	cardRunOneBuffer             = 10 * time.Second
	cardWorkerConcurrency        = 5
	cardStartupLoadTimeout       = 10 * time.Second
	cardEventRetentionPerMonitor = 1000
)

var (
	ErrCardMonitorNotFound                = infraerrors.NotFound("CARD_MONITOR_NOT_FOUND", "card platform monitor not found")
	ErrCardMonitorMissingName             = infraerrors.BadRequest("CARD_MONITOR_MISSING_NAME", "name is required")
	ErrCardMonitorInvalidPlatform         = infraerrors.BadRequest("CARD_MONITOR_INVALID_PLATFORM", "platform_type must be ldxp")
	ErrCardMonitorInvalidAuthMode         = infraerrors.BadRequest("CARD_MONITOR_INVALID_AUTH_MODE", "auth_mode must be public/token/cookie")
	ErrCardMonitorInvalidInterval         = infraerrors.BadRequest("CARD_MONITOR_INVALID_INTERVAL", "interval_seconds must be in [60, 86400]")
	ErrCardMonitorInvalidFetchPages       = infraerrors.BadRequest("CARD_MONITOR_INVALID_FETCH_PAGES", "fetch_pages must be in [1, 500]")
	ErrCardMonitorMissingCredential       = infraerrors.BadRequest("CARD_MONITOR_MISSING_CREDENTIAL", "token/cookie mode requires credential")
	ErrCardMonitorCredentialDecryptFailed = infraerrors.InternalServer("CARD_MONITOR_CREDENTIAL_DECRYPT_FAILED", "credential decryption failed; please re-edit the monitor")
)

type CardPlatformMonitor struct {
	ID                      int64
	Name                    string
	PlatformType            string
	BaseURL                 string
	ShopURL                 string
	AuthMode                string
	Credential              string
	Enabled                 bool
	IntervalSeconds         int
	FetchPages              int
	LastCheckedAt           *time.Time
	LastError               string
	Note                    string
	CreatedBy               int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
	CredentialDecryptFailed bool `json:"-"`
}

type CardProductSnapshot struct {
	ID                int64      `json:"id"`
	MonitorID         int64      `json:"monitor_id"`
	PlatformName      string     `json:"platform_name"`
	PlatformType      string     `json:"platform_type"`
	ExternalProductID string     `json:"external_product_id"`
	Title             string     `json:"title"`
	Merchant          string     `json:"merchant"`
	Category          string     `json:"category"`
	ImageURL          string     `json:"image_url"`
	ProductURL        string     `json:"product_url"`
	Price             *float64   `json:"price"`
	CostPrice         *float64   `json:"cost_price"`
	Stock             *int64     `json:"stock"`
	Sales             *int64     `json:"sales"`
	Status            string     `json:"status"`
	LowestPrice       *float64   `json:"lowest_price"`
	FirstSeenAt       time.Time  `json:"first_seen_at"`
	LastSeenAt        time.Time  `json:"last_seen_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastEventType     string     `json:"last_event_type,omitempty"`
	LastEventAt       *time.Time `json:"last_event_at,omitempty"`
}

type CardPriceEvent struct {
	ID         int64     `json:"id"`
	MonitorID  int64     `json:"monitor_id"`
	ProductID  *int64    `json:"product_id"`
	Platform   string    `json:"platform"`
	EventType  string    `json:"event_type"`
	Title      string    `json:"title"`
	OldPrice   *float64  `json:"old_price"`
	NewPrice   *float64  `json:"new_price"`
	OldStock   *int64    `json:"old_stock"`
	NewStock   *int64    `json:"new_stock"`
	Content    string    `json:"content"`
	DetectedAt time.Time `json:"detected_at"`
}

type CardMonitorCreateParams struct {
	Name            string
	PlatformType    string
	BaseURL         string
	ShopURL         string
	AuthMode        string
	Credential      string
	Enabled         bool
	IntervalSeconds int
	FetchPages      int
	Note            string
	CreatedBy       int64
}

type CardMonitorUpdateParams struct {
	Name            *string
	PlatformType    *string
	BaseURL         *string
	ShopURL         *string
	AuthMode        *string
	Credential      *string
	Enabled         *bool
	IntervalSeconds *int
	FetchPages      *int
	Note            *string
}

type CardMonitorListParams struct {
	PlatformType string
	Enabled      *bool
	Search       string
	Page         int
	PageSize     int
}

type CardProductSearchParams struct {
	Search       string
	MonitorID    int64
	PlatformType string
	Status       string
	InStockOnly  bool
	Sort         string
	Page         int
	PageSize     int
}

type CardEventListParams struct {
	MonitorID int64
	EventType string
	Search    string
	Page      int
	PageSize  int
}

type CardSummary struct {
	PlatformCount int64 `json:"platform_count"`
	ProductCount  int64 `json:"product_count"`
	PriceDown     int64 `json:"price_down"`
	Restock       int64 `json:"restock"`
	ErrorCount    int64 `json:"error_count"`
}

type CardProbeResult struct {
	Products []*CardProductSnapshot `json:"products"`
	Events   []*CardPriceEvent      `json:"events"`
}

type CardPlatformMonitorRepository interface {
	Create(ctx context.Context, m *CardPlatformMonitor) error
	GetByID(ctx context.Context, id int64) (*CardPlatformMonitor, error)
	Update(ctx context.Context, m *CardPlatformMonitor) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params CardMonitorListParams) ([]*CardPlatformMonitor, int64, error)
	ListEnabled(ctx context.Context) ([]*CardPlatformMonitor, error)
	MarkChecked(ctx context.Context, id int64, checkedAt time.Time, lastErr string) error
	ListProducts(ctx context.Context, params CardProductSearchParams) ([]*CardProductSnapshot, int64, error)
	FindProduct(ctx context.Context, monitorID int64, externalID string) (*CardProductSnapshot, error)
	UpsertProduct(ctx context.Context, p *CardProductSnapshot, rawJSON []byte, at time.Time) error
	InsertEvents(ctx context.Context, rows []*CardPriceEvent) error
	ListEvents(ctx context.Context, params CardEventListParams) ([]*CardPriceEvent, int64, error)
	Summary(ctx context.Context, search string) (CardSummary, error)
	PruneEventsForMonitor(ctx context.Context, monitorID int64, keep int) error
}

type CardPlatformScheduler interface {
	Schedule(m *CardPlatformMonitor)
	Unschedule(id int64)
}
