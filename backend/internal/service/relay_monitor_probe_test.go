package service

import (
	"testing"
	"time"
)

func TestParseSub2APIGroups(t *testing.T) {
	body := []byte(`{"code":0,"message":"ok","data":[
		{"id":1,"name":"codex-team","rate_multiplier":0.01,"platform":"openai"},
		{"id":2,"name":"GPT PLUS","rate_multiplier":0.02},
		{"id":3,"name":"","rate_multiplier":9}
	]}`)
	out, err := parseSub2APIGroups(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 空名分组被跳过，剩 2 个；按名排序 "GPT PLUS" < "codex-team"（大写在前）。
	if len(out) != 2 {
		t.Fatalf("want 2 groups, got %d: %+v", len(out), out)
	}
	if out[0].GroupName != "GPT PLUS" || out[0].Rate != 0.02 {
		t.Errorf("unexpected first group: %+v", out[0])
	}
	if out[1].GroupName != "codex-team" || out[1].Rate != 0.01 {
		t.Errorf("unexpected second group: %+v", out[1])
	}
}

func TestParseSub2APIGroupsInvalid(t *testing.T) {
	if _, err := parseSub2APIGroups([]byte(`not json`)); err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestParseNewAPIGroupsTopLevel(t *testing.T) {
	body := []byte(`{"success":true,"data":[],"group_ratio":{"default":1,"vip":0.8},"usable_group":{"default":"默认","vip":"VIP"}}`)
	out, err := parseNewAPIGroups(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 groups, got %d", len(out))
	}
	got := map[string]float64{}
	for _, g := range out {
		got[g.GroupName] = g.Rate
	}
	if got["default"] != 1 || got["vip"] != 0.8 {
		t.Errorf("unexpected ratios: %+v", got)
	}
}

func TestParseNewAPIGroupsNested(t *testing.T) {
	// 兜底：group_ratio 嵌在 data 内。
	body := []byte(`{"success":true,"data":{"group_ratio":{"svip":0.5}}}`)
	out, err := parseNewAPIGroups(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].GroupName != "svip" || out[0].Rate != 0.5 {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestParseNewAPIGroupsMissing(t *testing.T) {
	if _, err := parseNewAPIGroups([]byte(`{"success":true,"data":[]}`)); err == nil {
		t.Fatal("expected error when group_ratio absent")
	}
}

func TestParseSub2APILoginToken(t *testing.T) {
	tok, err := parseSub2APILoginToken([]byte(`{"code":0,"message":"ok","data":{"access_token":"jwt-abc","refresh_token":"r"}}`))
	if err != nil || tok != "jwt-abc" {
		t.Fatalf("want jwt-abc, got %q err=%v", tok, err)
	}
	// 2FA → error
	if _, err := parseSub2APILoginToken([]byte(`{"data":{"requires_2fa":true}}`)); err == nil {
		t.Error("expected error when 2FA required")
	}
	// 无 token → error
	if _, err := parseSub2APILoginToken([]byte(`{"data":{"access_token":""}}`)); err == nil {
		t.Error("expected error when token empty")
	}
	// 非法 json → error
	if _, err := parseSub2APILoginToken([]byte(`nope`)); err == nil {
		t.Error("expected error on invalid json")
	}
}

func TestBuildRateChange(t *testing.T) {
	m := &RelayMonitor{ID: 7, Name: "mdkj", System: RelaySystemSub2API, Vendor: "OpenAI"}
	now := time.Now()

	// 首次见到（old 无）→ nil
	if c := buildRateChange(m, RelayGroupRate{GroupName: "g", Rate: 0.01}, map[string]float64{}, now); c != nil {
		t.Errorf("first sighting should not produce a change, got %+v", c)
	}
	// 不变 → nil
	if c := buildRateChange(m, RelayGroupRate{GroupName: "g", Rate: 0.01}, map[string]float64{"g": 0.01}, now); c != nil {
		t.Errorf("equal rate should not produce a change")
	}
	// 涨
	up := buildRateChange(m, RelayGroupRate{GroupName: "g", Rate: 0.02}, map[string]float64{"g": 0.01}, now)
	if up == nil || up.Direction != RelayDirectionUp || up.OldRate != 0.01 || up.NewRate != 0.02 {
		t.Errorf("unexpected up change: %+v", up)
	}
	if up.Site != "mdkj" || up.Vendor != "OpenAI" {
		t.Errorf("change should carry monitor metadata: %+v", up)
	}
	// 跌
	down := buildRateChange(m, RelayGroupRate{GroupName: "g", Rate: 0.005}, map[string]float64{"g": 0.01}, now)
	if down == nil || down.Direction != RelayDirectionDown {
		t.Errorf("unexpected down change: %+v", down)
	}
}

func TestFormatRate(t *testing.T) {
	cases := map[float64]string{
		0.005: "0.005x",
		0.01:  "0.01x",
		1:     "1x",
		0:     "0x",
	}
	for in, want := range cases {
		if got := formatRate(in); got != want {
			t.Errorf("formatRate(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestRatesEqual(t *testing.T) {
	if !ratesEqual(0.1, 0.1+1e-12) {
		t.Error("tiny float jitter should be treated as equal")
	}
	if ratesEqual(0.01, 0.02) {
		t.Error("distinct rates should not be equal")
	}
}
