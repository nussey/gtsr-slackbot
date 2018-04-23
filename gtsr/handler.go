package gtsr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/nlopes/slack"
)

const noCallback = "NOCALLBACK"

// TODO(alex): better name pls
type callbackListener struct {
	callbacks map[string]*Messenger

	mutex *sync.Mutex
}

func (l *callbackListener) registerCallback(id string, msngr *Messenger) {
	if id == noCallback {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if _, ok := l.callbacks[id]; ok {
		panic("callback id collision")
	}

	l.callbacks[id] = msngr
}

func (l *callbackListener) unregisterCallback(id string) {
	if id == noCallback {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	delete(l.callbacks, id)
}

func (sb *SlackBot) handleInteractiveMessages() error {

	http.HandleFunc("/", sb.interactionHandler)
	http.ListenAndServe(":8080", nil)

	return nil
}

func (sb *SlackBot) interactionHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	err := r.ParseForm()
	if err != nil {
		// TODO(nussey): don't panic
		panic(err)
	}

	payload := r.PostFormValue("payload") // to get params value with key

	var callback slack.AttachmentActionCallback
	json.Unmarshal([]byte(payload), &callback)
	if err != nil {
		// TODO(nussey): don't panic
		panic(err)
	}

	if callback.Token != sb.token {
		fmt.Println("garbage or illegal callback handled")
		return
	}

	action := callback.Actions[0]
	callbackID := callback.CallbackID
	actionID := noCallback
	if action.Type == "button" {
		actionID = action.Value
	} else if action.Type == "select" {
		actionID = action.SelectedOptions[0].Value
	} else {
		fmt.Println("failed to parse callback")
		return
	}

	if actionID == noCallback {
		fmt.Println("no-op")
		return
	}

	sb.gm.listener.mutex.Lock()
	defer sb.gm.listener.mutex.Unlock()

	if sb.gm.listener.callbacks[callbackID] == nil {
		fmt.Println("unregistered callback!")
		return
	}
	sb.gm.listener.callbacks[callbackID].respond(actionID, true)
}
