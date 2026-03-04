package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FCMSubscriptions holds the schema definition for the FCMSubscriptions entity.
type FCMSubscriptions struct {
	ent.Schema
}

func (FCMSubscriptions) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the FCMSubscriptions.
func (FCMSubscriptions) Fields() []ent.Field {
	return []ent.Field{
		field.String("token").NotEmpty(),
		field.Int("profile_id"),
	}
}

// Edges of the FCMSubscriptions.
func (FCMSubscriptions) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("fcm_push_subscriptions").
			Field("profile_id").
			Required().
			Unique(),
	}
}

// Indexes of the FCMSubscriptions.
func (FCMSubscriptions) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token", "profile_id").Unique(),
	}
}
