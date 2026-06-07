package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type CardPlatformMonitorService struct {
	repo      CardPlatformMonitorRepository
	encryptor SecretEncryptor
	scheduler CardPlatformScheduler
}

func NewCardPlatformMonitorService(repo CardPlatformMonitorRepository, encryptor SecretEncryptor) *CardPlatformMonitorService {
	return &CardPlatformMonitorService{repo: repo, encryptor: encryptor}
}

func (s *CardPlatformMonitorService) SetScheduler(scheduler CardPlatformScheduler) {
	s.scheduler = scheduler
}

func (s *CardPlatformMonitorService) List(ctx context.Context, params CardMonitorListParams) ([]*CardPlatformMonitor, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 20
	}
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range items {
		s.decryptInPlace(item)
	}
	return items, total, nil
}

func (s *CardPlatformMonitorService) Get(ctx context.Context, id int64) (*CardPlatformMonitor, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.decryptInPlace(m)
	return m, nil
}

func (s *CardPlatformMonitorService) Create(ctx context.Context, p CardMonitorCreateParams) (*CardPlatformMonitor, error) {
	if err := validateCardCreate(p); err != nil {
		return nil, err
	}
	encrypted, err := s.encryptCredential(strings.TrimSpace(p.Credential))
	if err != nil {
		return nil, err
	}
	interval := p.IntervalSeconds
	if interval == 0 {
		interval = cardDefaultIntervalSeconds
	}
	pages := p.FetchPages
	if pages == 0 {
		pages = cardDefaultFetchPages
	}
	m := &CardPlatformMonitor{
		Name:            strings.TrimSpace(p.Name),
		PlatformType:    normalizeCardPlatform(p.PlatformType),
		BaseURL:         normalizeEndpoint(p.BaseURL),
		ShopURL:         strings.TrimSpace(p.ShopURL),
		AuthMode:        normalizeCardAuthMode(p.AuthMode),
		Credential:      encrypted,
		Enabled:         p.Enabled,
		IntervalSeconds: interval,
		FetchPages:      pages,
		Note:            strings.TrimSpace(p.Note),
		CreatedBy:       p.CreatedBy,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	m.Credential = strings.TrimSpace(p.Credential)
	if s.scheduler != nil {
		s.scheduler.Schedule(m)
	}
	return m, nil
}

func (s *CardPlatformMonitorService) Update(ctx context.Context, id int64, p CardMonitorUpdateParams) (*CardPlatformMonitor, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyCardUpdate(m, p); err != nil {
		return nil, err
	}
	if needsCardCredential(m.AuthMode) && strings.TrimSpace(m.Credential) == "" && (p.Credential == nil || strings.TrimSpace(*p.Credential) == "") {
		return nil, ErrCardMonitorMissingCredential
	}
	newPlain, credUpdated, err := s.applyCredentialUpdate(m, p.Credential)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	if credUpdated {
		m.Credential = newPlain
	} else {
		s.decryptInPlace(m)
	}
	if s.scheduler != nil {
		s.scheduler.Schedule(m)
	}
	return m, nil
}

func (s *CardPlatformMonitorService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.scheduler != nil {
		s.scheduler.Unschedule(id)
	}
	return nil
}

func (s *CardPlatformMonitorService) SearchProducts(ctx context.Context, params CardProductSearchParams) ([]*CardProductSnapshot, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 30
	}
	return s.repo.ListProducts(ctx, params)
}

func (s *CardPlatformMonitorService) ListEvents(ctx context.Context, params CardEventListParams) ([]*CardPriceEvent, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 50
	}
	return s.repo.ListEvents(ctx, params)
}

func (s *CardPlatformMonitorService) Summary(ctx context.Context, search string) (CardSummary, error) {
	return s.repo.Summary(ctx, strings.TrimSpace(search))
}

func (s *CardPlatformMonitorService) ListEnabledMonitors(ctx context.Context) ([]*CardPlatformMonitor, error) {
	items, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		s.decryptInPlace(item)
	}
	return items, nil
}

func (s *CardPlatformMonitorService) Probe(ctx context.Context, id int64) (*CardProbeResult, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.CredentialDecryptFailed {
		return nil, ErrCardMonitorCredentialDecryptFailed
	}
	return s.probeMonitor(ctx, m)
}

func (s *CardPlatformMonitorService) RunProbe(ctx context.Context, id int64) error {
	_, err := s.Probe(ctx, id)
	return err
}

