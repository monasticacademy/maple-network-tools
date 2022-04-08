// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
)

var client *pubsub.Client

var (
	project = os.Getenv("PROJECT")
	topic   = os.Getenv("TOPIC")
)

func init() {
	var err error
	ctx := context.Background()
	client, err = pubsub.NewClient(ctx, project)
	if err != nil {
		panic(err)
	}
}

// PrintGoogleDoc is the entrypoint for the cloud function
func PrintGoogleDoc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	document := r.URL.Query().Get("document") // a google doc ID
	if document == "" {
		http.Error(w,
			"missing 'document' URL parameter",
			http.StatusBadRequest)
		return
	}

	msg := map[string]string{"document": document}
	buf, err := json.Marshal(msg)
	if err != nil {
		http.Error(w,
			"error marshalling json: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	topic := client.Topic(topic)
	promise := topic.Publish(ctx, &pubsub.Message{
		Data: buf,
	})

	_, err = promise.Get(ctx)
	if err != nil {
		http.Error(w,
			"error pushing message to pubsub: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "submitted document %s to the pubsub topic\n", document)
}
