package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// MonthlySubscription holds the schema definition for the MonthlySubscription entity.
type MonthlySubscription struct {
	ent.Schema
}

func (MonthlySubscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the MonthlySubscription.
func (MonthlySubscription) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("product").
			Default(domain.ProductTypeFree.Value).
			Values(domain.ProductTypes.Values()...),
		field.Bool("is_active").
			Default(false).
			Comment("Whether this subscription is active or not."),
		field.Bool("paid").
			Default(false).
			Comment("Whether this subscription was paid or not."),
		field.Bool("is_trial").
			Default(true).
			Comment("Whether this subscription is a trial or not."),

		field.Time("started_at").
			Optional().
			Nillable().
			Comment("When the subscription started being effective."),
		field.Time("expired_on").
			Optional().
			Nillable().
			Comment("If the subscription expires, when it does so."),
		field.Time("cancelled_at").
			Optional().
			Nillable().
			Comment("Cancelling is effective after current period ends."),
		field.Int("paying_profile_id"),
	}
}

// Edges of the MonthlySubscription.
func (MonthlySubscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("benefactors", Profile.Type).
			Comment("Who is on this subscription.").
			Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("payer", Profile.Type).
			Comment("Who is paying for this subscription").
			Unique().
			Field("paying_profile_id").
			Required().
			Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

// Indexes of the MonthlySubscription.
func (MonthlySubscription) Indexes() []ent.Index {
	return []ent.Index{
		// Someone can only have a single active subscription at a time.
		index.Fields("paying_profile_id", "is_active").Unique(),
	}
}
