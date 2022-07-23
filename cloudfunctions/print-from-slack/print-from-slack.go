// Package p contains an HTTP Cloud Function.
package p

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// max file size 9MB (pubsub max is 10MB)
const maxFileSize = 10 * 1 << 20

var (
	pubsubClient *pubsub.Client
	slackClient  *slack.Client
)

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

	log.Println(string(body))

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

		log.Printf("inner event type was %s (%T)", innerEvent.Type, innerEvent.Data)

		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			log.Println("got an @mention for the app")
			_, _, err = slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			if err != nil {
				log.Println("error writing message back in response to @mention:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		case *slack.FileSharedEvent:
			// ev.File.Permalink points to a HTML page with a link to download
			// ev.File.PermalinkPublic points to the original file if the file was shared by URL
			// ev.File.PrivateURL points to the file contents and requires an authorization header
			// ev.File.PrivateURLPublic points to the file contents, requires an authorization header, and additionall adds heads to force a browser download
			log.Printf("a file was shared (ID %s)", ev.FileID)

			f, _, _, err := slackClient.GetFileInfoContext(ctx, ev.FileID, 1, 0)
			if err != nil {
				log.Printf("error retrieving file info for %s from slack: %v", ev.FileID, err)
				w.WriteHeader(http.StatusFailedDependency)
				return
			}

			json.NewEncoder(os.Stderr).Encode(f)

			if f.Size > maxFileSize {
				log.Printf("file size was too large: %d bytes (max is %d)", ev.File.Size, maxFileSize)
				w.WriteHeader(http.StatusFailedDependency)
				return
			}

			// create http request
			var content bytes.Buffer
			err = slackClient.GetFileContext(ctx, f.URLPrivate, &content)
			if err != nil {
				log.Println("error downloading a shared file:", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			log.Printf("downloaded a file of size %d bytes", content.Len())

			// push the file to pubsub
			topic := pubsubClient.Topic(topic)
			promise := topic.Publish(ctx, &pubsub.Message{
				Data: content.Bytes(),
			})

			// do not block the request on the result of the promise
			defer func() {
				_, err = promise.Get(ctx)
				if err != nil {
					log.Printf("error pushing message to pubsub: %v", err)
				}
			}()

			// report the result
			log.Printf("pushed a file of %d bytes to pubsub", content.Len())
		}
	}
}
