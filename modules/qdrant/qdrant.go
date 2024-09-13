package qdrant

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Container represents the Qdrant container type used in the module
type Container struct {
	*testcontainers.DockerContainer
}

// Run creates an instance of the Qdrant container type
func Run(ctx context.Context, img string, opts ...testcontainers.RequestCustomizer) (*Container, error) {
	req := testcontainers.Request{
		Image:        img,
		ExposedPorts: []string{"6333/tcp", "6334/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("6333/tcp").WithStartupTimeout(5*time.Second),
			wait.ForListeningPort("6334/tcp").WithStartupTimeout(5*time.Second),
		),
		Started: true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&req); err != nil {
			return nil, err
		}
	}

	ctr, err := testcontainers.Run(ctx, req)
	var c *Container
	if ctr != nil {
		c = &Container{DockerContainer: ctr}
	}

	if err != nil {
		return c, fmt.Errorf("generic container: %w", err)
	}

	return c, nil
}

// RESTEndpoint returns the REST endpoint of the Qdrant container
func (c *Container) RESTEndpoint(ctx context.Context) (string, error) {
	containerPort, err := c.MappedPort(ctx, "6333/tcp")
	if err != nil {
		return "", fmt.Errorf("failed to get container port: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container host")
	}

	return fmt.Sprintf("http://%s:%s", host, containerPort.Port()), nil
}

// GRPCEndpoint returns the gRPC endpoint of the Qdrant container
func (c *Container) GRPCEndpoint(ctx context.Context) (string, error) {
	containerPort, err := c.MappedPort(ctx, "6334/tcp")
	if err != nil {
		return "", fmt.Errorf("failed to get container port: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container host")
	}

	return fmt.Sprintf("%s:%s", host, containerPort.Port()), nil
}

// WebUI returns the web UI endpoint of the Qdrant container
func (c *Container) WebUI(ctx context.Context) (string, error) {
	s, err := c.RESTEndpoint(ctx)
	if err != nil {
		return "", err
	}

	return s + "/dashboard", nil
}
