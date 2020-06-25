package logger

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

const (
	ResourceAlreadyExistsCode = "ResourceAlreadyExistsException"
)

// Client client for handling log events.
type Client struct {
	Group string
	Stream string
	client *cloudwatchlogs.CloudWatchLogs
	token *string
}

// New client which creates the log group, stream and returns a client for batching logs to it.
func New(client *cloudwatchlogs.CloudWatchLogs, group, stream string) (*Client, error) {
	batch := &Client{
		Group: group,
		Stream: stream,
		client: client,
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

func (c *Client) PutBatchLogEvents(events []*cloudwatchlogs.InputLogEvent, size int) error {
	for _, chunk := range chunkMessages(events, size) {
		input := &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(c.Group),
			LogStreamName: aws.String(c.Stream),
			LogEvents:     chunk,
		}

		if c.token != nil {
			input.SequenceToken = c.token
		}

		err := c.PutLogEvents(input)
		if err != nil {
			return err
		}
	}

	return nil
}

// PutLogEvents will attempt to execute and handle invalid tokens.
func (c *Client) PutLogEvents(input *cloudwatchlogs.PutLogEventsInput) error {
	resp, err := c.client.PutLogEvents(input)
	if err != nil {
		if exception, ok := err.(*cloudwatchlogs.InvalidSequenceTokenException); ok {
			c.token = exception.ExpectedSequenceToken
			return c.PutLogEvents(input)
		}

		return err
	}

	c.token = resp.NextSequenceToken

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

// Helper function to split a batch of log event input into chunks.
func chunkMessages(messages []*cloudwatchlogs.InputLogEvent, size int) [][]*cloudwatchlogs.InputLogEvent {
	var chunks [][]*cloudwatchlogs.InputLogEvent

	for i := 0; i < len(messages); i += size {
		end := i + size

		if end > len(messages) {
			end = len(messages)
		}

		chunks = append(chunks, messages[i:end])
	}

	return chunks
}
