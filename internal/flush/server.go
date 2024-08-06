package flush

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/aws/cloudwatchlogs/dispatcher"
	"github.com/skpr/fluentbit-cloudwatchlogs/internal/fluentbit/json"
)

const (
	// AnnotationProject is used to construct a CloudWatch Logs group.
	AnnotationProject = "fluentbit.skpr.io/project"
	// AnnotationEnvironment is used to construct a CloudWatch Logs group.
	AnnotationEnvironment = "fluentbit.skpr.io/environment"
	// AnnotationGroupOverride is used for overriding the default project/environment naming convention.
	AnnotationGroupOverride = "fluentbit.skpr.io/group-override"
)

// Server for handling flush requests.
type Server struct {
	// Client for interacting with CloudWatch Logs.
	Client *cloudwatchlogs.Client
	// Prefix to apply to CloudWatch Logs groups.
	Prefix string
	// Cluster which this process resides.
	Cluster string
	// Lock to ensure we only have one process pushing logs.
	lock sync.Mutex
	// Amount of events to keep before flushing.
	BatchSize int
	// Toggles on debugging.
	Debug bool
}

// ServeHTTP
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This ensures that we are only processing a single request at a time.
	// If we don't do this there is a chance that requests could compete with
	// each other if they are pushing to the same stream.
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Println("Parsing new request")

	lines, err := json.Parse(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Failed to parse request:", err)
		return
	}

	log.Println("Initialising dispatcher client")

	client, err := dispatcher.New(s.Client, s.BatchSize, s.Debug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("Failed to setup dispatcher:", err)
		return
	}

	for _, line := range lines {
		group, err := groupName(s.Prefix, s.Cluster, line.Kubernetes.Annotations)
		if err != nil {
			if s.Debug {
				log.Printf("skipping %s/%s because: %s\n", line.Kubernetes.Namespace, line.Kubernetes.Pod, err)
			}

			continue
		}

		err = client.Add(group, line.Kubernetes.Container, line.Timestamp, line.Log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("Failed to add log to dispatcher:", err)
			return
		}
	}

	err = client.Send(context.TODO())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("Failed to send logs:", err)
		return
	}
}

// Helper function to get the value from a Kubernetes resource annotation.
func getAnnotationValue(annotations map[string]string, key string) (string, error) {
	if _, ok := annotations[key]; !ok {
		return "", fmt.Errorf("not found: %s", key)
	}

	return annotations[key], nil
}

// Helper function to determine the group name for a Pod.
func groupName(prefix, cluster string, annotations map[string]string) (string, error) {
	if override, ok := annotations[AnnotationGroupOverride]; ok {
		return override, nil
	}

	project, err := getAnnotationValue(annotations, AnnotationProject)
	if err != nil {
		return "", err
	}

	environment, err := getAnnotationValue(annotations, AnnotationEnvironment)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/%s/%s/%s/%s", prefix, cluster, project, environment), nil
}
