package main

import (
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/aws/cloudwatchlogs/dispatcher"
	"github.com/skpr/fluentbit-cloudwatchlogs/internal/fluentbit/json"
)

var (
	cliAddr   = kingpin.Flag("addr", "Address to receive flush requests from Fluent Bit").Default(":8080").String()
	cliRegion = kingpin.Arg("region", "Region where logs will be displatched to.").Default(endpoints.ApSoutheast2RegionID).String()
)

func main() {
	kingpin.Parse()

	log.Println("Starting server")

	http.HandleFunc("/", handler)

	err := http.ListenAndServe(cliAddr, nil)
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	lines, err := json.Parse(r.Body)
	if err != nil {
		panic(err)
	}

	client, err := dispatcher.New("ap-southeast-2")
	if err != nil {
		panic(err)
	}

	for _, line := range lines {
		err := client.Add(line.Kubernetes.Namespace, line.Kubernetes.Pod, line.Timestamp, line.Log)
		if err != nil {
			panic(err)
		}
	}

	err = client.Send()
	if err != nil {
		panic(err)
	}
}
