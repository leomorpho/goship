package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Emojis holds the schema definition for the Emojis entity.
type Emojis struct {
	ent.Schema
}

// Fields of the Emojis.
func (Emojis) Fields() []ent.Field {
	return []ent.Field{
		field.String("unified_code").NotEmpty(),
		field.String("shortcode").NotEmpty(),
	}
}

// Edges of the Emojis.
func (Emojis) Edges() []ent.Edge {
	return []ent.Edge{
		// edge.To("answer_reactions", AnswerReactions.Type),
	}
}
