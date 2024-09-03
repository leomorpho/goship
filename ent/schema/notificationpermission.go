package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// NotificationPermission holds the schema definition for the NotificationPermission entity.
type NotificationPermission struct {
	ent.Schema
}

func (NotificationPermission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the NotificationPermission.
func (NotificationPermission) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("permission").
			Values(domain.NotificationPermissions.Values()...),
		field.Enum("platform").
			Values(domain.NotificationPlatforms.Values()...),
		field.Int("profile_id"),
		field.String("token").
			Comment("For permissions cancellable through out-of-app-platform, this is like an auth token"),
	}
}

// Edges of the NotificationPermission.
func (NotificationPermission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("notification_permissions").
			Field("profile_id").
			Required().
			Unique(),
	}
}

// Indexes of the Card.
func (NotificationPermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("profile_id", "permission", "platform").Unique(),
	}
}