func (s *CardPlatformMonitorService) probeMonitor(ctx context.Context, m *CardPlatformMonitor) (*CardProbeResult, error) {
	products, raws, err := probeCardProducts(ctx, m)
	if err != nil {
		s.markChecked(ctx, m.ID, err.Error())
		return nil, err
	}
	now := time.Now()
	result := &CardProbeResult{Products: make([]*CardProductSnapshot, 0, len(products))}
	for idx, p := range products {
		old, findErr := s.repo.FindProduct(ctx, m.ID, p.ExternalProductID)
		if findErr != nil {
			slog.Warn("card_monitor: find old product failed", "monitor_id", m.ID, "product", p.ExternalProductID, "error", findErr)
		}
		events := buildCardEvents(m, old, p, now)
		raw := []byte(`{}`)
		if idx < len(raws) {
			raw = raws[idx]
		}
		if err := s.repo.UpsertProduct(ctx, p, raw, now); err != nil {
			slog.Warn("card_monitor: upsert product failed", "monitor_id", m.ID, "product", p.ExternalProductID, "error", err)
			continue
		}
		for _, ev := range events {
			productID := p.ID
			ev.ProductID = &productID
		}
		if err := s.repo.InsertEvents(ctx, events); err != nil {
			slog.Warn("card_monitor: insert events failed", "monitor_id", m.ID, "product", p.ExternalProductID, "error", err)
		}
		result.Products = append(result.Products, p)
		result.Events = append(result.Events, events...)
	}
	if err := s.repo.PruneEventsForMonitor(ctx, m.ID, cardEventRetentionPerMonitor); err != nil {
		slog.Warn("card_monitor: prune events failed", "monitor_id", m.ID, "error", err)
	}
	s.markChecked(ctx, m.ID, "")
	return result, nil
}

func buildCardEvents(m *CardPlatformMonitor, old, next *CardProductSnapshot, at time.Time) []*CardPriceEvent {
	if next == nil {
		return nil
	}
	if old == nil {
		return []*CardPriceEvent{newCardEvent(m, next, CardEventNewProduct, nil, preferredPrice(next), nil, next.Stock, "发现新商品", at)}
	}
	events := make([]*CardPriceEvent, 0, 3)
	oldPrice := preferredPrice(old)
	newPrice := preferredPrice(next)
	if oldPrice != nil && newPrice != nil && !floatEqual(*oldPrice, *newPrice) {
		eventType := CardEventPriceUp
		content := "价格上涨"
		if *newPrice < *oldPrice {
			eventType = CardEventPriceDown
			content = "价格下降"
			if old.LowestPrice == nil || *newPrice < *old.LowestPrice {
				eventType = CardEventNewLow
				content = "刷新历史低价"
			}
		}
		events = append(events, newCardEvent(m, next, eventType, oldPrice, newPrice, old.Stock, next.Stock, content, at))
	}
	if old.Stock != nil && next.Stock != nil {
		if *old.Stock <= 0 && *next.Stock > 0 {
			events = append(events, newCardEvent(m, next, CardEventRestock, oldPrice, newPrice, old.Stock, next.Stock, "商品补货", at))
		} else if *old.Stock > 0 && *next.Stock <= 0 {
			events = append(events, newCardEvent(m, next, CardEventSoldOut, oldPrice, newPrice, old.Stock, next.Stock, "商品售罄", at))
		}
	}
	if old.Status != next.Status {
		if next.Status == "offline" {
			events = append(events, newCardEvent(m, next, CardEventOffline, oldPrice, newPrice, old.Stock, next.Stock, "商品下架", at))
		} else if old.Status == "offline" && next.Status == "online" {
			events = append(events, newCardEvent(m, next, CardEventOnline, oldPrice, newPrice, old.Stock, next.Stock, "商品重新上架", at))
		}
	}
	return events
}

func newCardEvent(m *CardPlatformMonitor, p *CardProductSnapshot, typ string, oldPrice, newPrice *float64, oldStock, newStock *int64, content string, at time.Time) *CardPriceEvent {
	return &CardPriceEvent{
		MonitorID:  m.ID,
		Platform:   m.Name,
		EventType:  typ,
		Title:      p.Title,
		OldPrice:   oldPrice,
		NewPrice:   newPrice,
		OldStock:   oldStock,
		NewStock:   newStock,
		Content:    content,
		DetectedAt: at,
	}
}

func preferredPrice(p *CardProductSnapshot) *float64 {
	if p == nil {
		return nil
	}
	if p.CostPrice != nil {
		return p.CostPrice
	}
	return p.Price
}

func floatEqual(a, b float64) bool {
	if a > b {
		return a-b < 0.000001
	}
	return b-a < 0.000001
}

