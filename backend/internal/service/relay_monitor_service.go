package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// RelayMonitorService 中转站监控管理服务。
type RelayMonitorService struct {
	repo      RelayMonitorRepository
	encryptor SecretEncryptor
	// scheduler 由 wire 通过 SetScheduler 注入；CRUD 后回调即时同步定时任务。
	scheduler RelayScheduler
}

// NewRelayMonitorService 创建服务实例。
func NewRelayMonitorService(repo RelayMonitorRepository, encryptor SecretEncryptor) *RelayMonitorService {
	return &RelayMonitorService{repo: repo, encryptor: encryptor}
}

// RelayProbeResult 一次探测的结果：当前各被跟踪分组倍率 + 本次检测到的变化。
type RelayProbeResult struct {
	Rates   []RelayGroupRate   `json:"rates"`
	Changes []*RelayRateChange `json:"changes"`
}

// ---------- CRUD ----------

// List 列表查询（system/enabled/search 过滤 + 分页）。返回的 Credential 已解密。
func (s *RelayMonitorService) List(ctx context.Context, params RelayMonitorListParams) ([]*RelayMonitor, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 20
	}
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list relay monitors: %w", err)
	}
	for _, it := range items {
		s.decryptInPlace(it)
	}
	return items, total, nil
}

// Get 查询单个监控（解密凭证）。
func (s *RelayMonitorService) Get(ctx context.Context, id int64) (*RelayMonitor, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.decryptInPlace(m)
	return m, nil
}

// Create 创建监控（内部加密凭证）。
func (s *RelayMonitorService) Create(ctx context.Context, p RelayMonitorCreateParams) (*RelayMonitor, error) {
	if err := validateRelayCreate(p); err != nil {
		return nil, err
	}
	encrypted, err := s.encryptCredential(p.Credential)
	if err != nil {
		return nil, err
	}
	interval := p.IntervalSeconds
	if interval == 0 {
		interval = relayDefaultIntervalSeconds
	}
	m := &RelayMonitor{
		Name:            strings.TrimSpace(p.Name),
		System:          p.System,
		BaseURL:         normalizeEndpoint(p.BaseURL),
		Vendor:          strings.TrimSpace(p.Vendor),
		AuthAccount:     strings.TrimSpace(p.AuthAccount),
		Credential:      encrypted, // 传入 repository 时为密文
		WatchedGroups:   normalizeModels(p.WatchedGroups),
		Enabled:         p.Enabled,
		IntervalSeconds: interval,
		CreatedBy:       p.CreatedBy,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create relay monitor: %w", err)
	}
	m.Credential = strings.TrimSpace(p.Credential)
	if s.scheduler != nil {
		s.scheduler.Schedule(m)
	}
	return m, nil
}

// validateRelayCreate 聚拢 Create 入参校验。
func validateRelayCreate(p RelayMonitorCreateParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrRelayMonitorMissingName
	}
	if err := validateRelaySystem(p.System); err != nil {
		return err
	}
	if err := validateRelayInterval(p.IntervalSeconds); err != nil {
		return err
	}
	if err := validateEndpoint(p.BaseURL); err != nil {
		return err
	}
	// sub2api 需要凭证：账号密码模式(邮箱+密码) 或 Token 模式(直接填 JWT)。
	// 两种模式都要求 credential 非空(密码或 token)；auth_account 仅在密码模式填。
	if p.System == RelaySystemSub2API && strings.TrimSpace(p.Credential) == "" {
		return ErrRelayMonitorMissingCredential
	}
	return nil
}

// Update 更新监控。Credential：nil/空 = 不改；非空 = 加密覆盖。
func (s *RelayMonitorService) Update(ctx context.Context, id int64, p RelayMonitorUpdateParams) (*RelayMonitor, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	groupsChanged, err := applyRelayUpdate(existing, p)
	if err != nil {
		return nil, err
	}
	// sub2api 校验：凭证(密码或 token)要么已存在、要么本次提供。
	if existing.System == RelaySystemSub2API {
		credMissing := strings.TrimSpace(existing.Credential) == "" &&
			(p.Credential == nil || strings.TrimSpace(*p.Credential) == "")
		if credMissing {
			return nil, ErrRelayMonitorMissingCredential
		}
	}

	newPlain, credUpdated, err := s.applyCredentialUpdate(existing, p.Credential)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update relay monitor: %w", err)
	}

	// 被跟踪分组缩减时，清理不再跟踪的快照，避免下次误判涨跌。
	if groupsChanged {
		if err := s.repo.DeleteSnapshotsNotIn(ctx, existing.ID, existing.WatchedGroups); err != nil {
			slog.Warn("relay_monitor: prune snapshots failed", "monitor_id", existing.ID, "error", err)
		}
	}

	if credUpdated {
		existing.Credential = newPlain
	} else {
		s.decryptInPlace(existing)
	}
	if s.scheduler != nil {
		s.scheduler.Schedule(existing)
	}
	return existing, nil
}

