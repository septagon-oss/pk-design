// Package delivery exposes the canonical OSS design source as a governed,
// provider-neutral delivery and receive boundary.
package delivery

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// delivery.go owns the public OSS snapshot, native Figma-variable projection,
// and pure transactional plan for writing accepted token changes back to DTCG.
// Product and client layers extend this exact snapshot digest.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/septagon-oss/pk-design/pkg/figma"
	"github.com/septagon-oss/pk-design/pkg/handoff"
	"github.com/septagon-oss/pk-design/pkg/themes"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

const (
	// OSSProfileID is the public parent identity pinned by downstream products.
	OSSProfileID = "platformkit-oss"
	// OSSProductName is the display identity used by native design tools.
	OSSProductName = "PlatformKit OSS"
	// OSSOwner is the immutable source owner recorded on every managed token.
	OSSOwner = "septagon-oss/pk-design"
)

// SourcePlan is a pure, fully validated replacement plan. Computing it never
// mutates disk; CLIs may compare SourceDigest immediately before an atomic
// replacement.
type SourcePlan struct {
	SourceURI     string           `json:"sourceUri"`
	SourceDigest  string           `json:"sourceDigest"`
	CurrentDigest string           `json:"currentDigest"`
	NextSnapshot  handoff.Snapshot `json:"nextSnapshot"`
	Contents      []byte           `json:"-"`
	Changes       int              `json:"changes"`
}

// DefaultSnapshot returns the canonical editable OSS token graph.
func DefaultSnapshot(generatedAt time.Time) (handoff.Snapshot, error) {
	return SnapshotFromSource(
		themes.DefaultSource(),
		handoff.Provenance{
			Source:      "pk-design",
			GeneratedAt: generatedAt,
			Reason:      "canonical OSS DTCG delivery",
		},
	)
}

// SnapshotFromSource validates a canonical OSS DTCG source and projects it
// into the shared optimistic-concurrency contract.
func SnapshotFromSource(source []byte, provenance handoff.Provenance) (handoff.Snapshot, error) {
	theme, err := themes.ParseDefaultSource(source)
	if err != nil {
		return handoff.Snapshot{}, err
	}
	return handoff.FromSet(theme.Tokens, handoff.SetSnapshotOptions{
		Profile: handoff.Profile{
			ID:      OSSProfileID,
			Kind:    handoff.ProfileOSS,
			Version: theme.Version,
		},
		Provenance: provenance,
		Origin: handoff.Origin{
			Owner:     OSSOwner,
			Layer:     "oss",
			SourceURI: themes.DefaultSourceURI,
			Writable:  true,
		},
		Metadata: map[string]string{
			"theme":        theme.ID,
			"sourceFormat": "dtcg",
		},
	})
}

// DefaultFigmaVariables returns native editable Variables whose bindings point
// back to the canonical OSS DTCG source. It never contains raster design
// layers.
func DefaultFigmaVariables(generatedAt time.Time) (figma.Bundle, error) {
	snapshot, err := DefaultSnapshot(generatedAt)
	if err != nil {
		return figma.Bundle{}, err
	}
	return figma.Render(snapshot, figma.Options{
		Product: OSSProductName,
		Metadata: map[string]string{
			"delivery": "native-editable",
			"profile":  OSSProfileID,
		},
	})
}

// PlanDefaultChanges validates and applies a complete Figma-originated change
// set in memory, emits canonical DTCG JSON, reparses it, and verifies that the
// resulting source graph exactly matches the planned snapshot.
func PlanDefaultChanges(source []byte, changes handoff.ChangeSet) (SourcePlan, error) {
	current, err := SnapshotFromSource(source, handoff.Provenance{
		Source:      "pk-design",
		GeneratedAt: changes.Provenance.GeneratedAt,
		Reason:      "current OSS DTCG source",
	})
	if err != nil {
		return SourcePlan{}, err
	}
	currentDigest, err := current.Digest()
	if err != nil {
		return SourcePlan{}, err
	}
	next, err := handoff.Apply(current, changes)
	if err != nil {
		return SourcePlan{}, err
	}
	theme, err := themes.ParseDefaultSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	set := theme.Tokens
	for index, change := range changes.Changes {
		token := change.After
		if token == nil {
			token = change.Before
		}
		path, err := ownedSetPath(*token)
		if err != nil {
			return SourcePlan{}, fmt.Errorf("OSS token change[%d]: %w", index, err)
		}
		switch change.Operation {
		case handoff.OperationAdd, handoff.OperationUpdate:
			set.Values[path] = tokens.CopyValue(change.After.Value)
			if change.After.Type == "" {
				delete(set.Types, path)
			} else {
				set.Types[path] = change.After.Type
			}
			if strings.TrimSpace(change.After.Description) == "" {
				delete(set.Descriptions, path)
			} else {
				set.Descriptions[path] = change.After.Description
			}
		case handoff.OperationRemove:
			delete(set.Values, path)
			delete(set.Types, path)
			delete(set.Descriptions, path)
			delete(set.Extensions, path)
		default:
			return SourcePlan{}, fmt.Errorf(
				"OSS token change[%d]: unsupported operation %q",
				index,
				change.Operation,
			)
		}
	}
	contents, err := tokens.DTCGJSON(set)
	if err != nil {
		return SourcePlan{}, fmt.Errorf("OSS token changes: render DTCG source: %w", err)
	}
	contents = append(contents, '\n')
	compiled, err := SnapshotFromSource(contents, changes.Provenance)
	if err != nil {
		return SourcePlan{}, fmt.Errorf("OSS token changes: validate rendered DTCG source: %w", err)
	}
	nextDigest, err := next.Digest()
	if err != nil {
		return SourcePlan{}, err
	}
	compiledDigest, err := compiled.Digest()
	if err != nil {
		return SourcePlan{}, err
	}
	if compiledDigest != nextDigest {
		return SourcePlan{}, fmt.Errorf(
			"OSS token changes: rendered source graph %s does not match planned graph %s",
			compiledDigest,
			nextDigest,
		)
	}
	sum := sha256.Sum256(source)
	return SourcePlan{
		SourceURI:     themes.DefaultSourceURI,
		SourceDigest:  "sha256:" + hex.EncodeToString(sum[:]),
		CurrentDigest: currentDigest,
		NextSnapshot:  compiled,
		Contents:      contents,
		Changes:       len(changes.Changes),
	}, nil
}

func ownedSetPath(token handoff.Token) (string, error) {
	if token.Mode != handoff.DefaultMode {
		return "", fmt.Errorf("mode %q is not owned by the mode-independent OSS source", token.Mode)
	}
	segments, err := handoff.PathSegments(token.Path)
	if err != nil {
		return "", err
	}
	path := strings.Join(segments, ".")
	pointerSegments := make([]string, 0, len(segments)+2)
	pointerSegments = append(pointerSegments, "tokens")
	pointerSegments = append(pointerSegments, segments...)
	pointerSegments = append(pointerSegments, "$value")
	pointer, err := handoff.PathFromSegments(pointerSegments...)
	if err != nil {
		return "", err
	}
	want := handoff.Origin{
		Owner:     OSSOwner,
		Layer:     "oss",
		SourceURI: themes.DefaultSourceURI,
		Pointer:   pointer,
		Writable:  true,
	}
	if token.Origin != want {
		return "", fmt.Errorf(
			"token %q origin %#v is not the canonical OSS origin %#v",
			token.Path,
			token.Origin,
			want,
		)
	}
	return path, nil
}