func validateCardCreate(p CardMonitorCreateParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrCardMonitorMissingName
	}
	if err := validateCardPlatform(p.PlatformType); err != nil {
		return err
	}
	if err := validateCardAuthMode(p.AuthMode); err != nil {
		return err
	}
	if err := validateEndpoint(p.BaseURL); err != nil {
		return err
	}
	if err := validateCardInterval(p.IntervalSeconds); err != nil {
		return err
	}
	if err := validateCardFetchPages(p.FetchPages); err != nil {
		return err
	}
	if needsCardCredential(normalizeCardAuthMode(p.AuthMode)) && strings.TrimSpace(p.Credential) == "" {
		return ErrCardMonitorMissingCredential
	}
	return nil
}

func applyCardUpdate(m *CardPlatformMonitor, p CardMonitorUpdateParams) error {
	if p.Name != nil {
		m.Name = strings.TrimSpace(*p.Name)
	}
	if p.PlatformType != nil {
		if err := validateCardPlatform(*p.PlatformType); err != nil {
			return err
		}
		m.PlatformType = normalizeCardPlatform(*p.PlatformType)
	}
	if p.BaseURL != nil {
		if err := validateEndpoint(*p.BaseURL); err != nil {
			return err
		}
		m.BaseURL = normalizeEndpoint(*p.BaseURL)
	}
	if p.ShopURL != nil {
		m.ShopURL = strings.TrimSpace(*p.ShopURL)
	}
	if p.AuthMode != nil {
		if err := validateCardAuthMode(*p.AuthMode); err != nil {
			return err
		}
		m.AuthMode = normalizeCardAuthMode(*p.AuthMode)
	}
	if p.Enabled != nil {
		m.Enabled = *p.Enabled
	}
	if p.IntervalSeconds != nil {
		if err := validateCardInterval(*p.IntervalSeconds); err != nil {
			return err
		}
		m.IntervalSeconds = *p.IntervalSeconds
	}
	if p.FetchPages != nil {
		if err := validateCardFetchPages(*p.FetchPages); err != nil {
			return err
		}
		m.FetchPages = *p.FetchPages
	}
	if p.Note != nil {
		m.Note = strings.TrimSpace(*p.Note)
	}
	return nil
}

func (s *CardPlatformMonitorService) applyCredentialUpdate(m *CardPlatformMonitor, raw *string) (string, bool, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return "", false, nil
	}
	plain := strings.TrimSpace(*raw)
	encrypted, err := s.encryptor.Encrypt(plain)
	if err != nil {
		return "", false, fmt.Errorf("encrypt card credential: %w", err)
	}
	m.Credential = encrypted
	return plain, true, nil
}

func (s *CardPlatformMonitorService) encryptCredential(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	return s.encryptor.Encrypt(strings.TrimSpace(plain))
}

func (s *CardPlatformMonitorService) decryptInPlace(m *CardPlatformMonitor) {
	if m == nil || m.Credential == "" {
		return
	}
	plain, err := s.encryptor.Decrypt(m.Credential)
	if err != nil {
		m.Credential = ""
		m.CredentialDecryptFailed = true
		return
	}
	m.Credential = plain
}

func (s *CardPlatformMonitorService) markChecked(ctx context.Context, id int64, lastErr string) {
	if err := s.repo.MarkChecked(ctx, id, time.Now(), lastErr); err != nil {
		slog.Warn("card_monitor: mark checked failed", "monitor_id", id, "error", err)
	}
}

func validateCardPlatform(v string) error {
	if normalizeCardPlatform(v) != CardPlatformLDXP {
		return ErrCardMonitorInvalidPlatform
	}
	return nil
}

func validateCardAuthMode(v string) error {
	switch normalizeCardAuthMode(v) {
	case CardAuthModePublic, CardAuthModeToken, CardAuthModeCookie:
		return nil
	default:
		return ErrCardMonitorInvalidAuthMode
	}
}

func validateCardInterval(v int) error {
	if v == 0 {
		return nil
	}
	if v < cardMinIntervalSeconds || v > cardMaxIntervalSeconds {
		return ErrCardMonitorInvalidInterval
	}
	return nil
}

func validateCardFetchPages(v int) error {
	if v == 0 {
		return nil
	}
	if v < 1 || v > cardMaxFetchPages {
		return ErrCardMonitorInvalidFetchPages
	}
	return nil
}

func normalizeCardPlatform(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return CardPlatformLDXP
	}
	return v
}

func normalizeCardAuthMode(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return CardAuthModeToken
	}
	return v
}

func needsCardCredential(mode string) bool {
	return mode == CardAuthModeToken || mode == CardAuthModeCookie
}
