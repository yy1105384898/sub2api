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

const cardMonitorMaxPageSize = 100

type CardPlatformMonitorHandler struct {
	svc *service.CardPlatformMonitorService
}

func NewCardPlatformMonitorHandler(svc *service.CardPlatformMonitorService) *CardPlatformMonitorHandler {
	return &CardPlatformMonitorHandler{svc: svc}
}

type cardMonitorRequest struct {
	Name            string `json:"name" binding:"required,max=100"`
	PlatformType    string `json:"platform_type" binding:"omitempty,oneof=ldxp"`
	BaseURL         string `json:"base_url" binding:"required,max=500"`
	ShopURL         string `json:"shop_url" binding:"max=500"`
	AuthMode        string `json:"auth_mode" binding:"omitempty,oneof=public token cookie"`
	Credential      string `json:"credential" binding:"max=8000"`
	Enabled         *bool  `json:"enabled"`
	IntervalSeconds int    `json:"interval_seconds" binding:"omitempty,min=60,max=86400"`
	FetchPages      int    `json:"fetch_pages" binding:"omitempty,min=1,max=500"`
	Note            string `json:"note" binding:"max=500"`
}

type cardMonitorUpdateRequest struct {
	Name            *string `json:"name" binding:"omitempty,max=100"`
	PlatformType    *string `json:"platform_type" binding:"omitempty,oneof=ldxp"`
	BaseURL         *string `json:"base_url" binding:"omitempty,max=500"`
	ShopURL         *string `json:"shop_url" binding:"omitempty,max=500"`
	AuthMode        *string `json:"auth_mode" binding:"omitempty,oneof=public token cookie"`
	Credential      *string `json:"credential" binding:"omitempty,max=8000"`
	Enabled         *bool   `json:"enabled"`
	IntervalSeconds *int    `json:"interval_seconds" binding:"omitempty,min=60,max=86400"`
	FetchPages      *int    `json:"fetch_pages" binding:"omitempty,min=1,max=500"`
	Note            *string `json:"note" binding:"omitempty,max=500"`
}

func parseCardMonitorID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_CARD_MONITOR_ID", "invalid card monitor id"))
		return 0, false
	}
	return id, true
}

func (h *CardPlatformMonitorHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > cardMonitorMaxPageSize {
		pageSize = cardMonitorMaxPageSize
	}
	items, total, err := h.svc.List(c.Request.Context(), service.CardMonitorListParams{
		PlatformType: strings.TrimSpace(c.Query("platform_type")),
		Enabled:      parseListEnabled(c.Query("enabled")),
		Search:       strings.TrimSpace(c.Query("search")),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, cardMonitorsToResponse(items), total, page, pageSize)
}

func (h *CardPlatformMonitorHandler) Get(c *gin.Context) {
	id, ok := parseCardMonitorID(c)
	if !ok {
		return
	}
	m, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cardMonitorToResponse(m))
}

