package main

import (
	"crypto/rand"
	"errors"
	"math/big"
)

var errEmptyInput = errors.New("participants and variants must not be empty")

// Assignment is a result fixed before the first wheel spin.
type Assignment struct {
	Participant string `json:"participant"`
	Variant     string `json:"variant"`
}

// Distribute creates the fairest possible assignment pool and shuffles it with
// cryptographically secure randomness. It never mutates its input slices.
func Distribute(participants, variants []string) ([]Assignment, error) {
	if len(participants) == 0 || len(variants) == 0 {
		return []Assignment{}, errEmptyInput
	}

	p, v := len(participants), len(variants)
	q, r := p/v, p%v
	pool := make([]string, 0, p)
	for _, variant := range variants {
		for range q {
			pool = append(pool, variant)
		}
	}

	// Selecting r different variants is equivalent to shuffling their indices
	// and taking the first r.
	indices := make([]int, v)
	for i := range indices {
		indices[i] = i
	}
	if err := cryptoShuffle(indices, func(i, j int) { indices[i], indices[j] = indices[j], indices[i] }); err != nil {
		return nil, err
	}
	for _, i := range indices[:r] {
		pool = append(pool, variants[i])
	}

	if err := cryptoShuffle(pool, func(i, j int) { pool[i], pool[j] = pool[j], pool[i] }); err != nil {
		return nil, err
	}

	assignments := make([]Assignment, p)
	for i, participant := range participants {
		assignments[i] = Assignment{Participant: participant, Variant: pool[i]}
	}
	return assignments, nil
}

// cryptoShuffle performs an unbiased Fisher-Yates shuffle using crypto/rand.
func cryptoShuffle[T any](items []T, swap func(i, j int)) error {
	for i := len(items) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		swap(i, int(n.Int64()))
	}
	return nil
}
