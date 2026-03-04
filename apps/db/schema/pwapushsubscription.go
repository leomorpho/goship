package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PwaPushSubscription holds the schema definition for the PwaPushSubscription entity.
type PwaPushSubscription struct {
	ent.Schema
}

func (PwaPushSubscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the PwaPushSubscription.
func (PwaPushSubscription) Fields() []ent.Field {
	return []ent.Field{
		field.String("endpoint").NotEmpty(),
		field.String("p256dh").NotEmpty(),
		field.String("auth").NotEmpty(),
		field.Int("profile_id"),
	}
}

// Edges of the PwaPushSubscription.
func (PwaPushSubscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("pwa_push_subscriptions").
			Field("profile_id").
			Required().
			Unique(),
	}
}

// Indexes of the Card.
func (PwaPushSubscription) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("profile_id", "endpoint").Unique(),
	}
}
