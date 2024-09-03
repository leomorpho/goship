package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

// ImageSize holds the schema definition for the ImageSize entity.
type ImageSize struct {
	ent.Schema
}

func (ImageSize) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the ImageSize.
func (ImageSize) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("size").
			Values(domain.ImageSizes.Values()...).
			Comment("The size of this image instance"),
		field.Int("width").Positive(),
		field.Int("height").Positive(),
	}
}

// Edges of the ImageSize.
func (ImageSize) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("file", FileStorage.Type).
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		// We add the "Required" method to the builder
		// to make this edge required on entity creation.
		// i.e. Card cannot be created without its owner.
		edge.From("image", Image.Type).
			Ref("sizes").
			Unique(),
	}
}
