package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/ent"
)

// GeoClient contains dependencies for geospatial calculations and database interactions.
type (
	GeoClient struct {
		config      *config.Config
		orm         *ent.Client
		redisClient *redis.Client // TODO: abstract away
	}

	// Point represents a geographical point with latitude and longitude.
	Point struct {
		Latitude  float64
		Longitude float64
	}
)

// NewGeoClient initializes a new GeoClient with the given configuration and ORM client.
func NewGeoClient(cfg *config.Config, orm *ent.Client) *GeoClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.Cache.Hostname, cfg.Cache.Port),
		DB:   cfg.Cache.Database,
	})
	return &GeoClient{
		config:      cfg,
		orm:         orm,
		redisClient: rdb,
	}
}

// AddPointsToRedis adds a list of geographical points to a Redis Geo set.
// The function takes a slice of points and a Redis set name.
func (g *GeoClient) AddPointsToRedis(points []Point, setName string) error {
	for _, point := range points {
		if _, err := g.redisClient.GeoAdd(context.Background(), setName, &redis.GeoLocation{
			Longitude: point.Longitude,
			Latitude:  point.Latitude,
		}).Result(); err != nil {
			return err
		}
	}
	return nil
}

// FindNeighboursByDist finds all points within a given distance from a central point.
// The function takes a central point, a Redis Geo set name, and a distance in km.
// It returns a slice of points within the given distance.
func (g *GeoClient) FindNeighboursByDist(point Point, setName string, distanceInKm float64) ([]Point, error) {
	res, err := g.redisClient.GeoRadius(context.Background(), setName, point.Longitude, point.Latitude, &redis.GeoRadiusQuery{
		Radius: distanceInKm,
		Unit:   "km",
	}).Result()

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, errors.New("no points found within the specified distance")
	}

	var neighbours []Point
	for _, location := range res {
		neighbours = append(neighbours, Point{Latitude: location.Latitude, Longitude: location.Longitude})
	}

	return neighbours, nil
}