func (h *CardPlatformMonitorHandler) Create(c *gin.Context) {
	var req cardMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	m, err := h.svc.Create(c.Request.Context(), service.CardMonitorCreateParams{
		Name: req.Name, PlatformType: req.PlatformType, BaseURL: req.BaseURL, ShopURL: req.ShopURL,
		AuthMode: req.AuthMode, Credential: req.Credential, Enabled: enabled,
		IntervalSeconds: req.IntervalSeconds, FetchPages: req.FetchPages, Note: req.Note, CreatedBy: subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, cardMonitorToResponse(m))
}

func (h *CardPlatformMonitorHandler) Update(c *gin.Context) {
	id, ok := parseCardMonitorID(c)
	if !ok {
		return
	}
	var req cardMonitorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	m, err := h.svc.Update(c.Request.Context(), id, service.CardMonitorUpdateParams{
		Name: req.Name, PlatformType: req.PlatformType, BaseURL: req.BaseURL, ShopURL: req.ShopURL,
		AuthMode: req.AuthMode, Credential: req.Credential, Enabled: req.Enabled,
		IntervalSeconds: req.IntervalSeconds, FetchPages: req.FetchPages, Note: req.Note,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cardMonitorToResponse(m))
}

func (h *CardPlatformMonitorHandler) Delete(c *gin.Context) {
	id, ok := parseCardMonitorID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CardPlatformMonitorHandler) Probe(c *gin.Context) {
	id, ok := parseCardMonitorID(c)
	if !ok {
		return
	}
	result, err := h.svc.Probe(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CardPlatformMonitorHandler) ProbeAll(c *gin.Context) {
	items, err := h.svc.ListEnabledMonitors(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	type item struct {
		MonitorID int64  `json:"monitor_id"`
		Name      string `json:"name"`
		Products  int    `json:"products"`
		Events    int    `json:"events"`
		Error     string `json:"error,omitempty"`
	}
	out := make([]item, 0, len(items))
	for _, m := range items {
		row := item{MonitorID: m.ID, Name: m.Name}
		res, perr := h.svc.Probe(c.Request.Context(), m.ID)
		if perr != nil {
			row.Error = perr.Error()
		} else {
			row.Products = len(res.Products)
			row.Events = len(res.Events)
		}
		out = append(out, row)
	}
	response.Success(c, gin.H{"probed": len(out), "results": out})
}

func (h *CardPlatformMonitorHandler) Products(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > cardMonitorMaxPageSize {
		pageSize = cardMonitorMaxPageSize
	}
	monitorID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("monitor_id")), 10, 64)
	items, total, err := h.svc.SearchProducts(c.Request.Context(), service.CardProductSearchParams{
		Search: strings.TrimSpace(c.Query("search")), MonitorID: monitorID,
		PlatformType: strings.TrimSpace(c.Query("platform_type")), Status: strings.TrimSpace(c.Query("status")),
		InStockOnly: c.Query("in_stock") == "true", Sort: strings.TrimSpace(c.Query("sort")),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *CardPlatformMonitorHandler) Events(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > cardMonitorMaxPageSize {
		pageSize = cardMonitorMaxPageSize
	}
	monitorID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("monitor_id")), 10, 64)
	items, total, err := h.svc.ListEvents(c.Request.Context(), service.CardEventListParams{
		MonitorID: monitorID, EventType: strings.TrimSpace(c.Query("event_type")),
		Search: strings.TrimSpace(c.Query("search")), Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *CardPlatformMonitorHandler) Summary(c *gin.Context) {
	s, err := h.svc.Summary(c.Request.Context(), strings.TrimSpace(c.Query("search")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, s)
}

func cardMonitorToResponse(m *service.CardPlatformMonitor) gin.H {
	if m == nil {
		return gin.H{}
	}
	var lastChecked *string
	if m.LastCheckedAt != nil {
		s := m.LastCheckedAt.Format(time.RFC3339)
		lastChecked = &s
	}
	return gin.H{
		"id": m.ID, "name": m.Name, "platform_type": m.PlatformType, "base_url": m.BaseURL, "shop_url": m.ShopURL,
		"auth_mode": m.AuthMode, "credential_masked": maskCredential(m.Credential),
		"has_credential":            m.Credential != "" || m.CredentialDecryptFailed,
		"credential_decrypt_failed": m.CredentialDecryptFailed,
		"enabled":                   m.Enabled, "interval_seconds": m.IntervalSeconds, "fetch_pages": m.FetchPages,
		"last_checked_at": lastChecked, "last_error": m.LastError, "note": m.Note,
		"created_at": m.CreatedAt.Format(time.RFC3339), "updated_at": m.UpdatedAt.Format(time.RFC3339),
	}
}

func cardMonitorsToResponse(items []*service.CardPlatformMonitor) []gin.H {
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, cardMonitorToResponse(item))
	}
	return out
}
