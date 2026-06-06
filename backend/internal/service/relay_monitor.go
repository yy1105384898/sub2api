package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 中转站监控（relay monitor）领域类型、仓储接口、错误与常量集中声明。
//
// 与渠道监控（channel_monitor）相互独立：渠道监控测的是自己挂的上游账号心跳，
// 中转站监控抓的是**外部**中转站（sub2api / newapi）对外公布的分组倍率，
// 记录涨/跌变化。两者复用同一套 SSRF 防护与 HTTP client（见 channel_monitor_ssrf.go）。

const (
	// RelaySystemSub2API / RelaySystemNewAPI 目标站点系统类型（也是 ent enum 值）。
	RelaySystemSub2API = "sub2api"
	RelaySystemNewAPI  = "newapi"

	// RelayDirectionUp 倍率变大（涨），RelayDirectionDown 倍率变小（跌）。
	RelayDirectionUp   = "up"
	RelayDirectionDown = "down"

	// relayMinIntervalSeconds / relayMaxIntervalSeconds 探测间隔上下限（与 schema Range 一致）。
	relayMinIntervalSeconds = 60
	relayMaxIntervalSeconds = 86400
	// relayDefaultIntervalSeconds 默认探测间隔。
	relayDefaultIntervalSeconds = 300

	// relayProbeTimeout 单次抓取目标站的总超时。
	relayProbeTimeout = 30 * time.Second
	// relayResponseMaxBytes 单次响应最大读取字节，防止恶意大响应 OOM。
	relayResponseMaxBytes = 2 * 1024 * 1024

	// relayChangeRetentionPerMonitor 每个监控保留的倍率变化历史上限，超出由探测后裁剪。
	relayChangeRetentionPerMonitor = 500

	// relayWorkerConcurrency 调度器并发执行的探测数（pond 池容量）。
	relayWorkerConcurrency = 5
	// relayStartupLoadTimeout Start 时加载 enabled 监控的总超时。
	relayStartupLoadTimeout = 10 * time.Second
	// relayRunOneBuffer runOne 的总超时缓冲。
	relayRunOneBuffer = 10 * time.Second
)

// 业务错误集中声明。
var (
	ErrRelayMonitorNotFound = infraerrors.NotFound(
		"RELAY_MONITOR_NOT_FOUND", "relay monitor not found",
	)
	ErrRelayMonitorInvalidSystem = infraerrors.BadRequest(
		"RELAY_MONITOR_INVALID_SYSTEM", "system must be one of sub2api/newapi",
	)
	ErrRelayMonitorInvalidInterval = infraerrors.BadRequest(
		"RELAY_MONITOR_INVALID_INTERVAL", "interval_seconds must be in [60, 86400]",
	)
	ErrRelayMonitorMissingName = infraerrors.BadRequest(
		"RELAY_MONITOR_MISSING_NAME", "name is required",
	)
	ErrRelayMonitorMissingCredential = infraerrors.BadRequest(
		"RELAY_MONITOR_MISSING_CREDENTIAL", "sub2api site requires login email and password",
	)
	ErrRelayMonitorCredentialDecryptFailed = infraerrors.InternalServer(
		"RELAY_MONITOR_CREDENTIAL_DECRYPT_FAILED", "credential decryption failed; please re-edit the monitor with a fresh token",
	)
	ErrRelayMonitorProbeFailed = infraerrors.BadRequest(
		"RELAY_MONITOR_PROBE_FAILED", "failed to fetch group rates from the target site",
	)
)

// RelayMonitor 中转站监控配置领域模型。
// Credential 在 service 层为明文（已解密）；传给 repository 时为密文（与 ChannelMonitor.APIKey 一致）。
type RelayMonitor struct {
	ID              int64
	Name            string
	System          string
	BaseURL         string
	Vendor          string
	AuthAccount     string
	Credential      string
	WatchedGroups   []string
	Enabled         bool
	IntervalSeconds int
	LastCheckedAt   *time.Time
	LastError       string
	CreatedBy       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// CredentialDecryptFailed 解密失败标志；探测前必须检查并拒绝执行。
	CredentialDecryptFailed bool `json:"-"`
}

