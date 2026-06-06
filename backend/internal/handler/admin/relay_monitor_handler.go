package admin

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	// relayMonitorMaxPageSize 列表分页上限。
	relayMonitorMaxPageSize = 100
	// relayCredentialMaskPrefix 凭证脱敏保留的明文前缀长度。
	relayCredentialMaskPrefix = 4
	// relayCredentialMaskSuffix 脱敏占位串。
	relayCredentialMaskSuffix = "***"
)

// RelayMonitorHandler 中转站监控管理后台 handler。
type RelayMonitorHandler struct {
	monitorService *service.RelayMonitorService
}

// NewRelayMonitorHandler 创建 handler。
func NewRelayMonitorHandler(monitorService *service.RelayMonitorService) *RelayMonitorHandler {
	return &RelayMonitorHandler{monitorService: monitorService}
}

// --- Request / Response ---

type relayMonitorCreateRequest struct {
	Name            string   `json:"name" binding:"required,max=100"`
	System          string   `json:"system" binding:"required,oneof=sub2api newapi"`
	BaseURL         string   `json:"base_url" binding:"required,max=500"`
	Vendor          string   `json:"vendor" binding:"max=50"`
	AuthAccount     string   `json:"auth_account" binding:"max=200"`
	Credential      string   `json:"credential" binding:"max=4000"`
	WatchedGroups   []string `json:"watched_groups"`
	Enabled         *bool    `json:"enabled"`
	IntervalSeconds int      `json:"interval_seconds" binding:"omitempty,min=60,max=86400"`
}

type relayMonitorUpdateRequest struct {
	Name            *string   `json:"name" binding:"omitempty,max=100"`
	System          *string   `json:"system" binding:"omitempty,oneof=sub2api newapi"`
	BaseURL         *string   `json:"base_url" binding:"omitempty,max=500"`
	Vendor          *string   `json:"vendor" binding:"omitempty,max=50"`
	AuthAccount     *string   `json:"auth_account" binding:"omitempty,max=200"`
	Credential      *string   `json:"credential" binding:"omitempty,max=4000"`
	WatchedGroups   *[]string `json:"watched_groups"`
	Enabled         *bool     `json:"enabled"`
	IntervalSeconds *int      `json:"interval_seconds" binding:"omitempty,min=60,max=86400"`
}

type relayFetchGroupsRequest struct {
	System      string `json:"system" binding:"required,oneof=sub2api newapi"`
	BaseURL     string `json:"base_url" binding:"required,max=500"`
	AuthAccount string `json:"auth_account" binding:"max=200"`
	Credential  string `json:"credential" binding:"max=4000"`
	MonitorID   int64  `json:"monitor_id"`
}

