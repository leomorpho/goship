package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// LastSeenOnline holds the schema definition for the LastSeenOnline entity.
type LastSeenOnline struct {
	ent.Schema
}

// Fields of the LastSeenOnline.
func (LastSeenOnline) Fields() []ent.Field {
	return []ent.Field{
		field.Time("seen_at").
			Immutable().
			Default(time.Now),
	}
}

// Edges of the LastSeenOnline.
func (LastSeenOnline) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("last_seen_at").
			Unique().
			Required(),
	}
}