// applyRelayUpdate 把非 nil 字段应用到 existing；返回 watched_groups 是否变化。
// 凭证字段在调用方单独处理。
func applyRelayUpdate(existing *RelayMonitor, p RelayMonitorUpdateParams) (groupsChanged bool, err error) {
	if p.Name != nil {
		existing.Name = strings.TrimSpace(*p.Name)
	}
	if p.System != nil {
		if err := validateRelaySystem(*p.System); err != nil {
			return false, err
		}
		existing.System = *p.System
	}
	if p.BaseURL != nil {
		if err := validateEndpoint(*p.BaseURL); err != nil {
			return false, err
		}
		existing.BaseURL = normalizeEndpoint(*p.BaseURL)
	}
	if p.Vendor != nil {
		existing.Vendor = strings.TrimSpace(*p.Vendor)
	}
	if p.AuthAccount != nil {
		existing.AuthAccount = strings.TrimSpace(*p.AuthAccount)
	}
	if p.WatchedGroups != nil {
		existing.WatchedGroups = normalizeModels(*p.WatchedGroups)
		groupsChanged = true
	}
	if p.Enabled != nil {
		existing.Enabled = *p.Enabled
	}
	if p.IntervalSeconds != nil {
		if err := validateRelayInterval(*p.IntervalSeconds); err != nil {
			return false, err
		}
		existing.IntervalSeconds = *p.IntervalSeconds
	}
	return groupsChanged, nil
}

// applyCredentialUpdate 处理 Update 中的 Credential 字段（参考 channel_monitor 的 applyAPIKeyUpdate）。
func (s *RelayMonitorService) applyCredentialUpdate(existing *RelayMonitor, raw *string) (plain string, updated bool, err error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return "", false, nil
	}
	plain = strings.TrimSpace(*raw)
	encrypted, encErr := s.encryptor.Encrypt(plain)
	if encErr != nil {
		return "", false, fmt.Errorf("encrypt credential: %w", encErr)
	}
	existing.Credential = encrypted
	return plain, true, nil
}

// Delete 删除监控（快照与历史通过外键 CASCADE 自动清理）。
func (s *RelayMonitorService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete relay monitor: %w", err)
	}
	if s.scheduler != nil {
		s.scheduler.Unschedule(id)
	}
	return nil
}

// ---------- 变化历史 / 汇总 ----------

// ListChanges 查询倍率变化历史（涨/跌公告）。
func (s *RelayMonitorService) ListChanges(ctx context.Context, params RelayRateChangeListParams) ([]*RelayRateChange, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 50
	}
	return s.repo.ListChanges(ctx, params)
}

// Summary 顶部统计卡：涨/跌公告数量（受 search 过滤，忽略分页与 direction）。
func (s *RelayMonitorService) Summary(ctx context.Context, search string) (RelayChangeSummary, error) {
	return s.repo.SummarizeChanges(ctx, RelayRateChangeListParams{Search: strings.TrimSpace(search)})
}

// DeleteChange 删除单条倍率变化历史，避免公告记录长期堆积。
func (s *RelayMonitorService) DeleteChange(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrRelayMonitorNotFound
	}
	return s.repo.DeleteChange(ctx, id)
}

// Overview 倍率总览：所有被跟踪分组的当前倍率，变化过的附带涨跌并排在前面。
func (s *RelayMonitorService) Overview(ctx context.Context, search string) ([]*RelayGroupOverview, error) {
	return s.repo.ListOverview(ctx, strings.TrimSpace(search))
}

// ---------- 探测 ----------

// FetchGroups 用给定配置（或已存在监控的凭证）抓取目标站全部分组+当前倍率，不落库。
// 供前端「拉取分组列表」让用户勾选要监控的分组。
// 当 monitorID > 0 且 credential 为空时，复用该监控已保存的凭证。
func (s *RelayMonitorService) FetchGroups(ctx context.Context, system, baseURL, authAccount, credential string, monitorID int64) ([]RelayGroupRate, error) {
	if err := validateRelaySystem(system); err != nil {
		return nil, err
	}
	if err := validateEndpoint(baseURL); err != nil {
		return nil, err
	}
	account := strings.TrimSpace(authAccount)
	cred := strings.TrimSpace(credential)
	// 密码留空且指定了已存在监控时，复用其已保存的账号/密码。
	if cred == "" && monitorID > 0 {
		existing, err := s.Get(ctx, monitorID)
		if err != nil {
			return nil, err
		}
		if existing.CredentialDecryptFailed {
			return nil, ErrRelayMonitorCredentialDecryptFailed
		}
		cred = existing.Credential
		if account == "" {
			account = existing.AuthAccount
		}
	}
	return probeRelayRates(ctx, system, normalizeEndpoint(baseURL), account, cred)
}

