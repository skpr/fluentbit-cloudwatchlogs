package main

import (
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/flush"
)

var (
	cliAddr    = kingpin.Flag("addr", "Address to receive flush requests from Fluent Bit").Default(":8080").String()
	cliRegion  = kingpin.Arg("region", "Region where logs will be dispatched to.").Default(endpoints.ApSoutheast2RegionID).String()
	cliPrefix  = kingpin.Arg("prefix", "Prefix to apply to CloudWatch Logs groups.").Required().String()
	cliCluster = kingpin.Arg("cluster", "Cluster which this process resides.").Required().String()
)

func main() {
	kingpin.Parse()

	log.Println("Starting server")

	server := &flush.Server{Region: *cliAddr}

	http.HandleFunc("/", server.ServeHTTP)

	err := http.ListenAndServe(*cliAddr, nil)
	if err != nil {
		panic(err)
	}
}
