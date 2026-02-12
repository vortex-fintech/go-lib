package schemaregistry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/sr"
)

type Client struct {
	registry *sr.Client
	timeout  time.Duration
}

type Config struct {
	URL     string
	Timeout int
}

type SchemaReference struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

type Schema struct {
	Schema     string            `json:"schema"`
	SchemaType string            `json:"schemaType"`
	References []SchemaReference `json:"references,omitempty"`
}

type RegistryClient interface {
	GetLatestSchema(subject string) (string, int, error)
	RegisterSchema(subject, schema string) (int, error)
	RegisterSchemaWithRefs(subject, schema string, refs []SchemaReference) (int, error)
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("schema registry URL is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	httpClient := &http.Client{Timeout: timeout}
	rawClient, err := sr.NewClient(
		sr.URLs(cfg.URL),
		sr.HTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("create schema registry client: %w", err)
	}

	return &Client{
		registry: rawClient,
		timeout:  timeout,
	}, nil
}

func (c *Client) RawClient() *sr.Client {
	return c.registry
}

func (c *Client) withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

func toSRSchema(schema string, refs []SchemaReference) sr.Schema {
	srRefs := make([]sr.SchemaReference, 0, len(refs))
	for _, ref := range refs {
		srRefs = append(srRefs, sr.SchemaReference{
			Name:    ref.Name,
			Subject: ref.Subject,
			Version: ref.Version,
		})
	}

	return sr.Schema{
		Schema:     schema,
		Type:       sr.TypeProtobuf,
		References: srRefs,
	}
}

func (c *Client) GetLatestSchema(subject string) (string, int, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	ss, err := c.registry.SchemaByVersion(ctx, subject, -1)
	if err != nil {
		return "", 0, err
	}

	return ss.Schema.Schema, ss.ID, nil
}

func (c *Client) RegisterSchema(subject, schema string) (int, error) {
	return c.RegisterSchemaWithRefs(subject, schema, nil)
}

func (c *Client) RegisterSchemaWithRefs(subject, schema string, refs []SchemaReference) (int, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	id, err := c.registry.RegisterSchema(ctx, subject, toSRSchema(schema, refs), -1, -1)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (c *Client) ValidateSchema(subject, schema string) (bool, error) {
	return c.ValidateSchemaWithRefs(subject, schema, nil)
}

func (c *Client) ValidateSchemaWithRefs(subject, schema string, refs []SchemaReference) (bool, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	result, err := c.registry.CheckCompatibility(ctx, subject, -1, toSRSchema(schema, refs))
	if err != nil {
		return false, err
	}

	return result.Is, nil
}

func (c *Client) GetAllSubjects() ([]string, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	subjects, err := c.registry.Subjects(ctx)
	if err != nil {
		return nil, err
	}

	return subjects, nil
}
