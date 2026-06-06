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

// RelayRateSnapshot holds the schema definition for the RelayRateSnapshot entity.
// 中转站当前倍率快照：每个监控的每个被跟踪分组保留一行最新倍率，
// 用于和下一次探测结果对比算涨跌。(monitor_id, group_name) 唯一。
type RelayRateSnapshot struct {
	ent.Schema
}

func (RelayRateSnapshot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "relay_rate_snapshots"},
	}
}

func (RelayRateSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("monitor_id"),
		field.String("group_name").
			NotEmpty().
			MaxLen(200),
		field.Float("rate").
			Comment("当前分组倍率"),
		field.Time("updated_at").
			Default(time.Now),
	}
}

func (RelayRateSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("monitor", RelayMonitor.Type).
			Ref("snapshots").
			Field("monitor_id").
			Unique().
			Required(),
	}
}

func (RelayRateSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("monitor_id", "group_name").
			Unique(),
	}
}
