// Package recovery implements an agentic AI recovery assistant that turns a
// failure/incident signal plus the set of available backups into an ordered,
// executable recovery plan and then drives that plan through injected recovery
// capabilities (recovery-point discovery, clean-room validation, restore and
// verification).
//
// The assistant is deliberately decoupled from concrete engines: it depends
// only on the small capability interfaces declared in this package, so the real
// backup/restore engines can be wired in by the caller while tests supply
// legitimate in-package fakes.
package recovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// PointKind classifies a recovery point by the type of artifact it
// restores from.
type PointKind string

const (
	// KindFull is a self-contained full backup.
	KindFull PointKind = "Full"
	// KindIncremental is an incremental backup that layers on a base.
	KindIncremental PointKind = "Incremental"
	// KindJournal is a transaction-journal (WAL) based point-in-time source.
	KindJournal PointKind = "Journal"
)

// Point describes a single candidate the assistant can recover from.
type Point struct {
	// BackupID uniquely identifies the underlying backup artifact.
	BackupID string
	// DBName is the logical database the point belongs to.
	DBName string
	// Time is the moment the recovery point represents.
	Time time.Time
	// Kind is the artifact classification of the point.
	Kind PointKind
	// SizeBytes is the on-disk size of the recovery artifact.
	SizeBytes int64
	// Immutable indicates the artifact is protected by object-lock (WORM).
	Immutable bool
}

// Incident is the failure signal that triggers a recovery.
type Incident struct {
	// DBName is the database that needs recovering.
	DBName string
	// TargetTime, when set, is the point-in-time recovery objective: the
	// assistant must not choose a recovery point newer than this time.
	TargetTime *time.Time
	// Symptom is a free-form description of the observed failure.
	Symptom string
}

// Step is a single ordered action in a recovery plan.
type Step struct {
	// Name is the machine-readable step identifier (one of the Step* consts).
	Name string
	// Rationale explains, in human terms, why the step exists and what it does.
	Rationale string
}

// Plan is the ordered, executable output of Assistant.Plan.
type Plan struct {
	// Steps are the actions to run, in order.
	Steps []Step
	// ChosenPoint is the recovery point the plan will restore from.
	ChosenPoint Point
	// Score is the numeric score the chosen point earned.
	Score float64
	// Rationale is the overall explanation of the plan and point selection.
	Rationale string
}

// ExecutionResult reports the outcome of executing a recovery plan.
type ExecutionResult struct {
	// Completed lists, in order, the step names that finished successfully.
	Completed []string
	// Failed is the name of the step that failed, or empty on success.
	Failed string
	// Success is true only when every step completed.
	Success bool
}

// Step name constants identify the ordered phases of a recovery plan.
const (
	// StepDiagnose analyzes the incident symptom.
	StepDiagnose = "diagnose"
	// StepSelect records the chosen recovery point and its scoring.
	StepSelect = "select-recovery-point"
	// StepCleanRoomValidate validates the chosen point in an isolated sandbox.
	StepCleanRoomValidate = "clean-room-validate"
	// StepRestore restores the chosen point into the target.
	StepRestore = "restore"
	// StepVerify verifies the restored target is healthy.
	StepVerify = "verify"
)

// Scoring weights. Clean-room validation dominates so an unvalidated point can
// never outrank a validated one; recency is the primary tie-breaker among
// validated points; immutability is the final tie-breaker.
const (
	weightValidated = 1000.0
	weightRecency   = 100.0
	weightImmutable = 10.0
)

// ErrNoRecoveryPoints is returned by Plan when no recovery point satisfies the
// incident constraints.
var ErrNoRecoveryPoints = errors.New("recovery: no recovery point satisfies the incident constraints")

// ErrNilPlan is returned by Execute when it is called with a nil plan.
var ErrNilPlan = errors.New("recovery: plan is nil")

// PointSource discovers the recovery points available for a database.
type PointSource interface {
	// AvailablePoints returns the recovery points known for dbName.
	AvailablePoints(ctx context.Context, dbName string) ([]Point, error)
}

// CleanRoomValidator validates a recovery point by restoring it into an
// isolated clean room and checking its integrity, without touching production.
type CleanRoomValidator interface {
	// Validate reports whether rp is safe to recover from. The detail string
	// carries a human-readable explanation regardless of the outcome.
	Validate(ctx context.Context, rp *Point) (ok bool, detail string, err error)
}

