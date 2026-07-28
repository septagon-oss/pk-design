package delivery_test

// Validates: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// delivery_test.go proves the OSS source is native-editable, digest-pinnable,
// and round-trips atomically without a parallel token source.

import (
	"errors"
	"testing"
	"time"

	"github.com/septagon-oss/pk-design/pkg/delivery"
	"github.com/septagon-oss/pk-design/pkg/handoff"
	"github.com/septagon-oss/pk-design/pkg/themes"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

var generatedAt = time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)

func TestDefaultDeliveryIsWritableNativeOSS(t *testing.T) {
	t.Parallel()

	snapshot, err := delivery.DefaultSnapshot(generatedAt)
	if err != nil {
		t.Fatalf("DefaultSnapshot() error = %v", err)
	}
	if snapshot.Profile.ID != delivery.OSSProfileID ||
		snapshot.Profile.Kind != handoff.ProfileOSS {
		t.Fatalf("unexpected OSS profile: %#v", snapshot.Profile)
	}
	for _, token := range snapshot.Tokens {
		if !token.Origin.Writable ||
			token.Origin.Owner != delivery.OSSOwner ||
			token.Origin.SourceURI != themes.DefaultSourceURI {
			t.Fatalf("token is not bound to writable OSS DTCG source: %#v", token)
		}
	}

	bundle, err := delivery.DefaultFigmaVariables(generatedAt)
	if err != nil {
		t.Fatalf("DefaultFigmaVariables() error = %v", err)
	}
	if bundle.SnapshotDigest == "" ||
		bundle.Snapshot.Profile.ID != delivery.OSSProfileID ||
		bundle.Metadata["delivery"] != "native-editable" {
		t.Fatalf("unexpected native Figma delivery: %#v", bundle)
	}
}

func TestPlanDefaultChangesRoundTripsThroughCanonicalDTCG(t *testing.T) {
	t.Parallel()

	base, err := delivery.DefaultSnapshot(generatedAt)
	if err != nil {
		t.Fatal(err)
	}
	edited := base
	edited.Tokens = append([]handoff.Token(nil), base.Tokens...)
	var changed bool
	for index := range edited.Tokens {
		if edited.Tokens[index].Path == "/color/accent/default" {
			edited.Tokens[index].Value = "#145c4f"
			changed = true
		}
	}
	if !changed {
		t.Fatal("fixture has no /color/accent/default token")
	}
	changeSet, err := handoff.Diff(base, edited, handoff.Provenance{
		Source:      "figma-desktop",
		GeneratedAt: generatedAt.Add(time.Hour),
		Author:      "designer@example.com",
		Reason:      "approved accent refinement",
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	plan, err := delivery.PlanDefaultChanges(themes.DefaultSource(), changeSet)
	if err != nil {
		t.Fatalf("PlanDefaultChanges() error = %v", err)
	}
	if plan.Changes != 1 || plan.SourceDigest == "" || len(plan.Contents) == 0 {
		t.Fatalf("incomplete source plan: %#v", plan)
	}
	theme, err := themes.ParseDefaultSource(plan.Contents)
	if err != nil {
		t.Fatalf("rendered source is invalid: %v", err)
	}
	if got := theme.Tokens.Values["color.accent.default"]; got != "#145c4f" {
		t.Fatalf("rendered source accent = %v", got)
	}
}

func TestPlanDefaultChangesRejectsStaleAndForeignTransactions(t *testing.T) {
	t.Parallel()

	base, err := delivery.DefaultSnapshot(generatedAt)
	if err != nil {
		t.Fatal(err)
	}
	edited := base
	edited.Tokens = append([]handoff.Token(nil), base.Tokens...)
	edited.Tokens[0].Value = "#000001"
	changeSet, err := handoff.Diff(base, edited, handoff.Provenance{
		Source: "figma-desktop", GeneratedAt: generatedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	drifted := themes.DefaultSource()
	driftedTheme, err := themes.ParseDefaultSource(drifted)
	if err != nil {
		t.Fatal(err)
	}
	driftedTheme.Tokens.Values["color.signal"] = "#d8f35e"
	drifted, err = tokensJSON(driftedTheme)
	if err != nil {
		t.Fatal(err)
	}
	_, err = delivery.PlanDefaultChanges(drifted, changeSet)
	var conflict handoff.ConflictError
	if !errors.As(err, &conflict) || conflict.Code != handoff.ConflictBaseDigest {
		t.Fatalf("stale transaction error = %v", err)
	}

	foreign := changeSet
	after := *foreign.Changes[0].After
	after.Origin.Owner = "client/collect"
	foreign.Changes[0].After = &after
	if _, err := delivery.PlanDefaultChanges(themes.DefaultSource(), foreign); err == nil {
		t.Fatal("foreign token ownership was accepted")
	}
}

func tokensJSON(theme themes.Theme) ([]byte, error) {
	raw, err := tokens.DTCGJSON(theme.Tokens)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
