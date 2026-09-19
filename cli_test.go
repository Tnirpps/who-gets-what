package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadTextList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "participants.txt")
	if err := os.WriteFile(path, []byte("\uFEFF Анна \r\n\r\n Борис\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readTextList(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Анна", "Борис"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestReadTextListRejectsOtherExtensions(t *testing.T) {
	_, err := readTextList("participants.csv")
	if err == nil || !strings.Contains(err.Error(), "TXT") {
		t.Fatalf("got error %v", err)
	}
}

func TestWriteAssignmentsCSV(t *testing.T) {
	assignments := []Assignment{
		{Participant: "Анна, А.", Variant: "Вариант \"3\""},
		{Participant: "Борис\nмладший", Variant: "Вариант 1"},
	}
	var output bytes.Buffer
	if err := writeAssignmentsCSV(&output, assignments); err != nil {
		t.Fatal(err)
	}

	reader := csv.NewReader(strings.NewReader(output.String()))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"participant", "variant"},
		{"Анна, А.", "Вариант \"3\""},
		{"Борис\nмладший", "Вариант 1"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("got %#v, want %#v", rows, want)
	}
}
