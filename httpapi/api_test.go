package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pharmacycounter/catalog"
	"pharmacycounter/domain"
	"pharmacycounter/query"
	"pharmacycounter/queue"
	"pharmacycounter/service"
	"pharmacycounter/store"
)

func TestAPIHealthAndDashboard(t *testing.T) {
	storage, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	business, err := service.New(storage, queue.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := business.SeedCounters([]domain.Counter{{Code: "C01", Name: "窗口一", Enabled: true, QueueLimit: 3}}); err != nil {
		t.Fatal(err)
	}
	server := New(business, query.New(storage), catalog.Default(), t.TempDir(), log.Default())
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	var payload struct {
		Data domain.Dashboard `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.WaitingCount != 0 {
		t.Fatalf("unexpected dashboard: %+v", payload.Data)
	}
}
