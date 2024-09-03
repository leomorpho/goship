package schema

import (
	"context"
	"strings"

	ge "github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/hook"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),
		field.String("email").
			NotEmpty().
			Unique(),
		field.String("password").
			Sensitive().
			NotEmpty(),
		field.Bool("verified").
			Default(false),
		field.Time("last_online").
			Optional(),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", PasswordToken.Type).
			Ref("user"),

		edge.To("profile", Profile.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("last_seen_at", LastSeenOnline.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Hooks of the User.
func (User) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.UserFunc(func(ctx context.Context, m *ge.UserMutation) (ent.Value, error) {
					if v, exists := m.Email(); exists {
						m.SetEmail(strings.ToLower(v))
					}
					return next.Mutate(ctx, m)
				})
			},
			// Limit the hook only for these operations.
			ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne,
		),
	}
}
