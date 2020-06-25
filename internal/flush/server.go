package flush

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
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
	// Client for interacting with CloudWatch Logs.
	Client *cloudwatchlogs.CloudWatchLogs
	// Prefix to apply to CloudWatch Logs groups.
	Prefix string
	// Cluster which this process resides.
	Cluster string
	// Lock to ensure we only have one process pushing logs.
	lock sync.Mutex
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

	lines, err := json.Parse(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Failed to parse request:", err)
		return
	}

	client, err := dispatcher.New(s.Client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("Failed to setup dispatcher:", err)
		return
	}

	for _, line := range lines {
		project, err := getAnnotationValue(line.Kubernetes.Annotations, AnnotationProject)
		if err != nil {
			if s.Debug {
				log.Printf("skipping %s/%s because annotation %s because: %s\n", line.Kubernetes.Namespace, line.Kubernetes.Pod, AnnotationProject, err)
			}

			continue
		}

		environment, err := getAnnotationValue(line.Kubernetes.Annotations, AnnotationEnvironment)
		if err != nil {
			if s.Debug {
				log.Printf("skipping %s/%s because annotation %s because: %s\n", line.Kubernetes.Namespace, line.Kubernetes.Pod, AnnotationEnvironment, err)
			}

			continue
		}

		group := groupName(s.Prefix, s.Cluster, project, environment, line.Kubernetes.Container)

		err = client.Add(group, line.Kubernetes.Pod, line.Timestamp, line.Log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("Failed to add log to dispatcher:", err)
			return
		}
	}

	err = client.Send()
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

// Helper function to generate a group name.
// This function is here to
func groupName(prefix, cluster, project, environment, container string) string {
	return fmt.Sprintf("/%s/%s/%s/%s/%s", prefix, cluster, project, environment, container)
}
