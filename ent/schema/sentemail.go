package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// SentEmail holds the schema definition for the SentEmail entity.
type SentEmail struct {
	ent.Schema
}

func (SentEmail) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the SentEmail.
func (SentEmail) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Values(domain.NotificationPermissions.Values()...),
	}
}

// Edges of the SentEmail.
func (SentEmail) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("sent_emails").
			Unique().
			Required(),
	}
}
