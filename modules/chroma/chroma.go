package chroma

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Container represents the Chroma container type used in the module
type Container struct {
	*testcontainers.DockerContainer
}

// Run creates an instance of the Chroma container type
func Run(ctx context.Context, img string, opts ...testcontainers.RequestCustomizer) (*Container, error) {
	req := testcontainers.Request{
		Image:        img,
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8000/tcp"),
			wait.ForLog("Application startup complete"),
			wait.ForHTTP("/api/v1/heartbeat").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}),
		), // 5 seconds it's not enough for the container to start
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

// RESTEndpoint returns the REST endpoint of the Chroma container
func (c *Container) RESTEndpoint(ctx context.Context) (string, error) {
	containerPort, err := c.MappedPort(ctx, "8000/tcp")
	if err != nil {
		return "", fmt.Errorf("failed to get container port: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container host")
	}

	return fmt.Sprintf("http://%s:%s", host, containerPort.Port()), nil
}
