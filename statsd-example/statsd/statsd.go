package statsd

import (
	"log"

	"gopkg.in/alexcesaro/statsd.v2"
)

var defaultClient *statsd.Client

func init() {
	var err error
	defaultClient, err = statsd.New(statsd.Address("statsd:8125")) // Connect to the UDP port 8125 by default.
	if err != nil {
		// If nothing is listening on the target port, an error is returned and
		// the returned client does nothing but is still usable. So we can
		// just log the error and go on.
		log.Print(err)
	}
	// defer defaultClient.Close()
}

func Close() {
	defaultClient.Close()
}

func Increment(bucket string) {
	defaultClient.Increment(bucket)
}
