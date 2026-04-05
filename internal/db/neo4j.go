package db

import (
	"context"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jClient struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jClient(uri, user, password string) (*Neo4jClient, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, err
	}
	return &Neo4jClient{driver: driver}, nil
}

func (c *Neo4jClient) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.driver.Close(ctx)
}

func (c *Neo4jClient) Query(query string, params map[string]any) ([]map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	for result.Next(ctx) {
		record := result.Record()
		row := make(map[string]any)
		for _, key := range record.Keys {
			val, _ := record.Get(key)
			row[key] = convertValue(val)
		}
		records = append(records, row)
	}
	return records, nil
}

func convertValue(v any) any {
	switch val := v.(type) {
	case neo4j.Node:
		return val.Props
	case neo4j.Relationship:
		return val.Props
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = convertValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			out[k] = convertValue(item)
		}
		return out
	default:
		return v
	}
}