// Restorer restores a recovery point into a concrete target.
type Restorer interface {
	// Restore restores rp into target.
	Restore(ctx context.Context, rp *Point, target string) error
}

// Verifier checks that a restored target is healthy.
type Verifier interface {
	// Verify reports whether target is healthy after a restore. The detail
	// string carries a human-readable explanation regardless of the outcome.
	Verify(ctx context.Context, target string) (ok bool, detail string, err error)
}

// Assistant is the agentic recovery planner and executor. It is safe to reuse
// across incidents and holds only the injected capability interfaces.
type Assistant struct {
	source    PointSource
	validator CleanRoomValidator
	restorer  Restorer
	verifier  Verifier
}

// NewAssistant builds an Assistant from the capability interfaces it drives.
func NewAssistant(source PointSource, validator CleanRoomValidator, restorer Restorer, verifier Verifier) *Assistant {
	return &Assistant{
		source:    source,
		validator: validator,
		restorer:  restorer,
		verifier:  verifier,
	}
}

// scoredPoint pairs a candidate with its computed score and probe result.
type scoredPoint struct {
	point     Point
	score     float64
	validated bool
	detail    string
}

// Plan chooses the best recovery point for the incident and emits an ordered,
// executable recovery plan.
//
// Selection rules, in priority order: the point must satisfy the incident's
// TargetTime (nothing newer than the objective is eligible); clean-room
// validated points are strongly preferred; among those, the most recent point
// wins; immutability is the final tie-breaker. The scoring behind the choice is
// explained in the returned plan's Rationale and in the select step.
func (a *Assistant) Plan(ctx context.Context, incident Incident) (*Plan, error) {
	points, err := a.source.AvailablePoints(ctx, incident.DBName)
	if err != nil {
		return nil, fmt.Errorf("recovery: discover points: %w", err)
	}

	candidates := eligiblePoints(points, incident.TargetTime)
	if len(candidates) == 0 {
		return nil, ErrNoRecoveryPoints
	}

	scored := a.scoreCandidates(ctx, candidates)
	best := scored[0]

	plan := &Plan{
		ChosenPoint: best.point,
		Score:       best.score,
		Rationale:   planRationale(incident, scored),
		Steps:       buildSteps(incident, &best),
	}
	return plan, nil
}

// eligiblePoints filters to the database's points that satisfy the target time.
func eligiblePoints(points []Point, target *time.Time) []Point {
	eligible := make([]Point, 0, len(points))
	for _, p := range points {
		if target != nil && p.Time.After(*target) {
			continue
		}
		eligible = append(eligible, p)
	}
	return eligible
}

// scoreCandidates probes and scores every candidate, returning them sorted best
// first. Ties are broken deterministically by recency then BackupID.
func (a *Assistant) scoreCandidates(ctx context.Context, candidates []Point) []scoredPoint {
	minT, maxT := timeRange(candidates)

	scored := make([]scoredPoint, 0, len(candidates))
	for i := range candidates {
		p := candidates[i]
		ok, detail := a.probe(ctx, &p)
		scored = append(scored, scoredPoint{
			point:     p,
			score:     score(&p, ok, minT, maxT),
			validated: ok,
			detail:    detail,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].point.Time.Equal(scored[j].point.Time) {
			return scored[i].point.Time.After(scored[j].point.Time)
		}
		return scored[i].point.BackupID < scored[j].point.BackupID
	})
	return scored
}

// probe runs the clean-room validator to learn whether a candidate is safe. A
// validator error is treated as "not validated" for scoring so a flaky probe
// downranks rather than aborts planning; the detail is preserved.
func (a *Assistant) probe(ctx context.Context, p *Point) (ok bool, detail string) {
	valid, msg, err := a.validator.Validate(ctx, p)
	if err != nil {
		return false, fmt.Sprintf("validation probe failed: %v", err)
	}
	return valid, msg
}

// timeRange returns the earliest and latest times among the candidates.
func timeRange(candidates []Point) (minT, maxT time.Time) {
	minT = candidates[0].Time
	maxT = candidates[0].Time
	for _, p := range candidates[1:] {
		if p.Time.Before(minT) {
			minT = p.Time
		}
		if p.Time.After(maxT) {
			maxT = p.Time
		}
	}
	return minT, maxT
}

// score computes a candidate's composite score from its validation state,
// recency (normalized across the candidate pool) and immutability.
func score(p *Point, validated bool, minT, maxT time.Time) float64 {
	var total float64
	if validated {
		total += weightValidated
	}
	if p.Immutable {
		total += weightImmutable
	}
	total += weightRecency * recencyNorm(p.Time, minT, maxT)
	return total
}

