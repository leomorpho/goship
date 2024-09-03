package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/mikestefanello/pagoda/ent/hook"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

func (Notification) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Values(domain.NotificationTypes.Values()...).
			Comment("Type of notification (e.g., message, update)"),
		field.String("title").
			Default(""). // TODO: had to set a default because this field was added after model creation and had data in it.
			Comment("Title the notification"),
		field.String("text").
			Comment("Main content of the notification"),
		field.String("link").
			Optional().
			Nillable().
			Comment("Optional URL for the resource related to the notification"),
		field.Bool("read").
			Comment("Indicates if the notification has been read").
			Default(false),
		field.Time("read_at").
			Comment("Time when the notification was read").
			Optional().
			Nillable(),
		field.Int("profile_id_who_caused_notification").
			Optional().
			Nillable(),
		field.Int("resource_id_tied_to_notif").
			Optional().
			Nillable(),
		field.Bool("read_in_notifications_center").
			Optional().
			Nillable(),
	}
}

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", Profile.Type).
			Ref("notifications").
			Unique(),
	}
}

func (Notification) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.EnsureUTCHook(
			"read_at",
		),
	}
}
