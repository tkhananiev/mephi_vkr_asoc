package service

import "testing"

func TestScopedGroupKeyIncludesConsoleProduct(t *testing.T) {
	owner := int64(7)
	productA := int64(11)
	productB := int64(22)
	base := "CVE-2024-0001::CWE-79::lib::1.0.0"

	keyA := scopedGroupKey(&owner, &productA, base)
	keyB := scopedGroupKey(&owner, &productB, base)
	if keyA == keyB {
		t.Fatalf("products A and B must not share group_key: %q", keyA)
	}
	wantA := "u:7:p:11:" + base
	wantB := "u:7:p:22:" + base
	if keyA != wantA {
		t.Fatalf("product A key=%q want %q", keyA, wantA)
	}
	if keyB != wantB {
		t.Fatalf("product B key=%q want %q", keyB, wantB)
	}
}

func TestScopedGroupKeyOwnerWithoutProduct(t *testing.T) {
	owner := int64(7)
	base := "::::.env::"
	got := scopedGroupKey(&owner, nil, base)
	want := "u:7:" + base
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestScopedGroupKeyGlobalUnscoped(t *testing.T) {
	base := "CVE-1::::pkg::"
	got := scopedGroupKey(nil, nil, base)
	if got != base {
		t.Fatalf("got %q want %q", got, base)
	}
}

func TestGroupKeyMatchesProductScope(t *testing.T) {
	owner := int64(7)
	productA := int64(11)
	productB := int64(22)

	keyA := "u:7:p:11:CVE-1::::lib::1"
	keyB := "u:7:p:22:CVE-1::::lib::1"
	legacy := "u:7:CVE-1::::lib::1"

	if !groupKeyMatchesProductScope(keyA, &owner, &productA) {
		t.Fatal("product A key must match product A")
	}
	if groupKeyMatchesProductScope(keyB, &owner, &productA) {
		t.Fatal("product B key must not match product A filter")
	}
	if !groupKeyMatchesProductScope(legacy, &owner, &productA) {
		t.Fatal("legacy keys defer to finding-link SQL; matcher must not exclude them")
	}
	if groupKeyMatchesProductScope(keyA, &owner, &productB) {
		t.Fatal("product A key must not match product B filter")
	}
}
