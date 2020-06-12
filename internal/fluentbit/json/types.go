package json

import (
	"time"
)

// Line
type Line struct {
	Timestamp  time.Time  `json:"timestamp"`
	Log        string     `json:"log"`
	Kubernetes Kubernetes `json:"kubernetes"`
}

// Kubernetes metadata which relates to a log line.
type Kubernetes struct {
	Namespace   string            `json:"namespace_name"`
	Pod         string            `json:"pod_name"`
	Container   string            `json:"container_name"`
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`
}
