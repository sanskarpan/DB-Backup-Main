package integrations

import (
	"context"
	"errors"
	"testing"
)

// countingIntegration embeds MockIntegration (defined in integration_test.go)
// and overrides CreateIncident to inject an error and count invocations.
type countingIntegration struct {
	*MockIntegration
	err   error
	calls int
}

func (c *countingIntegration) CreateIncident(_ context.Context, _ *Incident) (*IncidentResponse, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	return &IncidentResponse{ID: "ok", Status: "created"}, nil
}

func newCounting(name string, err error) *countingIntegration {
	return &countingIntegration{MockIntegration: &MockIntegration{typ: IntegrationTypeJira, name: name}, err: err}
}

func TestIncidentDispatcher_DispatchesToAll(t *testing.T) {
	a := newCounting("a", nil)
	b := newCounting("b", nil)
	d := NewIncidentDispatcher(a, b)

	if !d.Enabled() {
		t.Fatal("dispatcher should be enabled with integrations")
	}
	if len(d.Integrations()) != 2 {
		t.Fatalf("Integrations len = %d, want 2", len(d.Integrations()))
	}

	if err := d.Dispatch(context.Background(), &Incident{Title: "boom"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Fatalf("expected each integration called once, got a=%d b=%d", a.calls, b.calls)
	}
}

func TestIncidentDispatcher_AggregatesErrors(t *testing.T) {
	errA := errors.New("jira down")
	a := newCounting("a", errA)
	b := newCounting("b", nil)
	c := newCounting("c", errors.New("teams down"))
	d := NewIncidentDispatcher(a, b, c)

	err := d.Dispatch(context.Background(), &Incident{Title: "boom"})
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	// Best-effort: every integration is still attempted despite earlier failures.
	if a.calls != 1 || b.calls != 1 || c.calls != 1 {
		t.Fatalf("all integrations should be attempted, got a=%d b=%d c=%d", a.calls, b.calls, c.calls)
	}
	if !errors.Is(err, errA) {
		t.Fatalf("aggregated error should wrap the first failure, got %v", err)
	}
}

func TestIncidentDispatcher_EmptyDisabled(t *testing.T) {
	d := NewIncidentDispatcher()
	if d.Enabled() {
		t.Fatal("empty dispatcher should be disabled")
	}
	if err := d.Dispatch(context.Background(), &Incident{Title: "boom"}); err != nil {
		t.Fatalf("Dispatch on empty dispatcher: %v", err)
	}
}
