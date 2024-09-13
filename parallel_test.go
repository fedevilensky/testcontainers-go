package testcontainers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/testcontainers/testcontainers-go/wait"
)

func TestParallelContainers(t *testing.T) {
	tests := []struct {
		name      string
		reqs      ParallelRequest
		resLen    int
		expErrors int
	}{
		{
			name: "running two containers (one error)",
			reqs: ParallelRequest{
				{
					Image: "nginx",
					ExposedPorts: []string{
						"10080/tcp",
					},
					Started: true,
				},
				{
					Image: "bad bad bad",
					ExposedPorts: []string{
						"10081/tcp",
					},
					Started: true,
				},
			},
			resLen:    1,
			expErrors: 1,
		},
		{
			name: "running two containers (all errors)",
			reqs: ParallelRequest{
				{
					Image: "bad bad bad",
					ExposedPorts: []string{
						"10081/tcp",
					},
					Started: true,
				},
				{
					Image: "bad bad bad",
					ExposedPorts: []string{
						"10081/tcp",
					},
					Started: true,
				},
			},
			resLen:    0,
			expErrors: 2,
		},
		{
			name: "running two containers (success)",
			reqs: ParallelRequest{
				{
					Image: "nginx",
					ExposedPorts: []string{
						"10080/tcp",
					},
					Started: true,
				},
				{
					Image: "nginx",
					ExposedPorts: []string{
						"10081/tcp",
					},
					Started: true,
				},
			},
			resLen:    2,
			expErrors: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ParallelContainers(context.Background(), tc.reqs, ParallelContainersOptions{})
			if err != nil {
				require.NotZero(t, tc.expErrors)
				var e ParallelContainersError
				errors.As(err, &e)
				if len(e.Errors) != tc.expErrors {
					t.Fatalf("expected errors: %d, got: %d\n", tc.expErrors, len(e.Errors))
				}
			}

			for _, c := range res {
				c := c
				CleanupContainer(t, c)
			}

			if len(res) != tc.resLen {
				t.Fatalf("expected containers: %d, got: %d\n", tc.resLen, len(res))
			}
		})
	}
}

func TestParallelContainersWithReuse(t *testing.T) {
	const (
		postgresPort     = 5432
		postgresPassword = "test"
		postgresUser     = "test"
		postgresDb       = "test"
	)

	natPort := fmt.Sprintf("%d/tcp", postgresPort)

	req := Request{
		Image:        "postgis/postgis",
		Name:         "test-postgres",
		ExposedPorts: []string{natPort},
		Env: map[string]string{
			"POSTGRES_PASSWORD": postgresPassword,
			"POSTGRES_USER":     postgresUser,
			"POSTGRES_DATABASE": postgresDb,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithPollInterval(100 * time.Millisecond).
			WithOccurrence(2),
		Started: true,
		Reuse:   true,
	}

	parallelRequest := ParallelRequest{
		req,
		req,
		req,
	}

	ctx := context.Background()

	res, err := ParallelContainers(ctx, parallelRequest, ParallelContainersOptions{})
	if err != nil {
		var e ParallelContainersError
		errors.As(err, &e)
		t.Fatalf("expected errors: %d, got: %d\n", 0, len(e.Errors))
	}
	// Container is reused, only terminate first container
	CleanupContainer(t, res[0])
}
