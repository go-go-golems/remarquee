package serve

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/remarquee/pkg/rmfiles"
	"github.com/juruen/rmapi/model"
)

type notificationFailureAPI struct{ *fakeApiCtx }

func (f *notificationFailureAPI) SyncComplete() error { return errors.New("notification unavailable") }

func TestDeleteNotificationFailureReportsCommittedOutcome(t *testing.T) {
	f := &notificationFailureAPI{newFake()}
	ai := f.add("", "ai", model.DirectoryType)
	doc := f.add(ai.ID, "Document", model.DocumentType)
	svc := rmfiles.NewService(f, rmfiles.Config{DefaultRemoteDir: "/ai"})
	server := NewServer(svc, Options{})
	rr := do(t, server, http.MethodDelete, "/api/entries", map[string]any{"ids": []string{doc.ID}, "confirm": "DELETE", "recursive": false})
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Deleted      int                            `json:"deleted"`
		RemainingIDs []string                       `json:"remainingIds"`
		FailedID     string                         `json:"failedId"`
		Reconciled   bool                           `json:"reconciled"`
		Error        struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Deleted != 1 || len(body.RemainingIDs) != 0 || body.FailedID != "" || !body.Reconciled {
		t.Fatalf("incorrect committed outcome: %+v", body)
	}
	if body.Error.Code != "partial_delete" || !strings.Contains(body.Error.Message, "notification unavailable") {
		t.Fatalf("missing notification failure: %+v", body.Error)
	}
	listed, err := svc.List("/ai")
	if err != nil || len(listed) != 0 {
		t.Fatalf("deleted record remains in List: %v %v", listed, err)
	}
	if _, _, err := svc.Download(doc.ID); !errors.Is(err, rmfiles.ErrNotFound) {
		t.Fatalf("deleted ID download error=%v", err)
	}
}
