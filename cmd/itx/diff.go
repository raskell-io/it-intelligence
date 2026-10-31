package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
)

// diffSnapshots prints the facts added and removed between two snapshots: what changed in the landscape.
func diffSnapshots(ctx context.Context, e *env, a, b string) error {
	read := func(id string) (map[string]bool, error) {
		var path string
		if err := e.store.Pool.QueryRow(ctx, `select path from snapshot where id = $1`, id).Scan(&path); err != nil {
			return nil, fmt.Errorf("snapshot %s: %w", id, err)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		out := map[string]bool{}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			out[sc.Text()] = true
		}
		return out, sc.Err()
	}
	A, err := read(a)
	if err != nil {
		return err
	}
	B, err := read(b)
	if err != nil {
		return err
	}
	var removed, added []string
	for l := range A {
		if !B[l] {
			removed = append(removed, l)
		}
	}
	for l := range B {
		if !A[l] {
			added = append(added, l)
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	for _, l := range removed {
		fmt.Println("- " + l)
	}
	for _, l := range added {
		fmt.Println("+ " + l)
	}
	fmt.Printf("%d removed, %d added\n", len(removed), len(added))
	return nil
}
