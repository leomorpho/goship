package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PhoneVerificationCode holds the schema definition for the PhoneVerificationCode entity.
type PhoneVerificationCode struct {
	ent.Schema
}

func (PhoneVerificationCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the PhoneVerificationCode.
func (PhoneVerificationCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("The verification code"),
		field.Int("profile_id"),
	}
}

// Edges of the PhoneVerificationCode.
func (PhoneVerificationCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("phone_verification_code").
			Field("profile_id").
			Unique().
			Required(),
	}
}

// Indexes of the Profile.
func (PhoneVerificationCode) Indexes() []ent.Index {
	return []ent.Index{
		// Someone can only have a single active subscription at a time.
		index.Fields("code", "profile_id").Unique(),
	}
}
