package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RelayRateChange holds the schema definition for the RelayRateChange entity.
// 中转站倍率变化历史：每检测到一个分组倍率发生变化记录一行（涨/跌公告）。
// 日志类表，无软删除；每站超出保留上限由维护逻辑批量物理删。
type RelayRateChange struct {
	ent.Schema
}

func (RelayRateChange) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "relay_rate_changes"},
	}
}

func (RelayRateChange) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("monitor_id"),
		field.String("site").
			MaxLen(100).
			Comment("监控名称快照，便于历史独立展示"),
		field.String("system").
			MaxLen(20),
		field.String("vendor").
			Optional().
			Default("").
			MaxLen(50),
		field.String("group_name").
			NotEmpty().
			MaxLen(200),
		field.Float("old_rate"),
		field.Float("new_rate"),
		field.Enum("direction").
			Values("up", "down").
			Comment("up=涨(倍率变大), down=跌(倍率变小)"),
		field.String("content").
			Optional().
			Default("").
			MaxLen(500).
			Comment("公告文本，如 分组倍率从 0.005x 变为 0.001x"),
		field.Time("detected_at").
			Default(time.Now),
	}
}

func (RelayRateChange) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("monitor", RelayMonitor.Type).
			Ref("changes").
			Field("monitor_id").
			Unique().
			Required(),
	}
}

func (RelayRateChange) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("monitor_id", "detected_at"),
		index.Fields("direction", "detected_at"),
		index.Fields("detected_at"),
	}
}
