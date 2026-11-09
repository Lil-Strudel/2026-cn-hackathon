package notes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func fakeOpenAI(t *testing.T, status int, reply string) (*OpenAI, *map[string]any) {
	t.Helper()
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	o, err := NewOpenAI("test-key", "test-model")
	if err != nil {
		t.Fatal(err)
	}
	o.URL = server.URL
	return o, &got
}

func message(content string) string {
	return `{"status":"completed","usage":{"input_tokens":900,"output_tokens":120},"output":[
		{"type":"reasoning","summary":[]},
		{"type":"message","content":[` + content + `]}]}`
}

func TestWrite(t *testing.T) {
	text, _ := json.Marshal(`{"summary":"Cilium replaces kube-proxy.","notes":["eBPF routes traffic","Hubble shows flows"]}`)
	o, req := fakeOpenAI(t, http.StatusOK, message(`{"type":"output_text","text":`+string(text)+`}`))

	got, err := o.Write(context.Background(), "today we talk about Cilium")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "Cilium replaces kube-proxy." || !slices.Equal(got.Notes, []string{"eBPF routes traffic", "Hubble shows flows"}) {
		t.Errorf("notes = %+v", got)
	}
	if (*req)["model"] != "test-model" || (*req)["input"] != "today we talk about Cilium" || (*req)["store"] != false {
		t.Errorf("request = %v", *req)
	}
	format := (*req)["text"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["strict"] != true {
		t.Errorf("format = %v", format)
	}
}

func TestWriteFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		reply  string
		want   string
	}{
		{"http error", http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`, "429"},
		{"refusal", http.StatusOK, message(`{"type":"refusal","refusal":"no"}`), "refused"},
		{"incomplete", http.StatusOK, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`, "max_output_tokens"},
		{"no text", http.StatusOK, `{"status":"completed","output":[]}`, "no output text"},
		{"not the schema", http.StatusOK, message(`{"type":"output_text","text":"plain words"}`), "openai notes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, _ := fakeOpenAI(t, tt.status, tt.reply)
			_, err := o.Write(context.Background(), "transcript")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}
