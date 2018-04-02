package gtsr

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlopes/slack"
)

func handleInteractiveMessages() error {

	http.HandleFunc("/", interactionHandler)
	http.ListenAndServe(":8080", nil)

	return nil
}

func interactionHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	payload := r.PostFormValue("payload") // to get params value with key

	// TODO(nussey): ADD VERIFICATION

	// var callback AutoGenerated
	var callback slack.AttachmentActionCallback
	json.Unmarshal([]byte(payload), &callback)
	if err != nil {
		// TODO(nussey): don't panic
		panic(err)
	}
	fmt.Println(callback.CallbackID)
	fmt.Printf("%+v\n", callback.Actions[0].SelectedOptions[0].Value)
}