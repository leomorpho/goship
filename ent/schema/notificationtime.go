package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// NotificationTime holds the schema definition for the NotificationTime entity.
type NotificationTime struct {
	ent.Schema
}

func (NotificationTime) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the NotificationTime.
func (NotificationTime) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Values(domain.NotificationTypes.Values()...).
			Comment("Type of notification (e.g., message, update)"),
		field.Int("send_minute").
			Comment("Minutes since UTC midnight (0-1439) when the notification can be sent").
			Min(0).
			Max(1439),
		field.Int("profile_id").
			Comment("A user should only have 1 entry").
			Unique(),
	}
}

// Edges of the NotificationTime.
func (NotificationTime) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("notification_times").
			Field("profile_id").
			Unique().
			Required(),
	}
}

// Indexes of the Profile.
func (NotificationTime) Indexes() []ent.Index {
	return []ent.Index{
		// Someone can only have a single notification time per type
		index.Fields("profile_id", "type").Unique(),
	}
}
