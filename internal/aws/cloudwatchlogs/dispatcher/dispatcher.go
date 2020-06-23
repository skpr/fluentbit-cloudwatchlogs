package dispatcher

import (
	"log"
	"time"

	"github.com/docker/docker/daemon/logger"
	"github.com/moby/moby/daemon/logger/awslogs"
)

// Client for orchestrating dispatching to CloudWatch Logs.
type Client struct {
	region string
	Groups map[string]Streams
}

// Streams which will be updated.
type Streams map[string]Lines

// Lines which will be pushed to CloudWatch Logs.
type Lines []*logger.Message

// New client for dispatching logs to CloudWatch Logs.
func New(region string) (*Client, error) {
	return &Client{
		region: region,
		Groups: make(map[string]Streams),
	}, nil
}

// Add log messages into a list which is grouped by LogGroup and Stream.
func (c *Client) Add(group, stream string, timestamp time.Time, message string) error {
	if _, ok := c.Groups[group]; !ok {
		c.Groups[group] = make(Streams)
	}

	c.Groups[group][stream] = append(c.Groups[group][stream], &logger.Message{
		Line:      []byte(message),
		Timestamp: timestamp,
	})

	return nil
}

// Send logs to CloudWatch Logs.
func (c *Client) Send() error {
	for group, streams := range c.Groups {
		for stream, lines := range streams {
			log.Printf("Pushing %d logs for %s/%s\n", len(lines), group, stream)

			cw, err := awslogs.New(logger.Context{
				Config: map[string]string{
					"ConfigRegion":      c.region,
					"ConfigCreateGroup": "true",
					"ConfigGroup":       group,
					"ConfigStream":      stream,
				},
			})
			if err != nil {
				return err
			}

			for _, line := range lines {
				err = cw.Log(line)
				if err != nil {
					return err
				}
			}

			err = cw.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
