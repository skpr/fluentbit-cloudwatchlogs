package dispatcher

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/aws/cloudwatchlogs/logger"
)

// Client for orchestrating dispatching to CloudWatch Logs.
type Client struct {
	// Client for interacting with CloudWatch Logs.
	client *cloudwatchlogs.Client
	// Amount of events to keep before pushing.
	batchSize int
	// Content which will be pushed to CloudWatch Logs.
	Groups map[string]Streams
	// Turns on debugging output.
	debug bool
}

// Streams which will be updated.
type Streams map[string]Lines

// Lines which will be pushed to CloudWatch Logs.
type Lines []types.InputLogEvent

// New client for dispatching logs to CloudWatch Logs.
func New(client *cloudwatchlogs.Client, batchSize int, debug bool) (*Client, error) {
	return &Client{
		client:    client,
		Groups:    make(map[string]Streams),
		batchSize: batchSize,
		debug:     debug,
	}, nil
}

// Add log messages into a list which is grouped by LogGroup and Stream.
func (c *Client) Add(group, stream string, timestamp time.Time, message string) error {
	if _, ok := c.Groups[group]; !ok {
		c.Groups[group] = make(Streams)
	}

	c.Groups[group][stream] = append(c.Groups[group][stream], types.InputLogEvent{
		Message:   aws.String(message),
		Timestamp: aws.Int64(timestamp.UnixNano() / int64(time.Millisecond)),
	})

	return nil
}

// Send logs to CloudWatch Logs.
func (c *Client) Send(ctx context.Context) error {
	for group, streams := range c.Groups {
		for stream, lines := range streams {
			if c.debug {
				log.Printf("Pushing %d logs for %s/%s\n", len(lines), group, stream)
			}

			l, err := logger.New(ctx, c.client, group, stream, c.batchSize)
			if err != nil {
				return err
			}

			for _, line := range lines {
				err = l.Add(ctx, line)
				if err != nil {
					return err
				}
			}

			err = l.Flush(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
