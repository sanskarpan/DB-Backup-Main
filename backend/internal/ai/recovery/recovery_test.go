package recovery

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeSource is a legitimate test double for RecoveryPointSource.
type fakeSource struct {
	points []RecoveryPoint
	err    error
}

func (f *fakeSource) AvailablePoints(_ context.Context, _ string) ([]RecoveryPoint, error) {
	return f.points, f.err
}

// fakeValidator validates points whose BackupID is in the ok set, and records
// every point it was asked to validate.
type fakeValidator struct {
	ok      map[string]bool
	err     error
	called  []string
	failAll bool
}

func (f *fakeValidator) Validate(_ context.Context, rp RecoveryPoint) (bool, string, error) {
	f.called = append(f.called, rp.BackupID)
	if f.err != nil {
		return false, "boom", f.err
	}
	if f.failAll {
		return false, "clean room rejected artifact", nil
	}
	return f.ok[rp.BackupID], "clean room ok", nil
}

// fakeRestorer records whether Restore was reached and with what target.
type fakeRestorer struct {
	called bool
	target string
	rp     RecoveryPoint
	err    error
}

func (f *fakeRestorer) Restore(_ context.Context, rp RecoveryPoint, target string) error {
	f.called = true
	f.target = target
	f.rp = rp
	return f.err
}

// fakeVerifier reports a fixed health outcome.
type fakeVerifier struct {
	ok     bool
	err    error
	called bool
}

func (f *fakeVerifier) Verify(_ context.Context, _ string) (bool, string, error) {
	f.called = true
	return f.ok, "verified", f.err
}

// threePoints builds a canonical field of 3 recovery points relative to now.
func threePoints(now time.Time) []RecoveryPoint {
	return []RecoveryPoint{
		{BackupID: "p1", DBName: "app", Time: now.Add(-3 * time.Hour), Kind: KindFull, SizeBytes: 100, Immutable: true},
		{BackupID: "p2", DBName: "app", Time: now.Add(-1 * time.Hour), Kind: KindFull, SizeBytes: 120, Immutable: false},
		{BackupID: "p3", DBName: "app", Time: now.Add(-2 * time.Hour), Kind: KindIncremental, SizeBytes: 40, Immutable: false},
	}
}

