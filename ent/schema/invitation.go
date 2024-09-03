package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Invitation holds the schema definition for the Invitation entity.
type Invitation struct {
	ent.Schema
}

func (Invitation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Invitation.
func (Invitation) Fields() []ent.Field {
	return []ent.Field{
		field.String("invitee_name").
			NotEmpty(),
		field.String("confirmation_code").
			NotEmpty().
			Unique(),
	}
}

// Edges of the Invitation.
func (Invitation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("inviter", Profile.Type).
			Ref("invitations").
			Unique().
			Required().
			Comment("The profile who created the invitation."),
	}
}
