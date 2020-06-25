package logger

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"log"
	"sync"
)

const (
	// ResourceAlreadyExistsCode is used to detect existing resources.
	ResourceAlreadyExistsCode = "ResourceAlreadyExistsException"
)

// Client client for handling log events.
type Client struct {
	// Client for interacting with CloudWatch Logs.
	client *cloudwatchlogs.CloudWatchLogs
	// Group which events will be pushed to.
	Group string
	// Stream which events will be pushed to.
	Stream string
	// Amount of events to keep before flushing.
	batchSize int
	// Events stored in memory before being pushed.
	events []*cloudwatchlogs.InputLogEvent
	// Lock to ensure logs are
	lock sync.Mutex
}

// New client which creates the log group, stream and returns a client for batching logs to it.
func New(client *cloudwatchlogs.CloudWatchLogs, group, stream string, batchSize int) (*Client, error) {
	batch := &Client{
		Group: group,
		Stream: stream,
		client: client,
		batchSize: batchSize,
	}

	err := PutLogGroup(client, group)
	if err != nil {
		return nil, err
	}

	err = PutLogStream(client, group, stream)
	if err != nil {
		return nil, err
	}

	return batch, nil
}

// Add event to the client.
func (c *Client) Add(event *cloudwatchlogs.InputLogEvent) error {
	c.events = append(c.events, event)

	if len(c.events) >= c.batchSize {
		return c.Flush()
	}

	return nil
}

// Flush events stored in the client.
func (c *Client) Flush() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.Group),
		LogStreamName: aws.String(c.Stream),
		LogEvents:     c.events,
	}

	// Reset the logs back to
	c.events = []*cloudwatchlogs.InputLogEvent{}

	return c.putLogEvents(input)
}


// PutLogEvents will attempt to execute and handle invalid tokens.
func (c *Client) putLogEvents(input *cloudwatchlogs.PutLogEventsInput) error {
	_, err := c.client.PutLogEvents(input)
	if err != nil {
		if exception, ok := err.(*cloudwatchlogs.InvalidSequenceTokenException); ok {
			log.Println("Refreshing token:", *input.LogGroupName, *input.LogStreamName)
			input.SequenceToken = exception.ExpectedSequenceToken
			return c.putLogEvents(input)
		}

		return err
	}

	return nil
}

// PutLogGroup will attempt to create a log group and not return an error if it already exists.
func PutLogGroup(client *cloudwatchlogs.CloudWatchLogs, name string) error {
	_, err := client.CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == ResourceAlreadyExistsCode {
				return nil
			}
		}

		return err
	}

	return nil
}

// PutLogStream will attempt to create a log stream and not return an error if it already exists.
func PutLogStream(client *cloudwatchlogs.CloudWatchLogs, group, stream string) error {
	_, err := client.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName: aws.String(group),
		LogStreamName: aws.String(stream),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == ResourceAlreadyExistsCode {
				return nil
			}
		}

		return err
	}

	return nil
}
