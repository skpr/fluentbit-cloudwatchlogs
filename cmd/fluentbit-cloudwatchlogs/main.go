package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"gopkg.in/alecthomas/kingpin.v2"

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

	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	server := &flush.Server{
		Client:  cloudwatchlogs.New(sess),
		Prefix:  *cliPrefix,
		Cluster: *cliCluster,
		BatchSize: *cliBatch,
		Debug:   *cliDebug,
	}

	http.HandleFunc("/", server.ServeHTTP)

	err = http.ListenAndServe(*cliAddr, nil)
	if err != nil {
		panic(err)
	}
}
