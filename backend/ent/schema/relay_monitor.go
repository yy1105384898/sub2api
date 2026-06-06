package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RelayMonitor holds the schema definition for the RelayMonitor entity.
// 中转站监控配置：定期抓取外部中转站（sub2api / newapi）的分组倍率，
// 只跟踪 watched_groups 指定的分组，记录涨/跌变化。
type RelayMonitor struct {
	ent.Schema
}

func (RelayMonitor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "relay_monitors"},
	}
}

func (RelayMonitor) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (RelayMonitor) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("中转站显示名称"),
		field.Enum("system").
			Values("sub2api", "newapi").
			Comment("目标站点使用的网关系统，决定探测解析器"),
		field.String("base_url").
			NotEmpty().
			MaxLen(500).
			Comment("中转站站点根地址，如 https://example.com"),
		field.String("vendor").
			Optional().
			Default("").
			MaxLen(50).
			Comment("厂商标签，如 OpenAI（仅展示用）"),
		field.String("credential_encrypted").
			Optional().
			Default("").
			Sensitive().
			Comment("AES-256-GCM 加密的访问凭证：sub2api 需要 Bearer token，newapi 可留空"),
		field.JSON("watched_groups", []string{}).
			Default([]string{}).
			Comment("需要监控的分组名列表；为空表示尚未选择，不探测任何分组"),
		field.Bool("enabled").
			Default(true),
		field.Int("interval_seconds").
			Range(60, 86400).
			Default(300).
			Comment("探测间隔秒数"),
		field.Time("last_checked_at").
			Optional().
			Nillable(),
		field.String("last_error").
			Optional().
			Default("").
			MaxLen(500).
			Comment("最近一次探测的错误信息，空表示成功"),
		field.Int64("created_by"),
	}
}

func (RelayMonitor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("snapshots", RelayRateSnapshot.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("changes", RelayRateChange.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelayMonitor) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled", "last_checked_at"),
		index.Fields("system"),
	}
}
