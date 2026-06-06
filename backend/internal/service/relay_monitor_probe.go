package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// 中转站倍率探测器。
//
// 按 system 选择解析器去抓目标站对外公布的分组倍率：
//   - sub2api: GET {base}/api/v1/groups/available（需 Bearer token），解析 data[].rate_multiplier
//   - newapi:  GET {base}/api/pricing（公开），解析 group_ratio map
//
// 复用 channel_monitor 的 SSRF-safe HTTP client（safeDialContext 在 socket 层拦私网）。

// relayHTTPClient 共享 SSRF-safe client，避免每次探测重建 transport。
var relayHTTPClient = newSSRFSafeHTTPClient(relayProbeTimeout)

// probeRelayRates 抓取目标站全部分组的当前倍率。
// 返回的分组未经 watched_groups 过滤，由调用方按需筛选（探测时筛选 / 拉取分组列表时全返）。
func probeRelayRates(ctx context.Context, system, baseURL, credential string) ([]RelayGroupRate, error) {
	switch system {
	case RelaySystemSub2API:
		return probeSub2API(ctx, baseURL, credential)
	case RelaySystemNewAPI:
		return probeNewAPI(ctx, baseURL)
	default:
		return nil, ErrRelayMonitorInvalidSystem
	}
}

// ---------- sub2api ----------

// sub2apiEnvelope sub2api 标准响应信封 {code, message, data}。
type sub2apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// sub2apiGroup /groups/available 返回的分组条目（只取需要的字段）。
type sub2apiGroup struct {
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

// probeSub2API 抓取 sub2api 站点的分组倍率。需要登录态 Bearer token。
func probeSub2API(ctx context.Context, baseURL, credential string) ([]RelayGroupRate, error) {
	if strings.TrimSpace(credential) == "" {
		return nil, ErrRelayMonitorMissingCredential
	}
	u := joinRelayURL(baseURL, "/api/v1/groups/available")
	body, err := relayGet(ctx, u, map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(credential),
	})
	if err != nil {
		return nil, err
	}
	return parseSub2APIGroups(body)
}

// parseSub2APIGroups 解析 sub2api /groups/available 响应为分组倍率列表（纯函数，便于单测）。
func parseSub2APIGroups(body []byte) ([]RelayGroupRate, error) {
	var env sub2apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("%w: invalid sub2api response", ErrRelayMonitorProbeFailed)
	}
	var groups []sub2apiGroup
	if err := json.Unmarshal(env.Data, &groups); err != nil {
		return nil, fmt.Errorf("%w: invalid sub2api group payload", ErrRelayMonitorProbeFailed)
	}
	out := make([]RelayGroupRate, 0, len(groups))
	for _, g := range groups {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			continue
		}
		out = append(out, RelayGroupRate{GroupName: name, Rate: g.RateMultiplier})
	}
	return sortRelayRates(out), nil
}

// ---------- newapi ----------

// newapiPricing /api/pricing 公开响应（只取 group_ratio）。
// 不同 newapi 版本字段略有差异：group_ratio 为分组倍率 map，usable_group 为分组显示名 map。
type newapiPricing struct {
	GroupRatio  map[string]float64 `json:"group_ratio"`
	UsableGroup map[string]any     `json:"usable_group"`
}

// probeNewAPI 抓取 newapi 站点模型广场的分组倍率（公开接口，无需凭证）。
func probeNewAPI(ctx context.Context, baseURL string) ([]RelayGroupRate, error) {
	u := joinRelayURL(baseURL, "/api/pricing")
	body, err := relayGet(ctx, u, nil)
	if err != nil {
		return nil, err
	}
	return parseNewAPIGroups(body)
}

// parseNewAPIGroups 解析 newapi /api/pricing 响应为分组倍率列表（纯函数，便于单测）。
func parseNewAPIGroups(body []byte) ([]RelayGroupRate, error) {
	// newapi 用 {success, data, group_ratio, usable_group} 信封；group_ratio 在顶层。
	var top struct {
		Data json.RawMessage `json:"data"`
		newapiPricing
	}
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, fmt.Errorf("%w: invalid newapi response", ErrRelayMonitorProbeFailed)
	}
	ratios := top.GroupRatio
	if len(ratios) == 0 {
		// 部分版本把 group_ratio 嵌在 data 内，兜底再解一层。
		var inner newapiPricing
		if len(top.Data) > 0 {
			_ = json.Unmarshal(top.Data, &inner)
		}
		ratios = inner.GroupRatio
	}
	if len(ratios) == 0 {
		return nil, fmt.Errorf("%w: newapi group_ratio not found", ErrRelayMonitorProbeFailed)
	}
	out := make([]RelayGroupRate, 0, len(ratios))
	for name, rate := range ratios {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, RelayGroupRate{GroupName: name, Rate: rate})
	}
	return sortRelayRates(out), nil
}

// ---------- HTTP / helpers ----------

// relayGet 发起一次 GET 并读取（受限）响应体。非 2xx 视为探测失败。
func relayGet(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request", ErrRelayMonitorProbeFailed)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sub2api-relay-monitor")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := relayHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRelayMonitorProbeFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, relayResponseMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: read body", ErrRelayMonitorProbeFailed)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: upstream HTTP %d", ErrRelayMonitorProbeFailed, resp.StatusCode)
	}
	return body, nil
}

// joinRelayURL 把 base origin 与 path 拼接（base 已由 validateEndpoint 保证为无 path 的 origin）。
func joinRelayURL(base, path string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") + path
}

// sortRelayRates 按分组名稳定排序，保证拉取分组列表与回归测试输出确定。
func sortRelayRates(in []RelayGroupRate) []RelayGroupRate {
	sort.Slice(in, func(i, j int) bool { return in[i].GroupName < in[j].GroupName })
	return in
}
