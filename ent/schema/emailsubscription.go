package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// EmailSubscription holds the schema definition for the EmailSubscription entity.
type EmailSubscription struct {
	ent.Schema
}

func (EmailSubscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the EmailSubscription.
func (EmailSubscription) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			NotEmpty().
			Unique(),
		field.Bool("verified").
			Default(false),
		field.String("confirmation_code").
			NotEmpty().
			Unique(),
		field.Float("latitude").
			Optional().
			Comment("The latitude of the subscriber's location."),
		field.Float("longitude").
			Optional().
			Comment("The longitude of the subscriber's location."),
	}
}

// Edges of the EmailSubscription.
func (EmailSubscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subscriptions", EmailSubscriptionType.Type).
			Comment("Subscriptions that this email is subscribed to"),
	}
}
