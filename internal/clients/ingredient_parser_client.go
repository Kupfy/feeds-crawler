package clients

import (
	"context"
	"fmt"

	pb "github.com/Kupfy/feeds-crawler/pkg/grpc/ingredientparser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type IngredientParserClient struct {
	conn   *grpc.ClientConn
	client pb.IngredientParserClient
}

func NewIngredientParserClient(addr string) (c *IngredientParserClient, err error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingredient parser client: %w", err)
	}

	return &IngredientParserClient{
		conn:   conn,
		client: pb.NewIngredientParserClient(conn),
	}, nil
}

func (c *IngredientParserClient) HealthCheck(ctx context.Context) error {
	healthClient := healthpb.NewHealthClient(c.conn)
	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{
		Service: "ingredient_parser",
	})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("ingredient parser not serving: %s", resp.Status)
	}
	return nil
}

func (c *IngredientParserClient) ParseIngredient(ctx context.Context, raw string) (*pb.ParseResponse, error) {
	resp, err := c.client.Parse(ctx, &pb.ParseRequest{
		Raw: raw,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse ingredient: %w", err)
	}

	return resp, nil
}

func (c *IngredientParserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
