package azurite

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// BlobPort is the default port used by Azurite
	BlobPort = "10000/tcp"
	// QueuePort is the default port used by Azurite
	QueuePort = "10001/tcp"
	// TablePort is the default port used by Azurite
	TablePort = "10002/tcp"

	// defaultCredentials {
	// AccountName is the default testing account name used by Azurite
	AccountName string = "devstoreaccount1"

	// AccountKey is the default testing account key used by Azurite
	AccountKey string = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
	// }
)

// Container represents the Azurite container type used in the module
type Container struct {
	*testcontainers.DockerContainer
	Settings options
}

func (c *Container) ServiceURL(ctx context.Context, srv Service) (string, error) {
	hostname, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	var port nat.Port
	switch srv {
	case BlobService:
		port = BlobPort
	case QueueService:
		port = QueuePort
	case TableService:
		port = TablePort
	default:
		return "", fmt.Errorf("unknown service: %s", srv)
	}

	mappedPort, err := c.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%d", hostname, mappedPort.Int()), nil
}

func (c *Container) MustServiceURL(ctx context.Context, srv Service) string {
	url, err := c.ServiceURL(ctx, srv)
	if err != nil {
		panic(err)
	}

	return url
}

// Run creates an instance of the Azurite container type
func Run(ctx context.Context, img string, opts ...testcontainers.RequestCustomizer) (*Container, error) {
	req := testcontainers.Request{
		Image:        img,
		ExposedPorts: []string{BlobPort, QueuePort, TablePort},
		Env:          map[string]string{},
		Entrypoint:   []string{"azurite"},
		Cmd:          []string{},
		Started:      true,
	}

	// 1. Gather all config options (defaults and then apply provided options)
	settings := defaultOptions()
	for _, opt := range opts {
		if err := opt.Customize(&req); err != nil {
			return nil, err
		}
	}

	// 2. evaluate the enabled services to apply the right wait strategy and Cmd options
	enabledServices := settings.EnabledServices
	if len(enabledServices) > 0 {
		waitingFor := make([]wait.Strategy, 0)
		for _, srv := range enabledServices {
			switch srv {
			case BlobService:
				req.Cmd = append(req.Cmd, "--blobHost", "0.0.0.0")
				waitingFor = append(waitingFor, wait.ForLog("Blob service is successfully listening"))
			case QueueService:
				req.Cmd = append(req.Cmd, "--queueHost", "0.0.0.0")
				waitingFor = append(waitingFor, wait.ForLog("Queue service is successfully listening"))
			case TableService:
				req.Cmd = append(req.Cmd, "--tableHost", "0.0.0.0")
				waitingFor = append(waitingFor, wait.ForLog("Table service is successfully listening"))
			}
		}

		if len(waitingFor) > 0 {
			req.WaitingFor = wait.ForAll(waitingFor...)
		}
	}

	ctr, err := testcontainers.Run(ctx, req)
	var c *Container
	if ctr != nil {
		c = &Container{DockerContainer: ctr, Settings: settings}
	}

	if err != nil {
		return c, fmt.Errorf("generic container: %w", err)
	}

	return c, nil
}
