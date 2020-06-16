package flush

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/aws/cloudwatchlogs/dispatcher"
	"github.com/skpr/fluentbit-cloudwatchlogs/internal/fluentbit/json"
)

const (
	// AnnotationProject is used to construct a CloudWatch Logs group.
	AnnotationProject = "fluentbit.skpr.io/project"
	// AnnotationEnvironment is used to construct a CloudWatch Logs group.
	AnnotationEnvironment = "fluentbit.skpr.io/environment"
)

// Server for handling flush requests.
type Server struct {
	// Region where the logs should be put.
	Region string
	// Prefix to apply to CloudWatch Logs groups.
	Prefix string
	// Cluster which this process resides.
	Cluster string
	// Lock to ensure we only have one process pushing logs.
	lock sync.Mutex
}

// ServeHTTP
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This ensures that we are only processing a single request at a time.
	// If we don't do this there is a chance that requests could compete with
	// each other if they are pushing to the same stream.
	s.lock.Lock()
	defer s.lock.Unlock()

	lines, err := json.Parse(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Failed to parse request: %w", err)
		return
	}

	client, err := dispatcher.New(s.Region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("Failed to setup dispatcher: %w", err)
		return
	}

	for _, line := range lines {
		project, err := getAnnotationValue(line.Kubernetes.Annotations, AnnotationProject)
		if err != nil {
			log.Println("Failed to get annotation: %w", err)
			continue
		}

		environment, err := getAnnotationValue(line.Kubernetes.Annotations, AnnotationEnvironment)
		if err != nil {
			log.Println("Failed to get annotation: %w", err)
			continue
		}

		group := groupName(s.Prefix, s.Cluster, project, environment, line.Kubernetes.Container)

		err = client.Add(group, line.Kubernetes.Pod, line.Timestamp, line.Log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("Failed to add log to dispatcher: %w", err)
			return
		}
	}

	err = client.Send()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("Failed to send logs: %w", err)
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

// Helper function to generate a group name.
// This function is here to
func groupName(prefix, cluster, project, environment, container string) string {
	return fmt.Sprintf("/%s/%s/%s/%s/%s", prefix, cluster, project, environment, container)
}
