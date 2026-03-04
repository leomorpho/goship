package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FileStorage holds the schema definition for the FileStorage entity.
type FileStorage struct {
	ent.Schema
}

func (FileStorage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

// Fields of the FileStorage.
func (FileStorage) Fields() []ent.Field {
	return []ent.Field{
		field.String("bucket_name").NotEmpty(),
		field.String("object_key").NotEmpty(),
		field.String("original_file_name").Optional(),
		field.Int64("file_size").Optional(),
		field.String("content_type").Optional(),
		field.String("file_hash").Optional(),
	}
}

// Edges of the FileStorage.
func (FileStorage) Edges() []ent.Edge {
	return nil
}

// Indexes of the FileStorage.
func (FileStorage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("bucket_name", "object_key").Unique(),
	}
}
