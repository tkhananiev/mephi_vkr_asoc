package nvd

import (
	"reflect"
	"testing"
)

func TestCWEAliasesFromNVDWeaknesses(t *testing.T) {
	weaknesses := []struct {
		Description []struct {
			Value string `json:"value"`
		} `json:"description"`
	}{
		{
			Description: []struct {
				Value string `json:"value"`
			}{
				{Value: "CWE-79 Improper Neutralization of Input During Web Page Generation ('Cross-site Scripting')"},
			},
		},
		{
			Description: []struct {
				Value string `json:"value"`
			}{
				{Value: "cwe-917: something"},
			},
		},
		{
			Description: []struct {
				Value string `json:"value"`
			}{
				{Value: ""},
			},
		},
		{
			Description: []struct {
				Value string `json:"value"`
			}{
				{Value: "duplicate CWE-79 again"},
			},
		},
	}
	got := cweAliasesFromNVDWeaknesses(weaknesses)
	want := []string{"CWE-79", "CWE-917"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
