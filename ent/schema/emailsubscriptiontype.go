package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// TODO: rename to EmailSubscriptionList
// EmailSubscriptionType holds the schema definition for the EmailSubscriptionType entity.
type EmailSubscriptionType struct {
	ent.Schema
}

func (EmailSubscriptionType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the EmailSubscriptionType.
func (EmailSubscriptionType) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("name").
			Values(domain.EmailSubscriptionLists.Values()...),
		field.Bool("active").
			Default(true),
	}
}

// Edges of the EmailSubscriptionType.
func (EmailSubscriptionType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscriber", EmailSubscription.Type).
			Ref("subscriptions").
			Comment("Subscriber subscribed to this subscription type."),
	}
}
