package service

import (
	"fmt"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

type MutationSnapshotInput struct {
	IfUnmodifiedSince string
	DryRun            bool
	ShowDiff          bool
}

func mutationNeedsRemoteSnapshot(in MutationSnapshotInput) bool {
	return strings.TrimSpace(in.IfUnmodifiedSince) != "" || in.DryRun || in.ShowDiff
}

func assertModifiedAtMatches(got, expected, resource string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	got = strings.TrimSpace(got)
	if got == "" {
		return fail.NewValidation(
			resource+" has no modified_at metadata",
			"retry without --if-unmodified-since or ensure GET returns metadata",
		)
	}
	if !modifiedAtMatches(got, expected) {
		return fail.NewValidation(
			resource+" modified_at does not match --if-unmodified-since",
			fmt.Sprintf("remote modified_at is %s", got),
		)
	}
	return nil
}

type MutationDiffInput struct {
	Current      map[string]any
	Next         map[string]any
	StripCurrent func(map[string]any)
	StripNext    func(map[string]any)
	Semantic     func(left, right map[string]any) string
	JSONDiff     func(left, right map[string]any) string
}

func computeMutationDiff(in MutationDiffInput) (semantic, jsonDiff string) {
	left := cloneMap(in.Current)
	right := cloneMap(in.Next)
	if in.StripCurrent != nil {
		in.StripCurrent(left)
	}
	if in.StripNext != nil {
		in.StripNext(right)
	}
	semanticFn := in.Semantic
	if semanticFn == nil {
		semanticFn = func(_, _ map[string]any) string { return "changes present" }
	}
	jsonFn := in.JSONDiff
	if jsonFn == nil {
		jsonFn = DiffDashboardPayloads
	}
	return semanticFn(left, right), jsonFn(left, right)
}
