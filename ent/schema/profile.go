package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Profile struct {
	ent.Schema
}

func (Profile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (Profile) Fields() []ent.Field {
	return []ent.Field{
		field.String("bio").
			Optional().
			Comment("A short bio of the user."),
		field.Time("birthdate").
			Optional().
			Comment("The birthdate of the user."),
		field.Int("age").
			Optional().
			Comment("The age of the user."),
		field.Bool("fully_onboarded").
			Default(false).
			Comment("An onboarded user has entered all required data to allow for basic app functionalities."),
		field.String("phone_number_e164").
			Optional().
			Comment("Phone number in E164 format"),
		field.String("country_code").
			Optional().
			Comment("Phone number country code"),
		field.Bool("phone_verified").
			Optional().
			Comment("Whether the associated phone number was verified to be reachable"),
		field.String("stripe_id").
			Unique().
			Optional().
			Comment("ID used by stripe payment system"),
	}
}

func (Profile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("friends", Profile.Type).
			Comment("Who the profile is friends/connected to.").
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("photos", Image.Type).
			Comment("Photos associated to that profile, not including the profile picture.").
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("profile_image", Image.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("notifications", Notification.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invitations", Invitation.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.To("fcm_push_subscriptions", FCMSubscriptions.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Comment("Track FCM push notification subscriptions, used for iOS"),
		edge.To("pwa_push_subscriptions", PwaPushSubscription.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Comment("Track PWA push notification subscriptions, used for all device types but iOS"),
		edge.To("notification_permissions", NotificationPermission.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("notification_times", NotificationTime.Type).
			Comment("Times at which a notification type should be sent to a profile").
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("phone_verification_code", PhoneVerificationCode.Type).
			Comment("Phone verification code associated with this user").
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("sent_emails", SentEmail.Type).
			Comment("Emails this profile was sent").
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.From("user", User.Type).
			Ref("profile").
			// Field("user_id").
			Unique().
			Required(),
		edge.From("subscription", MonthlySubscription.Type).
			Ref("benefactors").Annotations(),
	}
}

// // Indexes of the Profile.
// func (Profile) Indexes() []ent.Index {
// 	return []ent.Index{
// 		// Someone can only have a single active subscription at a time.
// 		index.Fields("id", "user_id").Unique(),
// 	}
// }
