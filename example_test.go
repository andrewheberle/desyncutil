package desyncutil_test

import (
	"fmt"
	"log"
	"net/url"

	"github.com/andrewheberle/desyncutil"
	"github.com/folbricht/desync"
)

func ExampleRateLimitedStore() {
	// parse store URL
	location, err := url.Parse("http://store:8080")
	if err != nil {
		log.Fatal(err)
	}

	// Set up remote store
	inner, err := desync.NewRemoteHTTPStore(location, desync.NewStoreOptionsWithDefaults())
	if err != nil {
		log.Fatal(err)
	}

	// Wrap the existing store, limiting throughput to 10 MB/s.
	store := desyncutil.NewRateLimitedStore(inner, 10*1024*1024)

	fmt.Println(store.String())
	// Output: rate-limited(http://store:8080/)
}
