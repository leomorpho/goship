package generators

import (
	"strings"
	"testing"
)

func TestRenderModelQueryTemplateIncludesCRUDQueriesAndSchemaHints(t *testing.T) {
	got := RenderModelQueryTemplate("Post", []ModelField{
		{Name: "title", Type: "string"},
		{Name: "published", Type: "bool"},
	})

	for _, want := range []string{
		"-- ship:generated:model:post",
		"-- Suggested migration columns:",
		"-- - title TEXT",
		"-- - published BOOLEAN",
		"-- name: CreatePost :one",
		"INSERT INTO posts (",
		"title,",
		"published",
		"-- name: ListPosts :many",
		"SELECT * FROM posts ORDER BY id DESC;",
		"-- name: UpdatePost :one",
		"title = ?",
		"published = ?",
		"-- name: DeletePost :exec",
		"DELETE FROM posts WHERE id = ?;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderModelQueryTemplate() missing %q\n%s", want, got)
		}
	}
}

func TestRenderModelQueryTemplateWithoutFieldsStillProducesCRUDShape(t *testing.T) {
	got := RenderModelQueryTemplate("Post", nil)
	for _, want := range []string{
		"-- ship:generated:model:post",
		"INSERT INTO posts DEFAULT VALUES RETURNING *;",
		"-- name: ListPosts :many",
		"-- name: UpdatePost :one",
		"UPDATE posts SET id = id WHERE id = ? RETURNING *;",
		"-- name: DeletePost :exec",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderModelQueryTemplate() missing %q\n%s", want, got)
		}
	}
}
