// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var (
	pubsubClient *pubsub.Client
	slackClient  *slack.Client
)

// note that this variable will be present in the cloud runtime:
//  var projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")

var (
	project = os.Getenv("GOOGLE_CLOUD_PROJECT") // dictated by Google Cloud
	topic   = os.Getenv("PUBSUB_TOPIC")
	token   = os.Getenv("SLACK_TOKEN")
	secret  = os.Getenv("SLACK_SIGNING_SECRET")
)

func init() {
	var err error
	ctx := context.Background()

	pubsubClient, err = pubsub.NewClient(ctx, project)
	if err != nil {
		panic(err)
	}

	slackClient = slack.New(token)
}

// HandleSlackEvent is the entrypoint for the cloud function
func HandleSlackEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("error reading request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sv, err := slack.NewSecretsVerifier(r.Header, secret)
	if err != nil {
		log.Println("error creating secrets verifier:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		log.Println("error writing to secret verifier:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		log.Println("secret verifier says not authorized:", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	event, err := slackevents.ParseEvent(
		json.RawMessage(body),
		slackevents.OptionNoVerifyToken())
	if err != nil {
		log.Println("error parsing slack event:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("got event of type %q", event.Type)

	if event.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			log.Println("error parsing slack event:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
		return
	}

	if event.Type == slackevents.CallbackEvent {
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			log.Println("got an @mention for the app")
			_, _, err = slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			if err != nil {
				log.Println("error writing message back in response to @mention:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		case *slackevents.MessageEvent:
			for _, file := range ev.Files {
				log.Printf("found a file with name %q, filetype %q, permalink %q, public permalink %q",
					file.Name, file.Filetype, file.Permalink, file.PermalinkPublic)
			}
		}
	}

	fmt.Fprintln(w, "done")
	return

	var buf []byte // TODO

	topic := pubsubClient.Topic(topic)
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

	fmt.Fprintln(w, "submitted document to the pubsub topic\n")
}
