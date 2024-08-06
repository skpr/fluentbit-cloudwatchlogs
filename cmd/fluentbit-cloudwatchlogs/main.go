package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/alecthomas/kingpin/v2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/skpr/fluentbit-cloudwatchlogs/internal/flush"
)

var (
	cliAddr    = kingpin.Flag("addr", "Address to receive flush requests from Fluent Bit").Default(":8080").String()
	cliPrefix  = kingpin.Flag("prefix", "Prefix to apply to CloudWatch Logs groups.").Envar("FLUENTBIT_CLOUDWATCHLOGS_PREFIX").Required().String()
	cliCluster = kingpin.Flag("cluster", "Cluster which this process resides.").Envar("FLUENTBIT_CLOUDWATCHLOGS_CLUSTER").Required().String()
	cliBatch   = kingpin.Flag("batch", "Amount of records which will be batched and sent.").Envar("FLUENTBIT_CLOUDWATCHLOGS_BATCH").Default("256").Int()
	cliDebug   = kingpin.Flag("debug", "Toggles on debugging.").Envar("FLUENTBIT_CLOUDWATCHLOGS_DEBUG").Bool()
)

func main() {
	kingpin.Parse()

	log.Println("Starting server")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	server := &flush.Server{
		Client:    cloudwatchlogs.NewFromConfig(cfg),
		Prefix:    *cliPrefix,
		Cluster:   *cliCluster,
		BatchSize: *cliBatch,
		Debug:     *cliDebug,
	}

	http.HandleFunc("/", server.ServeHTTP)

	err = http.ListenAndServe(*cliAddr, nil)
	if err != nil {
		panic(err)
	}
}