func TestPlanPicksMostRecentValidatedPoint(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	// p1 and p2 validate; p3 does not. Expect p2 (most recent validated).
	val := &fakeValidator{ok: map[string]bool{"p1": true, "p2": true}}
	a := NewAssistant(src, val, &fakeRestorer{}, &fakeVerifier{ok: true})

	plan, err := a.Plan(context.Background(), Incident{DBName: "app", Symptom: "corruption"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if plan.ChosenPoint.BackupID != "p2" {
		t.Fatalf("expected chosen point p2, got %q (score=%.2f, rationale=%s)",
			plan.ChosenPoint.BackupID, plan.Score, plan.Rationale)
	}

	wantOrder := []string{StepDiagnose, StepSelect, StepCleanRoomValidate, StepRestore, StepVerify}
	if len(plan.Steps) != len(wantOrder) {
		t.Fatalf("expected %d steps, got %d", len(wantOrder), len(plan.Steps))
	}
	for i, want := range wantOrder {
		if plan.Steps[i].Name != want {
			t.Errorf("step %d = %q, want %q", i, plan.Steps[i].Name, want)
		}
	}
}

func TestPlanPrefersValidatedOverMoreRecentUnvalidated(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	// Only p1 (oldest) validates; p2 (newest) does not. Validation must
	// dominate recency, so p1 must win.
	val := &fakeValidator{ok: map[string]bool{"p1": true}}
	a := NewAssistant(src, val, &fakeRestorer{}, &fakeVerifier{ok: true})

	plan, err := a.Plan(context.Background(), Incident{DBName: "app"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if plan.ChosenPoint.BackupID != "p1" {
		t.Fatalf("expected validated point p1 to win, got %q", plan.ChosenPoint.BackupID)
	}
}

func TestPlanExcludesPointsAfterTargetTime(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	val := &fakeValidator{ok: map[string]bool{"p1": true, "p2": true, "p3": true}}
	a := NewAssistant(src, val, &fakeRestorer{}, &fakeVerifier{ok: true})

	// Target between p3 (-2h) and p2 (-1h); p2 is newer than target -> excluded.
	target := now.Add(-90 * time.Minute)
	plan, err := a.Plan(context.Background(), Incident{DBName: "app", TargetTime: &target})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if plan.ChosenPoint.BackupID == "p2" {
		t.Fatalf("p2 is newer than the target time and must not be chosen")
	}
	// The validator must never have been asked about the excluded p2.
	for _, id := range val.called {
		if id == "p2" {
			t.Fatalf("validator was probed for excluded point p2")
		}
	}
}

func TestExecuteHealthyPathSucceeds(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	val := &fakeValidator{ok: map[string]bool{"p1": true, "p2": true}}
	rst := &fakeRestorer{}
	vrf := &fakeVerifier{ok: true}
	a := NewAssistant(src, val, rst, vrf)

	plan, err := a.Plan(context.Background(), Incident{DBName: "app", Symptom: "outage"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	res, err := a.Execute(context.Background(), plan, "/tmp/recovered.db")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got failure at %q", res.Failed)
	}
	if res.Failed != "" {
		t.Fatalf("expected no failed step, got %q", res.Failed)
	}
	if !rst.called {
		t.Fatal("restore step was not reached on the healthy path")
	}
	if rst.target != "/tmp/recovered.db" {
		t.Fatalf("restore target = %q, want /tmp/recovered.db", rst.target)
	}
	if rst.rp.BackupID != plan.ChosenPoint.BackupID {
		t.Fatalf("restored %q, want chosen %q", rst.rp.BackupID, plan.ChosenPoint.BackupID)
	}
	if !vrf.called {
		t.Fatal("verify step was not reached")
	}
	wantCompleted := []string{StepDiagnose, StepSelect, StepCleanRoomValidate, StepRestore, StepVerify}
	if len(res.Completed) != len(wantCompleted) {
		t.Fatalf("completed = %v, want %v", res.Completed, wantCompleted)
	}
	for i, want := range wantCompleted {
		if res.Completed[i] != want {
			t.Errorf("completed[%d] = %q, want %q", i, res.Completed[i], want)
		}
	}
}

func TestExecuteFailsClosedOnValidationFailure(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	// failAll: clean-room validation rejects every point, including at execute.
	val := &fakeValidator{failAll: true}
	rst := &fakeRestorer{}
	vrf := &fakeVerifier{ok: true}
	a := NewAssistant(src, val, rst, vrf)

	plan, err := a.Plan(context.Background(), Incident{DBName: "app", Symptom: "ransomware"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	res, err := a.Execute(context.Background(), plan, "/tmp/recovered.db")
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected fail-closed failure, got success")
	}
	if res.Failed != StepCleanRoomValidate {
		t.Fatalf("expected failure at %q, got %q", StepCleanRoomValidate, res.Failed)
	}
	if rst.called {
		t.Fatal("restore must NOT be reached after a failed clean-room validation")
	}
	if vrf.called {
		t.Fatal("verify must NOT be reached after a failed clean-room validation")
	}
	// Only diagnose and select completed before the fail-closed stop.
	wantCompleted := []string{StepDiagnose, StepSelect}
	if len(res.Completed) != len(wantCompleted) {
		t.Fatalf("completed = %v, want %v", res.Completed, wantCompleted)
	}
}

func TestExecuteFailsClosedOnVerificationFailure(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	val := &fakeValidator{ok: map[string]bool{"p2": true, "p1": true}}
	rst := &fakeRestorer{}
	vrf := &fakeVerifier{ok: false}
	a := NewAssistant(src, val, rst, vrf)

	plan, err := a.Plan(context.Background(), Incident{DBName: "app"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	res, err := a.Execute(context.Background(), plan, "/tmp/recovered.db")
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected fail-closed failure on verification, got success")
	}
	if res.Failed != StepVerify {
		t.Fatalf("expected failure at %q, got %q", StepVerify, res.Failed)
	}
	if !rst.called {
		t.Fatal("restore should have run before verification")
	}
}

func TestExecutePropagatesRestoreError(t *testing.T) {
	now := time.Now()
	src := &fakeSource{points: threePoints(now)}
	val := &fakeValidator{ok: map[string]bool{"p2": true, "p1": true}}
	rst := &fakeRestorer{err: errors.New("disk full")}
	vrf := &fakeVerifier{ok: true}
	a := NewAssistant(src, val, rst, vrf)

	plan, err := a.Plan(context.Background(), Incident{DBName: "app"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	res, err := a.Execute(context.Background(), plan, "/tmp/recovered.db")
	if err == nil {
		t.Fatal("expected restore error to propagate")
	}
	if res.Success {
		t.Fatal("expected failure on restore error")
	}
	if res.Failed != StepRestore {
		t.Fatalf("expected failure at %q, got %q", StepRestore, res.Failed)
	}
	if vrf.called {
		t.Fatal("verify must not run after a restore error")
	}
}

func TestPlanNoEligiblePointsReturnsSentinel(t *testing.T) {
	now := time.Now()
	// Only one point, newer than the target -> nothing eligible.
	src := &fakeSource{points: []RecoveryPoint{
		{BackupID: "px", DBName: "app", Time: now, Kind: KindFull},
	}}
	val := &fakeValidator{ok: map[string]bool{}}
	a := NewAssistant(src, val, &fakeRestorer{}, &fakeVerifier{ok: true})

	target := now.Add(-1 * time.Hour)
	plan, err := a.Plan(context.Background(), Incident{DBName: "app", TargetTime: &target})
	if !errors.Is(err, ErrNoRecoveryPoints) {
		t.Fatalf("expected ErrNoRecoveryPoints, got plan=%v err=%v", plan, err)
	}
	if plan != nil {
		t.Fatal("expected nil plan on no eligible points")
	}
}

func TestPlanSourceErrorPropagates(t *testing.T) {
	src := &fakeSource{err: errors.New("catalog unavailable")}
	a := NewAssistant(src, &fakeValidator{}, &fakeRestorer{}, &fakeVerifier{ok: true})

	_, err := a.Plan(context.Background(), Incident{DBName: "app"})
	if err == nil {
		t.Fatal("expected source error to propagate")
	}
}

func TestExecuteNilPlan(t *testing.T) {
	a := NewAssistant(&fakeSource{}, &fakeValidator{}, &fakeRestorer{}, &fakeVerifier{ok: true})
	_, err := a.Execute(context.Background(), nil, "/tmp/x")
	if !errors.Is(err, ErrNilPlan) {
		t.Fatalf("expected ErrNilPlan, got %v", err)
	}
}
