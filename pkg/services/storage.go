package services

import (
	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/ent"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
)

type StorageClient struct {
	client *storagerepo.StorageClient
}

func NewStorageClient(cfg *config.Config, orm *ent.Client) *StorageClient {
	storageRepo := storagerepo.NewStorageClient(cfg, orm)

	return &StorageClient{
		client: storageRepo,
	}
}
