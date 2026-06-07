package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

const cardResponseMaxBytes = 4 * 1024 * 1024

var ldxpPresetGoodsListURLs = []string{
	"https://pay.ldxp.cn/merchantApi/MyParent/searchGoodsList",
	"https://api.ldxp.cn/merchantApi/MyParent/searchGoodsList",
}

func probeCardProducts(ctx context.Context, m *CardPlatformMonitor) ([]*CardProductSnapshot, [][]byte, error) {
	switch m.PlatformType {
	case CardPlatformLDXP:
		return probeLDXPProducts(ctx, m)
	default:
		return nil, nil, ErrCardMonitorInvalidPlatform
	}
}

func probeLDXPProducts(ctx context.Context, m *CardPlatformMonitor) ([]*CardProductSnapshot, [][]byte, error) {
	if m.AuthMode == CardAuthModePublic {
		return nil, nil, fmt.Errorf("链动小铺公开页解析暂未启用，请先使用 Token 模式")
	}
	if strings.TrimSpace(m.Credential) == "" {
		return nil, nil, ErrCardMonitorMissingCredential
	}
	apiURLs, err := buildLDXPGoodsListURLs(m.BaseURL)
	if err != nil {
		return nil, nil, err
	}
	pages := m.FetchPages
	if pages <= 0 {
		pages = cardDefaultFetchPages
	}
	products := make([]*CardProductSnapshot, 0)
	raws := make([][]byte, 0)
	var lastErr error
	var serviceErr error
	for _, apiURL := range apiURLs {
		products, raws = products[:0], raws[:0]
		lastErr = nil
		for page := 1; page <= pages; page++ {
			body := map[string]any{
				"current":    page,
				"pageSize":   50,
				"name":       "",
				"goods_type": "",
				"keywords":   "",
			}
			payload, err := postLDXP(ctx, apiURL, strings.TrimSpace(m.Credential), body)
			if err != nil {
				lastErr = err
				if !isNetworkLookupError(err) {
					serviceErr = err
				}
				break
			}
			list := extractLDXPList(payload)
			for _, item := range list {
				p := normalizeLDXPProduct(m, item)
				if p.ExternalProductID == "" {
					continue
				}
				products = append(products, p)
				raws = append(raws, item)
			}
			if len(list) < 50 {
				break
			}
		}
		if lastErr == nil {
			return products, raws, nil
		}
	}
	if lastErr != nil {
		if serviceErr != nil {
			return nil, nil, serviceErr
		}
		return nil, nil, lastErr
	}
	return products, raws, nil
}

func postLDXP(ctx context.Context, apiURL, token string, body any) (map[string]any, error) {
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Merchant-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("链动小铺请求失败：%w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("链动小铺请求失败：HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, cardResponseMaxBytes))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		if strings.Contains(strings.TrimSpace(string(data[:minInt(len(data), 200)])), "<") {
			return nil, fmt.Errorf("链动小铺响应不是 JSON：平台预设接口返回了网页，请检查 Token 是否来自链动商户后台")
		}
		return nil, fmt.Errorf("链动小铺响应不是 JSON：%w", err)
	}
	if code := intFromAny(payload["code"], 0); code != 1 {
		msg := textFromAny(payload["msg"])
		if msg == "" {
			msg = textFromAny(payload["message"])
		}
		if msg == "" {
			msg = fmt.Sprintf("接口返回异常：%v", payload["code"])
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return payload, nil
}

func buildLDXPGoodsListURLs(rawBase string) ([]string, error) {
	rawBase = normalizeEndpoint(rawBase)
	if rawBase == "" || rawBase == CardPlatformLDXP {
		return ldxpPresetGoodsListURLs, nil
	}
	u, err := url.Parse(rawBase)
	if err != nil {
		return nil, err
	}
	if strings.Contains(u.Path, "/merchantApi/MyParent/searchGoodsList") {
		return []string{u.String()}, nil
	}
	u.Path = "/merchantApi/MyParent/searchGoodsList"
	u.RawQuery = ""
	u.Fragment = ""
	return []string{u.String()}, nil
}

func isNetworkLookupError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such host") || strings.Contains(msg, "lookup ")
}

func extractLDXPList(payload map[string]any) [][]byte {
	data, ok := payload["data"].(map[string]any)
	if !ok {
		return nil
	}
	rawList, ok := data["list"].([]any)
	if !ok {
		return nil
	}
	out := make([][]byte, 0, len(rawList))
	for _, item := range rawList {
		out = append(out, mustJSONBytes(item))
	}
	return out
}

func normalizeLDXPProduct(m *CardPlatformMonitor, raw []byte) *CardProductSnapshot {
	var item map[string]any
	_ = json.Unmarshal(raw, &item)
	id := firstText(item, "id", "goods_key", "goods_id")
	price := firstFloatPtr(item, "price")
	cost := firstFloatPtr(item, "cost_price", "agent_price_limit", "agent_price1", "agent_price2", "agent_price3")
	stock := firstIntPtr(item, "stock_count", "stock")
	sales := firstIntPtr(item, "sale_num", "sales")
	status := "unknown"
	switch intFromAny(item["status"], -999) {
	case 1:
		status = "online"
	case 0:
		status = "offline"
	}
	if stock != nil && *stock <= 0 && status == "online" {
		status = "sold_out"
	}
	category := nestedText(item, "category", "name")
	merchant := nestedText(item, "user", "nickname")
	productURL := ""
	if link := nestedText(item, "child", "link"); link != "" {
		productURL = link
	}
	lowest := cost
	if lowest == nil {
		lowest = price
	}
	return &CardProductSnapshot{
		MonitorID:         m.ID,
		PlatformName:      m.Name,
		PlatformType:      m.PlatformType,
		ExternalProductID: id,
		Title:             firstText(item, "name", "title"),
		Merchant:          merchant,
		Category:          category,
		ImageURL:          firstText(item, "image", "cover", "thumb"),
		ProductURL:        productURL,
		Price:             price,
		CostPrice:         cost,
		Stock:             stock,
		Sales:             sales,
		Status:            status,
		LowestPrice:       lowest,
	}
}

func firstText(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(textFromAny(m[key])); v != "" {
			return v
		}
	}
	return ""
}

func nestedText(m map[string]any, parent, child string) string {
	if sub, ok := m[parent].(map[string]any); ok {
		return strings.TrimSpace(textFromAny(sub[child]))
	}
	return ""
}

func textFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if math.Trunc(t) == t {
			return fmt.Sprintf("%.0f", t)
		}
		return fmt.Sprintf("%g", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

func firstFloatPtr(m map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if v, ok := floatFromAny(m[key]); ok && v >= 0 {
			return &v
		}
	}
	return nil
}

func firstIntPtr(m map[string]any, keys ...string) *int64 {
	for _, key := range keys {
		if v, ok := int64FromAny(m[key]); ok {
			return &v
		}
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func floatFromAny(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		var out float64
		if _, err := fmt.Sscanf(strings.ReplaceAll(t, ",", ""), "%f", &out); err == nil {
			return out, true
		}
	}
	return 0, false
}

func int64FromAny(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int:
		return int64(t), true
	case int64:
		return t, true
	case string:
		var out int64
		if _, err := fmt.Sscanf(strings.ReplaceAll(t, ",", ""), "%d", &out); err == nil {
			return out, true
		}
	}
	return 0, false
}

func intFromAny(v any, fallback int) int {
	if out, ok := int64FromAny(v); ok {
		return int(out)
	}
	return fallback
}

func mustJSONBytes(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
