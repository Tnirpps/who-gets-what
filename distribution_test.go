package main

import (
	"errors"
	"testing"
)

func TestDistributionInvariants(t *testing.T) {
	participants := []string{"Анна", "Борис", "Вера", "Глеб", "Даша", "Егор", "Женя", "Зоя"}
	variants := []string{"A", "B", "C"}
	got, err := Distribute(participants, variants)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(participants) {
		t.Fatalf("got %d assignments, want %d", len(got), len(participants))
	}
	known, counts := map[string]bool{"A": true, "B": true, "C": true}, map[string]int{}
	for i, assignment := range got {
		if assignment.Participant != participants[i] {
			t.Errorf("assignment %d belongs to %q, want %q", i, assignment.Participant, participants[i])
		}
		if !known[assignment.Variant] {
			t.Errorf("unknown variant %q", assignment.Variant)
		}
		counts[assignment.Variant]++
	}
	min, max := len(participants), 0
	for _, variant := range variants {
		if counts[variant] < min {
			min = counts[variant]
		}
		if counts[variant] > max {
			max = counts[variant]
		}
	}
	if max-min > 1 {
		t.Fatalf("unbalanced counts: %v", counts)
	}
}

func TestFewerParticipantsDoNotRepeatVariants(t *testing.T) {
	got, err := Distribute([]string{"A", "B"}, []string{"1", "2", "3", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Variant == got[1].Variant {
		t.Fatalf("variant repeated: %#v", got)
	}
}

func TestSingleParticipant(t *testing.T) {
	got, err := Distribute([]string{"Solo"}, []string{"X", "Y"})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestSingleVariant(t *testing.T) {
	got, err := Distribute([]string{"A", "B", "C"}, []string{"Only"})
	if err != nil {
		t.Fatal(err)
	}
	for _, assignment := range got {
		if assignment.Variant != "Only" {
			t.Fatalf("unexpected assignment: %#v", assignment)
		}
	}
}

func TestEmptyInputsDoNotPanic(t *testing.T) {
	cases := []struct{ p, v []string }{{nil, nil}, {[]string{"A"}, nil}, {nil, []string{"X"}}}
	for _, tc := range cases {
		got, err := Distribute(tc.p, tc.v)
		if !errors.Is(err, errEmptyInput) {
			t.Errorf("got error %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got assignments %#v", got)
		}
	}
}

func TestClean(t *testing.T) {
	got := clean([]string{"  Анна ", "", "  ", "Борис"})
	if len(got) != 2 || got[0] != "Анна" || got[1] != "Борис" {
		t.Fatalf("unexpected cleaned list: %#v", got)
	}
}
