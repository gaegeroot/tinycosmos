package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
)

const testToken = "123456:TEST-TOKEN"

// apiCall is one request the bot made to the fake Telegram server.
type apiCall struct {
	method string
	fields map[string]string
}

// fakeTelegram is a minimal stand-in for the Bot API. It records every call
// and returns canned successful responses.
type fakeTelegram struct {
	t   *testing.T
	srv *httptest.Server

	mu          sync.Mutex
	calls       []apiCall
	memberTypes map[string]string // user_id -> chat member status
	failMethods map[string]bool   // methods that return an error
	sendsBefore int               // fail sendMessage after this many successes; <0 disables
}

func newFakeTelegram(t *testing.T) *fakeTelegram {
	t.Helper()
	f := &fakeTelegram{t: t, memberTypes: map[string]string{}, failMethods: map[string]bool{}, sendsBefore: -1}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

// newBot returns a go-telegram client pointed at the fake server.
func (f *fakeTelegram) newBot() *bot.Bot {
	f.t.Helper()
	b, err := bot.New(testToken, bot.WithServerURL(f.srv.URL), bot.WithSkipGetMe())
	if err != nil {
		f.t.Fatalf("bot.New: %v", err)
	}
	return b
}

func (f *fakeTelegram) serve(w http.ResponseWriter, r *http.Request) {
	prefix := "/bot" + testToken + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "bad path", http.StatusNotFound)
		return
	}
	method := strings.TrimPrefix(r.URL.Path, prefix)

	fields := map[string]string{}
	if err := r.ParseMultipartForm(1 << 20); err == nil && r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
	}

	f.mu.Lock()
	f.calls = append(f.calls, apiCall{method: method, fields: fields})
	fail := f.failMethods[method]
	if method == "sendMessage" && f.sendsBefore >= 0 {
		if f.sendsBefore == 0 {
			fail = true
		} else {
			f.sendsBefore--
		}
	}
	memberType := f.memberTypes[fields["user_id"]]
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if fail {
		fmt.Fprint(w, `{"ok":false,"error_code":400,"description":"Bad Request: injected failure"}`)
		return
	}

	var result any
	switch method {
	case "sendMessage":
		result = map[string]any{
			"message_id": len(f.calls),
			"date":       0,
			"chat":       map[string]any{"id": json.Number(fields["chat_id"]), "type": "group"},
		}
	case "getChatMember":
		if memberType == "" {
			memberType = "member"
		}
		result = map[string]any{
			"status": memberType,
			"user":   map[string]any{"id": json.Number(fields["user_id"]), "is_bot": false, "first_name": "x"},
		}
	default:
		result = true
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
}

// callsTo returns the recorded calls to one method, in order.
func (f *fakeTelegram) callsTo(method string) []apiCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []apiCall
	for _, c := range f.calls {
		if c.method == method {
			out = append(out, c)
		}
	}
	return out
}

// sentTexts returns the text of every sendMessage call, in order.
func (f *fakeTelegram) sentTexts() []string {
	var out []string
	for _, c := range f.callsTo("sendMessage") {
		out = append(out, c.fields["text"])
	}
	return out
}

func (f *fakeTelegram) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}
