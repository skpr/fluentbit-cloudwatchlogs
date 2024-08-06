package logger

import (
	"context"
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// Client client for handling log events.
type Client struct {
	// Client for interacting with CloudWatch Logs.
	client *cloudwatchlogs.Client
	// Group which events will be pushed to.
	Group string
	// Stream which events will be pushed to.
	Stream string
	// Amount of events to keep before flushing.
	batchSize int
	// Events stored in memory before being pushed.
	events []types.InputLogEvent
	// Lock to ensure logs are
	lock sync.Mutex
}

// New client which creates the log group, stream and returns a client for batching logs to it.
func New(ctx context.Context, client *cloudwatchlogs.Client, group, stream string, batchSize int) (*Client, error) {
	batch := &Client{
		Group:     group,
		Stream:    stream,
		client:    client,
		batchSize: batchSize,
	}

	err := PutLogGroup(ctx, client, group)
	if err != nil {
		return nil, err
	}

	err = PutLogStream(ctx, client, group, stream)
	if err != nil {
		return nil, err
	}

	return batch, nil
}

// Add event to the client.
func (c *Client) Add(ctx context.Context, event types.InputLogEvent) error {
	c.events = append(c.events, event)

	if len(c.events) >= c.batchSize {
		return c.Flush(ctx)
	}

	return nil
}

// Flush events stored in the client.
func (c *Client) Flush(ctx context.Context) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.Group),
		LogStreamName: aws.String(c.Stream),
		LogEvents:     c.events,
	}

	// Reset the logs back to
	c.events = []types.InputLogEvent{}

	return c.putLogEvents(ctx, input)
}

// PutLogEvents will attempt to execute and handle invalid tokens.
func (c *Client) putLogEvents(ctx context.Context, input *cloudwatchlogs.PutLogEventsInput) error {
	_, err := c.client.PutLogEvents(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

// PutLogGroup will attempt to create a log group and not return an error if it already exists.
func PutLogGroup(ctx context.Context, client *cloudwatchlogs.Client, name string) error {
	_, err := client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(name),
	})
	if err != nil {
		var e *types.ResourceAlreadyExistsException

		if errors.As(err, &e) {
			return nil
		}

		return err
	}

	return nil
}

// PutLogStream will attempt to create a log stream and not return an error if it already exists.
func PutLogStream(ctx context.Context, client *cloudwatchlogs.Client, group, stream string) error {
	_, err := client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
	})
	if err != nil {
		var e *types.ResourceAlreadyExistsException

		if errors.As(err, &e) {
			return nil
		}

		return err
	}

	return nil
}