type relayMonitorResponse struct {
	ID                      int64    `json:"id"`
	Name                    string   `json:"name"`
	System                  string   `json:"system"`
	BaseURL                 string   `json:"base_url"`
	Vendor                  string   `json:"vendor"`
	AuthAccount             string   `json:"auth_account"`
	CredentialMasked        string   `json:"credential_masked"`
	HasCredential           bool     `json:"has_credential"`
	CredentialDecryptFailed bool     `json:"credential_decrypt_failed"`
	WatchedGroups           []string `json:"watched_groups"`
	Enabled                 bool     `json:"enabled"`
	IntervalSeconds         int      `json:"interval_seconds"`
	LastCheckedAt           *string  `json:"last_checked_at"`
	LastError               string   `json:"last_error"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
}

type relayGroupRateResponse struct {
	GroupName string  `json:"group_name"`
	Rate      float64 `json:"rate"`
}

type relayRateChangeResponse struct {
	ID         int64   `json:"id"`
	MonitorID  int64   `json:"monitor_id"`
	Site       string  `json:"site"`
	System     string  `json:"system"`
	Vendor     string  `json:"vendor"`
	GroupName  string  `json:"group_name"`
	OldRate    float64 `json:"old_rate"`
	NewRate    float64 `json:"new_rate"`
	Direction  string  `json:"direction"`
	Content    string  `json:"content"`
	DetectedAt string  `json:"detected_at"`
}

type relayProbeResultResponse struct {
	Rates   []relayGroupRateResponse  `json:"rates"`
	Changes []relayRateChangeResponse `json:"changes"`
}

// maskCredential 对凭证明文脱敏：前 4 字符 + "***"；空串返回空串。
func maskCredential(plain string) string {
	if plain == "" {
		return ""
	}
	if len(plain) <= relayCredentialMaskPrefix {
		return relayCredentialMaskSuffix
	}
	return plain[:relayCredentialMaskPrefix] + relayCredentialMaskSuffix
}

func relayMonitorToResponse(m *service.RelayMonitor) *relayMonitorResponse {
	if m == nil {
		return nil
	}
	groups := m.WatchedGroups
	if groups == nil {
		groups = []string{}
	}
	resp := &relayMonitorResponse{
		ID:                      m.ID,
		Name:                    m.Name,
		System:                  m.System,
		BaseURL:                 m.BaseURL,
		Vendor:                  m.Vendor,
		AuthAccount:             m.AuthAccount,
		CredentialMasked:        maskCredential(m.Credential),
		HasCredential:           m.Credential != "" || m.CredentialDecryptFailed,
		CredentialDecryptFailed: m.CredentialDecryptFailed,
		WatchedGroups:           groups,
		Enabled:                 m.Enabled,
		IntervalSeconds:         m.IntervalSeconds,
		LastError:               m.LastError,
		CreatedAt:               m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               m.UpdatedAt.Format(time.RFC3339),
	}
	if m.LastCheckedAt != nil {
		s := m.LastCheckedAt.Format(time.RFC3339)
		resp.LastCheckedAt = &s
	}
	return resp
}

func relayChangeToResponse(c *service.RelayRateChange) relayRateChangeResponse {
	return relayRateChangeResponse{
		ID:         c.ID,
		MonitorID:  c.MonitorID,
		Site:       c.Site,
		System:     c.System,
		Vendor:     c.Vendor,
		GroupName:  c.GroupName,
		OldRate:    c.OldRate,
		NewRate:    c.NewRate,
		Direction:  c.Direction,
		Content:    c.Content,
		DetectedAt: c.DetectedAt.Format(time.RFC3339),
	}
}

func relayProbeResultToResponse(r *service.RelayProbeResult) relayProbeResultResponse {
	out := relayProbeResultResponse{
		Rates:   make([]relayGroupRateResponse, 0, len(r.Rates)),
		Changes: make([]relayRateChangeResponse, 0, len(r.Changes)),
	}
	for _, gr := range r.Rates {
		out.Rates = append(out.Rates, relayGroupRateResponse{GroupName: gr.GroupName, Rate: gr.Rate})
	}
	for _, ch := range r.Changes {
		out.Changes = append(out.Changes, relayChangeToResponse(ch))
	}
	return out
}

// parseRelayMonitorID 提取并校验路径参数 :id。
func parseRelayMonitorID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_RELAY_MONITOR_ID", "invalid relay monitor id"))
		return 0, false
	}
	return id, true
}

// --- Handlers ---

// List GET /api/v1/admin/relay-monitors
func (h *RelayMonitorHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > relayMonitorMaxPageSize {
		pageSize = relayMonitorMaxPageSize
	}
	params := service.RelayMonitorListParams{
		Page:     page,
		PageSize: pageSize,
		System:   strings.TrimSpace(c.Query("system")),
		Enabled:  parseListEnabled(c.Query("enabled")),
		Search:   strings.TrimSpace(c.Query("search")),
	}
	items, total, err := h.monitorService.List(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*relayMonitorResponse, 0, len(items))
	for _, m := range items {
		out = append(out, relayMonitorToResponse(m))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// Get GET /api/v1/admin/relay-monitors/:id
func (h *RelayMonitorHandler) Get(c *gin.Context) {
	id, ok := parseRelayMonitorID(c)
	if !ok {
		return
	}
	m, err := h.monitorService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, relayMonitorToResponse(m))
}

// Create POST /api/v1/admin/relay-monitors
func (h *RelayMonitorHandler) Create(c *gin.Context) {
	var req relayMonitorCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	m, err := h.monitorService.Create(c.Request.Context(), service.RelayMonitorCreateParams{
		Name:            req.Name,
		System:          req.System,
		BaseURL:         req.BaseURL,
		Vendor:          req.Vendor,
		AuthAccount:     req.AuthAccount,
		Credential:      req.Credential,
		WatchedGroups:   req.WatchedGroups,
		Enabled:         enabled,
		IntervalSeconds: req.IntervalSeconds,
		CreatedBy:       subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, relayMonitorToResponse(m))
}

// Update PUT /api/v1/admin/relay-monitors/:id
func (h *RelayMonitorHandler) Update(c *gin.Context) {
	id, ok := parseRelayMonitorID(c)
	if !ok {
		return
	}
	var req relayMonitorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	m, err := h.monitorService.Update(c.Request.Context(), id, service.RelayMonitorUpdateParams{
		Name:            req.Name,
		System:          req.System,
		BaseURL:         req.BaseURL,
		Vendor:          req.Vendor,
		AuthAccount:     req.AuthAccount,
		Credential:      req.Credential,
		WatchedGroups:   req.WatchedGroups,
		Enabled:         req.Enabled,
		IntervalSeconds: req.IntervalSeconds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, relayMonitorToResponse(m))
}

// Delete DELETE /api/v1/admin/relay-monitors/:id
func (h *RelayMonitorHandler) Delete(c *gin.Context) {
	id, ok := parseRelayMonitorID(c)
	if !ok {
		return
	}
	if err := h.monitorService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Probe POST /api/v1/admin/relay-monitors/:id/probe
func (h *RelayMonitorHandler) Probe(c *gin.Context) {
	id, ok := parseRelayMonitorID(c)
	if !ok {
		return
	}
	result, err := h.monitorService.Probe(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, relayProbeResultToResponse(result))
}

// ProbeAll POST /api/v1/admin/relay-monitors/probe-all
// 对全部 enabled 监控逐一探测，返回每站的检测到的变化数量与错误。
func (h *RelayMonitorHandler) ProbeAll(c *gin.Context) {
	ctx := c.Request.Context()
	monitors, err := h.monitorService.ListEnabledMonitors(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	type probeAllItem struct {
		MonitorID int64  `json:"monitor_id"`
		Name      string `json:"name"`
		Changes   int    `json:"changes"`
		Error     string `json:"error,omitempty"`
	}
	results := make([]probeAllItem, 0, len(monitors))
	for _, m := range monitors {
		item := probeAllItem{MonitorID: m.ID, Name: m.Name}
		res, perr := h.monitorService.Probe(ctx, m.ID)
		if perr != nil {
			item.Error = perr.Error()
		} else {
			item.Changes = len(res.Changes)
		}
		results = append(results, item)
	}
	response.Success(c, gin.H{"probed": len(results), "results": results})
}

// FetchGroups POST /api/v1/admin/relay-monitors/fetch-groups
// 用给定配置抓取目标站全部分组+当前倍率（不落库），供前端勾选要监控的分组。
func (h *RelayMonitorHandler) FetchGroups(c *gin.Context) {
	var req relayFetchGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	rates, err := h.monitorService.FetchGroups(c.Request.Context(), req.System, req.BaseURL, req.AuthAccount, req.Credential, req.MonitorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]relayGroupRateResponse, 0, len(rates))
	for _, gr := range rates {
		out = append(out, relayGroupRateResponse{GroupName: gr.GroupName, Rate: gr.Rate})
	}
	response.Success(c, out)
}

// Changes GET /api/v1/admin/relay-monitors/changes
// 倍率变化历史（涨/跌公告），支持 direction/search/monitor_id 过滤 + 分页。
func (h *RelayMonitorHandler) Changes(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > relayMonitorMaxPageSize {
		pageSize = relayMonitorMaxPageSize
	}
	var monitorID int64
	if raw := strings.TrimSpace(c.Query("monitor_id")); raw != "" {
		monitorID, _ = strconv.ParseInt(raw, 10, 64)
	}
	direction := strings.ToLower(strings.TrimSpace(c.Query("direction")))
	if direction != service.RelayDirectionUp && direction != service.RelayDirectionDown {
		direction = ""
	}
	params := service.RelayRateChangeListParams{
		MonitorID: monitorID,
		Direction: direction,
		Search:    strings.TrimSpace(c.Query("search")),
		Page:      page,
		PageSize:  pageSize,
	}
	items, total, err := h.monitorService.ListChanges(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]relayRateChangeResponse, 0, len(items))
	for _, ch := range items {
		out = append(out, relayChangeToResponse(ch))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// Summary GET /api/v1/admin/relay-monitors/summary
// 顶部统计卡：涨/跌公告数量（受 search 过滤）。
func (h *RelayMonitorHandler) Summary(c *gin.Context) {
	summary, err := h.monitorService.Summary(c.Request.Context(), strings.TrimSpace(c.Query("search")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"up_count":   summary.UpCount,
		"down_count": summary.DownCount,
	})
}
