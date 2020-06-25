package dispatcher

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/aws/cloudwatchlogs/logger"
)

// Client for orchestrating dispatching to CloudWatch Logs.
type Client struct {
	client *cloudwatchlogs.CloudWatchLogs
	batchSize int
	Groups map[string]Streams
}

// Streams which will be updated.
type Streams map[string]Lines

// Lines which will be pushed to CloudWatch Logs.
type Lines []*cloudwatchlogs.InputLogEvent

// New client for dispatching logs to CloudWatch Logs.
func New(client *cloudwatchlogs.CloudWatchLogs) (*Client, error) {
	return &Client{
		client: client,
		Groups: make(map[string]Streams),
	}, nil
}

// Add log messages into a list which is grouped by LogGroup and Stream.
func (c *Client) Add(group, stream string, timestamp time.Time, message string) error {
	if _, ok := c.Groups[group]; !ok {
		c.Groups[group] = make(Streams)
	}

	c.Groups[group][stream] = append(c.Groups[group][stream], &cloudwatchlogs.InputLogEvent{
		Message:      aws.String(message),
		Timestamp: aws.Int64(timestamp.UnixNano() / int64(time.Millisecond)),
	})

	return nil
}

// Send logs to CloudWatch Logs.
func (c *Client) Send() error {
	for group, streams := range c.Groups {
		for stream, lines := range streams {
			log.Printf("Pushing %d logs for %s/%s\n", len(lines), group, stream)

			l, err := logger.New(c.client, group, stream)
			if err != nil {
				panic(err)
			}

			err = l.PutBatchLogEvents(lines, c.batchSize)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
