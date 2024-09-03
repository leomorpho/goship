package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"

	"github.com/weaviate/weaviate/entities/models"
)

// https://medium.com/@newbing/how-to-use-weaviate-to-store-openai-embedding-vectors-in-a-golang-program-8b91787197f1
// https://github.com/r0mdau/refind

type (
	MLClient struct {
		config         *config.Config
		orm            *ent.Client
		weaviateClient *weaviate.Client
	}
)

func NewMLClient(cfg *config.Config, orm *ent.Client) *MLClient {
	weaviateCfg := weaviate.Config{
		Host:   cfg.ML.WeaviateHost,
		Scheme: cfg.ML.WeaviateScheme,
	}
	client, err := weaviate.NewClient(weaviateCfg)
	if err != nil {
		panic(err)
	}

	classObj := &models.Class{
		Class:      "Question",
		Vectorizer: "text2vec-transformers", // If set to "none" you must always provide vectors yourself. Could be any other "text2vec-*" also.
		ModuleConfig: map[string]interface{}{
			"text2vec-openai":   map[string]interface{}{},
			"generative-openai": map[string]interface{}{},
		},
		Properties: []*models.Property{
			{
				Name:        "question",
				Description: "Contains the question text",
				DataType:    []string{"text"},
			},
			{
				Name:        "external_id",
				Description: "ID shared with systems external to Weaviate",
				DataType:    []string{"text"},
			},
		},
	}

	err = CreateWeaviateClassIfNotExist(client, classObj)
	if err != nil {
		log.Fatalf("Failed to create class: %v", err)
	}

	return &MLClient{
		config:         cfg,
		orm:            orm,
		weaviateClient: client,
	}
}

func fetchExistingWeaviateClasses(client *weaviate.Client) (map[string]bool, error) {
	schema, err := client.Schema().Getter().Do(context.Background())
	if err != nil {
		return nil, err
	}

	existingClassMap := make(map[string]bool)
	for _, class := range schema.Classes {
		existingClassMap[class.Class] = true
	}
	return existingClassMap, nil
}

// CreateWeaviateClassIfNotExist creates a new class if it does not already exist.
func CreateWeaviateClassIfNotExist(client *weaviate.Client, newClass *models.Class) error {
	existingClasses, err := fetchExistingWeaviateClasses(client)
	if err != nil {
		return err
	}

	if _, exists := existingClasses[newClass.Class]; !exists {
		err := client.Schema().ClassCreator().
			WithClass(newClass).
			Do(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}

// Checks if the question already exists in Weaviate
func (ml *MLClient) QuestionExists(ctx context.Context, sentence string) (bool, error) {
	gqlField := []graphql.Field{
		{
			Name: "_additional",
			Fields: []graphql.Field{
				{Name: "id"},
			},
		},
	}

	whereFilter := filters.Where().
		WithPath([]string{"question"}).
		WithOperator(filters.Equal).
		WithValueString(sentence)

	res, err := ml.weaviateClient.GraphQL().Get().
		WithClassName("Question").
		WithFields(gqlField...).
		WithWhere(whereFilter).
		WithLimit(1).
		Do(ctx)

	if err != nil {
		return false, err
	}

	if getRes, ok := res.Data["Get"]; ok {
		getMap, ok := getRes.(map[string]any)
		if ok {
			list, ok := getMap["Question"]
			if ok {
				retList, ok := list.([]any)
				if ok {
					return len(retList) > 0, nil
				} else {
					return false, errors.New("data not array list")
				}
			} else {
				return false, errors.New("data not found")
			}
		} else {
			return false, errors.New("no get data found")
		}
	}

	return false, errors.New("unexpected response format")
}

// Adds a new question to Weaviate if it doesn't already exist
func (ml *MLClient) AddQuestionToVectorStore(ctx context.Context, sentence string) error {
	exists, err := ml.QuestionExists(ctx, sentence)
	if err != nil {
		return err
	}

	if exists {
		log.Println("Question already exists, skipping.")
		return nil
	}

	w, err := ml.weaviateClient.Data().Creator().
		WithClassName("Question").
		WithProperties(map[string]interface{}{
			"question": sentence,
		}).
		Do(ctx)

	if err != nil {
		log.Fatal("Failed to add question to vector store:", err)
		return err
	}

	b, _ := json.MarshalIndent(w.Object, "", "  ")
	fmt.Println(string(b))

	return nil
}