// Probe 对一个监控执行一次探测：抓取倍率 → 对比快照 → 记录涨跌 → 更新快照与 last_checked。
func (s *RelayMonitorService) Probe(ctx context.Context, id int64) (*RelayProbeResult, error) {
	m, err := s.Get(ctx, id) // 已解密凭证
	if err != nil {
		return nil, err
	}
	if m.CredentialDecryptFailed {
		return nil, ErrRelayMonitorCredentialDecryptFailed
	}
	return s.probeMonitor(ctx, m)
}

// probeMonitor 执行探测核心逻辑，monitor 的凭证须已解密。
func (s *RelayMonitorService) probeMonitor(ctx context.Context, m *RelayMonitor) (*RelayProbeResult, error) {
	rates, err := probeRelayRates(ctx, m.System, m.BaseURL, m.AuthAccount, m.Credential)
	if err != nil {
		s.markChecked(ctx, m.ID, probeErrorMessage(err))
		return nil, err
	}

	watched := watchedSet(m.WatchedGroups)
	now := time.Now()
	old, err := s.loadSnapshotMap(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	result := &RelayProbeResult{Rates: make([]RelayGroupRate, 0, len(rates))}
	seen := make(map[string]struct{}, len(rates))
	for _, gr := range rates {
		if len(watched) > 0 {
			if _, ok := watched[gr.GroupName]; !ok {
				continue // 只跟踪选定分组
			}
		}
		seen[gr.GroupName] = struct{}{}
		result.Rates = append(result.Rates, gr)
		if change := buildRateChange(m, gr, old, now); change != nil {
			result.Changes = append(result.Changes, change)
		}
		if err := s.repo.UpsertSnapshot(ctx, m.ID, gr.GroupName, gr.Rate, now); err != nil {
			slog.Warn("relay_monitor: upsert snapshot failed", "monitor_id", m.ID, "group", gr.GroupName, "error", err)
		}
	}
	for groupName, oldRate := range old {
		if len(watched) > 0 {
			if _, ok := watched[groupName]; !ok {
				continue
			}
		}
		if _, ok := seen[groupName]; ok {
			continue
		}
		if oldRate < 0 {
			continue
		}
		change := buildRemovedGroupChange(m, groupName, oldRate, now)
		result.Changes = append(result.Changes, change)
		if err := s.repo.UpsertSnapshot(ctx, m.ID, groupName, -1, now); err != nil {
			slog.Warn("relay_monitor: mark removed snapshot failed", "monitor_id", m.ID, "group", groupName, "error", err)
		}
	}

	s.persistChanges(ctx, m, result.Changes)
	s.markChecked(ctx, m.ID, "")
	return result, nil
}

// buildRateChange 比对单个分组的新旧倍率，变化时构造一条变化记录；无变化或首次见到返回 nil。
func buildRateChange(m *RelayMonitor, gr RelayGroupRate, old map[string]float64, now time.Time) *RelayRateChange {
	prev, seen := old[gr.GroupName]
	if !seen || ratesEqual(prev, gr.Rate) {
		return nil
	}
	direction := RelayDirectionUp
	if gr.Rate < prev {
		direction = RelayDirectionDown
	}
	return &RelayRateChange{
		MonitorID:  m.ID,
		Site:       m.Name,
		System:     m.System,
		Vendor:     m.Vendor,
		GroupName:  gr.GroupName,
		OldRate:    prev,
		NewRate:    gr.Rate,
		Direction:  direction,
		Content:    fmt.Sprintf("分组倍率从 %s 变为 %s", formatRate(prev), formatRate(gr.Rate)),
		DetectedAt: now,
	}
}

func buildRemovedGroupChange(m *RelayMonitor, groupName string, oldRate float64, now time.Time) *RelayRateChange {
	return &RelayRateChange{
		MonitorID:  m.ID,
		Site:       m.Name,
		System:     m.System,
		Vendor:     m.Vendor,
		GroupName:  groupName,
		OldRate:    oldRate,
		NewRate:    0,
		Direction:  RelayDirectionDown,
		Content:    "分组已停用",
		DetectedAt: now,
	}
}

// persistChanges 写入变化记录并裁剪历史。无变化时直接返回。
func (s *RelayMonitorService) persistChanges(ctx context.Context, m *RelayMonitor, changes []*RelayRateChange) {
	if len(changes) == 0 {
		return
	}
	if err := s.repo.InsertChanges(ctx, changes); err != nil {
		slog.Error("relay_monitor: insert changes failed", "monitor_id", m.ID, "error", err)
		return
	}
	if err := s.repo.PruneChangesForMonitor(ctx, m.ID, relayChangeRetentionPerMonitor); err != nil {
		slog.Warn("relay_monitor: prune changes failed", "monitor_id", m.ID, "error", err)
	}
}

// markChecked 更新 last_checked_at 与 last_error（lastErr 为空表示成功）。
func (s *RelayMonitorService) markChecked(ctx context.Context, id int64, lastErr string) {
	if err := s.repo.MarkChecked(ctx, id, time.Now(), lastErr); err != nil {
		slog.Warn("relay_monitor: mark checked failed", "monitor_id", id, "error", err)
	}
}

// loadSnapshotMap 加载某监控的当前快照为 group->rate map。
func (s *RelayMonitorService) loadSnapshotMap(ctx context.Context, monitorID int64) (map[string]float64, error) {
	snaps, err := s.repo.ListSnapshots(ctx, monitorID)
	if err != nil {
		return nil, fmt.Errorf("load snapshots: %w", err)
	}
	out := make(map[string]float64, len(snaps))
	for _, sn := range snaps {
		out[sn.GroupName] = sn.Rate
	}
	return out, nil
}

// ---------- 调度器协作 ----------

// RelayScheduler 调度器接口，CRUD 后回调同步定时任务（setter 注入避免依赖环）。
type RelayScheduler interface {
	Schedule(m *RelayMonitor)
	Unschedule(id int64)
}

// SetScheduler 由 wire 在 runner 构造后注入。
func (s *RelayMonitorService) SetScheduler(sched RelayScheduler) {
	s.scheduler = sched
}

// ListEnabledMonitors 返回所有 enabled 监控（解密后），供 runner 启动建立任务表。
func (s *RelayMonitorService) ListEnabledMonitors(ctx context.Context) ([]*RelayMonitor, error) {
	all, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	for _, m := range all {
		s.decryptInPlace(m)
	}
	return all, nil
}

// RunProbe 供 runner 调用：按 id 探测一次（凭证内部解密）。
func (s *RelayMonitorService) RunProbe(ctx context.Context, id int64) error {
	_, err := s.Probe(ctx, id)
	return err
}

// ---------- helpers ----------

// encryptCredential 加密凭证；空串原样返回空串（newapi 站点无凭证）。
func (s *RelayMonitorService) encryptCredential(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	enc, err := s.encryptor.Encrypt(plain)
	if err != nil {
		return "", fmt.Errorf("encrypt credential: %w", err)
	}
	return enc, nil
}

// decryptInPlace 把 Credential 从密文解密为明文。空凭证直接跳过；
// 解密失败时清空并置 CredentialDecryptFailed=true（不报错，避免阻断列表）。
func (s *RelayMonitorService) decryptInPlace(m *RelayMonitor) {
	if m == nil || m.Credential == "" {
		return
	}
	plain, err := s.encryptor.Decrypt(m.Credential)
	if err != nil {
		slog.Warn("relay_monitor: decrypt credential failed", "monitor_id", m.ID, "error", err)
		m.Credential = ""
		m.CredentialDecryptFailed = true
		return
	}
	m.Credential = plain
}

// validateRelaySystem 校验 system 字符串。
func validateRelaySystem(sys string) error {
	switch sys {
	case RelaySystemSub2API, RelaySystemNewAPI:
		return nil
	default:
		return ErrRelayMonitorInvalidSystem
	}
}

// validateRelayInterval 校验探测间隔；0 视为使用默认值（由调用方填充），不报错。
func validateRelayInterval(sec int) error {
	if sec == 0 {
		return nil
	}
	if sec < relayMinIntervalSeconds || sec > relayMaxIntervalSeconds {
		return ErrRelayMonitorInvalidInterval
	}
	return nil
}

// watchedSet 把 watched_groups 列表转成集合；空列表返回空 map（调用方据此判断是否全跟踪）。
func watchedSet(groups []string) map[string]struct{} {
	out := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g != "" {
			out[g] = struct{}{}
		}
	}
	return out
}

// ratesEqual 浮点倍率相等判断（容差 1e-9，避免浮点抖动误报涨跌）。
func ratesEqual(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-9
}

// formatRate 把倍率格式化为 "0.005x" 形式（去掉多余尾零）。
func formatRate(r float64) string {
	s := strconv.FormatFloat(r, 'f', -1, 64)
	return s + "x"
}

// probeErrorMessage 把探测错误截断为存库用的简短信息。
func probeErrorMessage(err error) string {
	msg := err.Error()
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return msg
}
