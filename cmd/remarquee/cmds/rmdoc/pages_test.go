package rmdoc

import "testing"

func TestParsePageSelection1Based(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		total       int
		wantAll     bool
		wantPages   []int
		wantIndices []int
		wantErr     bool
	}{
		{name: "empty means all", spec: "", total: 3, wantAll: true, wantPages: []int{1, 2, 3}, wantIndices: []int{0, 1, 2}},
		{name: "single page", spec: "2", total: 3, wantPages: []int{2}, wantIndices: []int{1}},
		{name: "comma list", spec: "1, 3", total: 3, wantPages: []int{1, 3}, wantIndices: []int{0, 2}},
		{name: "range", spec: "2-4", total: 5, wantPages: []int{2, 3, 4}, wantIndices: []int{1, 2, 3}},
		{name: "mixed", spec: "1,3-4", total: 5, wantPages: []int{1, 3, 4}, wantIndices: []int{0, 2, 3}},
		{name: "zero", spec: "0", total: 3, wantErr: true},
		{name: "negative", spec: "-1", total: 3, wantErr: true},
		{name: "out of range", spec: "4", total: 3, wantErr: true},
		{name: "descending range", spec: "3-1", total: 3, wantErr: true},
		{name: "bad token", spec: "abc", total: 3, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePageSelection1Based(tt.spec, tt.total)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePageSelection1Based: %v", err)
			}
			if got.All != tt.wantAll {
				t.Fatalf("All=%v want=%v", got.All, tt.wantAll)
			}
			assertIntsEqual(t, "Pages1", got.Pages1, tt.wantPages)
			assertIntsEqual(t, "Indices0", got.Indices0, tt.wantIndices)
		})
	}
}

func TestFormatPages1Based(t *testing.T) {
	if got := formatPages1Based([]int{1, 3, 4}); got != "1,3,4" {
		t.Fatalf("formatPages1Based=%q", got)
	}
}

func assertIntsEqual(t *testing.T, name string, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s len=%d want=%d (got=%v want=%v)", name, len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]=%d want=%d (got=%v want=%v)", name, i, got[i], want[i], got, want)
		}
	}
}
