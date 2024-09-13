package cockroachdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"path/filepath"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var ErrTLSNotEnabled = fmt.Errorf("tls not enabled")

const (
	certsDir = "/tmp"

	defaultSQLPort   = "26257/tcp"
	defaultAdminPort = "8080/tcp"

	defaultUser      = "root"
	defaultPassword  = ""
	defaultDatabase  = "defaultdb"
	defaultStoreSize = "100%"
)

// Container represents the CockroachDB container type used in the module
type Container struct {
	*testcontainers.DockerContainer
	opts options
}

// MustConnectionString panics if the address cannot be determined.
func (c *Container) MustConnectionString(ctx context.Context) string {
	addr, err := c.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}
	return addr
}

// ConnectionString returns the dial address to open a new connection to CockroachDB.
func (c *Container) ConnectionString(ctx context.Context) (string, error) {
	port, err := c.MappedPort(ctx, defaultSQLPort)
	if err != nil {
		return "", err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	return connString(c.opts, host, port), nil
}

// TLSConfig returns config necessary to connect to CockroachDB over TLS.
func (c *Container) TLSConfig() (*tls.Config, error) {
	return connTLS(c.opts)
}

// Run creates an instance of the CockroachDB container type
func Run(ctx context.Context, img string, opts ...testcontainers.RequestCustomizer) (*Container, error) {
	o := defaultOptions()
	req := testcontainers.Request{
		Image: img,
		ExposedPorts: []string{
			defaultSQLPort,
			defaultAdminPort,
		},
		LifecycleHooks: []testcontainers.LifecycleHooks{
			{
				PreStarts: []testcontainers.CreatedContainerHook{
					func(ctx context.Context, ctr testcontainers.CreatedContainer) error {
						return addTLS(ctx, ctr, o)
					},
				},
			},
		},
		Started: true,
	}

	// apply options
	for _, opt := range opts {
		if apply, ok := opt.(Option); ok {
			apply(&o)
		}
		if err := opt.Customize(&req); err != nil {
			return nil, err
		}
	}

	// modify request
	for _, fn := range []modiferFunc{
		addEnvs,
		addCmd,
		addWaitingFor,
	} {
		if err := fn(&req, o); err != nil {
			return nil, err
		}
	}

	ctr, err := testcontainers.Run(ctx, req)
	var c *Container
	if ctr != nil {
		c = &Container{DockerContainer: ctr, opts: o}
	}

	if err != nil {
		return c, fmt.Errorf("generic container: %w", err)
	}

	return c, nil
}

type modiferFunc func(*testcontainers.Request, options) error

func addCmd(req *testcontainers.Request, opts options) error {
	req.Cmd = []string{
		"start-single-node",
		"--store=type=mem,size=" + opts.StoreSize,
	}

	// authN
	if opts.TLS != nil {
		if opts.User != defaultUser {
			return fmt.Errorf("unsupported user %s with TLS, use %s", opts.User, defaultUser)
		}
		if opts.Password != "" {
			return fmt.Errorf("cannot use password authentication with TLS")
		}
	}

	switch {
	case opts.TLS != nil:
		req.Cmd = append(req.Cmd, "--certs-dir="+certsDir)
	case opts.Password != "":
		req.Cmd = append(req.Cmd, "--accept-sql-without-tls")
	default:
		req.Cmd = append(req.Cmd, "--insecure")
	}
	return nil
}

func addEnvs(req *testcontainers.Request, opts options) error {
	if req.Env == nil {
		req.Env = make(map[string]string)
	}

	req.Env["COCKROACH_DATABASE"] = opts.Database
	req.Env["COCKROACH_USER"] = opts.User
	req.Env["COCKROACH_PASSWORD"] = opts.Password
	return nil
}

func addWaitingFor(req *testcontainers.Request, opts options) error {
	var tlsConfig *tls.Config
	if opts.TLS != nil {
		cfg, err := connTLS(opts)
		if err != nil {
			return err
		}
		tlsConfig = cfg
	}

	sqlWait := wait.ForSQL(defaultSQLPort, "pgx/v5", func(host string, port nat.Port) string {
		connStr := connString(opts, host, port)
		if tlsConfig == nil {
			return connStr
		}

		// register TLS config with pgx driver
		connCfg, err := pgx.ParseConfig(connStr)
		if err != nil {
			panic(err)
		}
		connCfg.TLSConfig = tlsConfig

		return stdlib.RegisterConnConfig(connCfg)
	})
	defaultStrategy := wait.ForAll(
		wait.ForHTTP("/health").WithPort(defaultAdminPort),
		sqlWait,
	)

	if req.WaitingFor == nil {
		req.WaitingFor = defaultStrategy
	} else {
		req.WaitingFor = wait.ForAll(req.WaitingFor, defaultStrategy)
	}

	return nil
}

func addTLS(ctx context.Context, ctr testcontainers.CreatedContainer, opts options) error {
	if opts.TLS == nil {
		return nil
	}

	caBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: opts.TLS.CACert.Raw,
	})
	files := map[string][]byte{
		"ca.crt":          caBytes,
		"node.crt":        opts.TLS.NodeCert,
		"node.key":        opts.TLS.NodeKey,
		"client.root.crt": opts.TLS.ClientCert,
		"client.root.key": opts.TLS.ClientKey,
	}
	for filename, contents := range files {
		if err := ctr.CopyToContainer(ctx, contents, filepath.Join(certsDir, filename), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func connString(opts options, host string, port nat.Port) string {
	user := url.User(opts.User)
	if opts.Password != "" {
		user = url.UserPassword(opts.User, opts.Password)
	}

	sslMode := "disable"
	if opts.TLS != nil {
		sslMode = "verify-full"
	}
	params := url.Values{
		"sslmode": []string{sslMode},
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     user,
		Host:     net.JoinHostPort(host, port.Port()),
		Path:     opts.Database,
		RawQuery: params.Encode(),
	}

	return u.String()
}

func connTLS(opts options) (*tls.Config, error) {
	if opts.TLS == nil {
		return nil, ErrTLSNotEnabled
	}

	keyPair, err := tls.X509KeyPair(opts.TLS.ClientCert, opts.TLS.ClientKey)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(opts.TLS.CACert)

	return &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{keyPair},
		ServerName:   "localhost",
	}, nil
}