// RelayGroupRate 一次探测得到的单个分组倍率。
type RelayGroupRate struct {
	GroupName string  `json:"group_name"`
	Rate      float64 `json:"rate"`
}

// RelayRateSnapshot 当前倍率快照（每个监控每个被跟踪分组一行）。
type RelayRateSnapshot struct {
	MonitorID int64
	GroupName string
	Rate      float64
	UpdatedAt time.Time
}

// RelayRateChange 倍率变化历史条目（涨/跌公告）。
type RelayRateChange struct {
	ID         int64     `json:"id"`
	MonitorID  int64     `json:"monitor_id"`
	Site       string    `json:"site"`
	System     string    `json:"system"`
	Vendor     string    `json:"vendor"`
	GroupName  string    `json:"group_name"`
	OldRate    float64   `json:"old_rate"`
	NewRate    float64   `json:"new_rate"`
	Direction  string    `json:"direction"`
	Content    string    `json:"content"`
	DetectedAt time.Time `json:"detected_at"`
}

// RelayMonitorCreateParams 创建参数。Credential 为明文 token（sub2api 必填，newapi 可空）。
type RelayMonitorCreateParams struct {
	Name            string
	System          string
	BaseURL         string
	Vendor          string
	AuthAccount     string
	Credential      string
	WatchedGroups   []string
	Enabled         bool
	IntervalSeconds int
	CreatedBy       int64
}

// RelayMonitorUpdateParams 更新参数；非 nil 字段才覆盖。
// Credential 为 nil 或空 = 不修改；非空 = 加密覆盖。
type RelayMonitorUpdateParams struct {
	Name            *string
	System          *string
	BaseURL         *string
	Vendor          *string
	AuthAccount     *string
	Credential      *string
	WatchedGroups   *[]string
	Enabled         *bool
	IntervalSeconds *int
}

// RelayMonitorListParams 监控列表查询参数。
type RelayMonitorListParams struct {
	System   string
	Enabled  *bool
	Search   string
	Page     int
	PageSize int
}

// RelayRateChangeListParams 倍率变化历史查询参数。
// Direction 为空返回全部；非空时只返回 up 或 down。Search 模糊匹配 site/vendor/group_name。
type RelayRateChangeListParams struct {
	MonitorID int64 // 0 表示不限定监控
	Direction string
	Search    string
	Page      int
	PageSize  int
}

// RelayChangeSummary 涨/跌公告汇总（顶部统计卡）。
type RelayChangeSummary struct {
	UpCount   int64 `json:"up_count"`
	DownCount int64 `json:"down_count"`
}

// RelayMonitorRepository 中转站监控数据访问接口。
// repository 实现负责与 ent 模型互转，并保持 credential 字段为密文。
type RelayMonitorRepository interface {
	// CRUD
	Create(ctx context.Context, m *RelayMonitor) error
	GetByID(ctx context.Context, id int64) (*RelayMonitor, error)
	Update(ctx context.Context, m *RelayMonitor) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params RelayMonitorListParams) ([]*RelayMonitor, int64, error)

	// 调度器辅助
	ListEnabled(ctx context.Context) ([]*RelayMonitor, error)
	MarkChecked(ctx context.Context, id int64, checkedAt time.Time, lastErr string) error

	// 快照
	ListSnapshots(ctx context.Context, monitorID int64) ([]*RelayRateSnapshot, error)
	UpsertSnapshot(ctx context.Context, monitorID int64, group string, rate float64, at time.Time) error
	DeleteSnapshotsNotIn(ctx context.Context, monitorID int64, groups []string) error

	// 变化历史
	InsertChanges(ctx context.Context, rows []*RelayRateChange) error
	ListChanges(ctx context.Context, params RelayRateChangeListParams) ([]*RelayRateChange, int64, error)
	SummarizeChanges(ctx context.Context, params RelayRateChangeListParams) (RelayChangeSummary, error)
	// PruneChangesForMonitor 仅保留某监控最新 keep 条变化，物理删除更早的。
	PruneChangesForMonitor(ctx context.Context, monitorID int64, keep int) error
}