// recencyNorm maps a time onto [0,1] within the candidate window, where the
// newest point scores 1. A degenerate window (single time) scores 1.
func recencyNorm(t, minT, maxT time.Time) float64 {
	span := maxT.Sub(minT)
	if span <= 0 {
		return 1
	}
	return float64(t.Sub(minT)) / float64(span)
}

// buildSteps assembles the ordered, executable steps for the chosen point.
func buildSteps(incident Incident, best *scoredPoint) []Step {
	symptom := incident.Symptom
	if symptom == "" {
		symptom = "unspecified failure"
	}
	return []Step{
		{
			Name:      StepDiagnose,
			Rationale: fmt.Sprintf("Diagnose incident on %q: %s.", incident.DBName, symptom),
		},
		{
			Name:      StepSelect,
			Rationale: selectRationale(best),
		},
		{
			Name:      StepCleanRoomValidate,
			Rationale: "Restore the chosen point into an isolated clean room and validate it before touching the target; fail closed if validation does not pass.",
		},
		{
			Name:      StepRestore,
			Rationale: fmt.Sprintf("Restore backup %s into the recovery target.", best.point.BackupID),
		},
		{
			Name:      StepVerify,
			Rationale: "Verify the restored target is healthy; fail closed if verification does not pass.",
		},
	}
}

// selectRationale explains the score breakdown for the chosen point.
func selectRationale(best *scoredPoint) string {
	return fmt.Sprintf(
		"Chose backup %s (%s, %s) scoring %.2f: clean-room validated=%t, immutable=%t, most recent eligible point.",
		best.point.BackupID,
		best.point.Kind,
		best.point.Time.Format(time.RFC3339),
		best.score,
		best.validated,
		best.point.Immutable,
	)
}

// planRationale summarizes the whole candidate field and the winning choice.
func planRationale(incident Incident, scored []scoredPoint) string {
	var b strings.Builder
	target := "none"
	if incident.TargetTime != nil {
		target = incident.TargetTime.Format(time.RFC3339)
	}
	fmt.Fprintf(&b, "Evaluated %d eligible recovery point(s) for %q (target-time=%s). ",
		len(scored), incident.DBName, target)
	for i, s := range scored {
		fmt.Fprintf(&b, "[%d] %s score=%.2f (validated=%t, immutable=%t); ",
			i+1, s.point.BackupID, s.score, s.validated, s.point.Immutable)
	}
	fmt.Fprintf(&b, "Selected %s.", scored[0].point.BackupID)
	return b.String()
}

// Execute runs the plan's steps in order via the injected capabilities. It is
// fail-closed: a failed clean-room validation or verification (or any step
// error) stops execution immediately with Success=false, and later steps
// (notably restore) are not reached.
func (a *Assistant) Execute(ctx context.Context, plan *Plan, target string) (*ExecutionResult, error) {
	if plan == nil {
		return nil, ErrNilPlan
	}

	result := &ExecutionResult{Completed: make([]string, 0, len(plan.Steps))}
	for i := range plan.Steps {
		step := plan.Steps[i]
		ok, err := a.runStep(ctx, step.Name, &plan.ChosenPoint, target)
		if err != nil {
			result.Failed = step.Name
			return result, fmt.Errorf("recovery: step %q: %w", step.Name, err)
		}
		if !ok {
			result.Failed = step.Name
			return result, nil
		}
		result.Completed = append(result.Completed, step.Name)
	}

	result.Success = true
	return result, nil
}

// runStep dispatches a single step to the appropriate injected capability. The
// diagnose and select steps are planning artifacts with no side effects and
// always succeed; the remaining steps drive the real recovery interfaces.
func (a *Assistant) runStep(ctx context.Context, name string, rp *Point, target string) (bool, error) {
	switch name {
	case StepDiagnose, StepSelect:
		return true, nil
	case StepCleanRoomValidate:
		ok, _, err := a.validator.Validate(ctx, rp)
		return ok, err
	case StepRestore:
		if err := a.restorer.Restore(ctx, rp, target); err != nil {
			return false, err
		}
		return true, nil
	case StepVerify:
		ok, _, err := a.verifier.Verify(ctx, target)
		return ok, err
	default:
		return false, fmt.Errorf("recovery: unknown step %q", name)
	}
}
