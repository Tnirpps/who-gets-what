package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func distributeTextFiles(participantsPath, variantsPath string) ([]Assignment, error) {
	participants, err := readTextList(participantsPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать участников: %w", err)
	}
	variants, err := readTextList(variantsPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать варианты: %w", err)
	}
	if len(participants) == 0 {
		return nil, errors.New("файл участников пуст")
	}
	if len(variants) == 0 {
		return nil, errors.New("файл вариантов пуст")
	}
	return Distribute(participants, variants)
}

func readTextList(path string) ([]string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".txt") {
		return nil, fmt.Errorf("ожидался TXT-файл: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimPrefix(string(data), "\uFEFF")
	return clean(strings.Split(text, "\n")), nil
}

func writeAssignmentsCSV(output io.Writer, assignments []Assignment) error {
	writer := csv.NewWriter(output)
	writer.UseCRLF = true
	if err := writer.Write([]string{"participant", "variant"}); err != nil {
		return err
	}
	for _, assignment := range assignments {
		if err := writer.Write([]string{assignment.Participant, assignment.Variant}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
