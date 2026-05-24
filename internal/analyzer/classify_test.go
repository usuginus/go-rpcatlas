package analyzer

import (
	"testing"

	"github.com/usuginus/calltrail-go/internal/model"
	"github.com/usuginus/calltrail-go/internal/rules"
)

func TestClassifyUsesReceiverTypeBeforeCurrentFilePath(t *testing.T) {
	ruleSet, err := rules.Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	ref := model.CallRef{
		Symbol:   "c.repos.Foo.FindFoo",
		Receiver: "c.repos.Foo",
		Method:   "FindFoo",
		File:     "internal/usecase/foo.go",
	}
	scope := scopeInfo{
		receiverVar:    "c",
		receiverFields: map[string]string{"repos": "Repositories"},
		structFields: map[string]map[string]string{
			"Repositories": {"Foo": "FooRepository"},
		},
	}

	if got := classify(ref, scope, ruleSet.Layers); got != "repository" {
		t.Fatalf("classify = %q, want repository", got)
	}
}

func TestClassifyDoesNotUseCurrentFilePathForUtilityCalls(t *testing.T) {
	ruleSet, err := rules.Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	ref := model.CallRef{
		Symbol:   "tx.ReadOnlyTransaction",
		Receiver: "tx",
		Method:   "ReadOnlyTransaction",
		File:     "internal/usecase/foo.go",
	}

	if got := classify(ref, scopeInfo{receiverVar: "c"}, ruleSet.Layers); got != "unknown" {
		t.Fatalf("classify = %q, want unknown", got)
	}
}
