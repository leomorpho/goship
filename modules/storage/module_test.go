package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/core"
)

type blobStorageStub struct{}

func (blobStorageStub) Put(context.Context, core.PutObjectInput) (core.StoredObject, error) {
	return core.StoredObject{Bucket: "main", Key: "avatars/test.png"}, nil
}

func (blobStorageStub) Delete(context.Context, string, string) error {
	return nil
}

func (blobStorageStub) PresignGet(context.Context, string, string, time.Duration) (string, error) {
	return "/uploads/main/avatars/test.png", nil
}

func TestNew(t *testing.T) {
	mod := New(blobStorageStub{})
	if mod == nil {
		t.Fatal("expected storage module")
	}
	if mod.ID() != "storage" {
		t.Fatalf("id = %q", mod.ID())
	}
	if mod.BlobStorage() == nil {
		t.Fatal("expected blob storage seam")
	}
}

func TestBlobStorageContract(t *testing.T) {
	mod := New(blobStorageStub{})
	out, err := mod.BlobStorage().Put(context.Background(), core.PutObjectInput{
		Bucket: "main",
		Key:    "avatars/test.png",
		Reader: strings.NewReader("hello"),
		Size:   5,
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if out.Key != "avatars/test.png" {
		t.Fatalf("key = %q", out.Key)
	}
}
