package util

import "testing"

func TestNormalizePage(t *testing.T) {
	tests := []struct {
		name         string
		pageNum      int32
		pageSize     int32
		wantPageNum  int
		wantPageSize int
		wantOffset   int
	}{
		{
			name:         "default values",
			pageNum:      0,
			pageSize:     0,
			wantPageNum:  1,
			wantPageSize: 10,
			wantOffset:   0,
		},
		{
			name:         "cap page size to max",
			pageNum:      0,
			pageSize:     100,
			wantPageNum:  1,
			wantPageSize: 50,
			wantOffset:   0,
		},
		{
			name:         "normal values",
			pageNum:      3,
			pageSize:     20,
			wantPageNum:  3,
			wantPageSize: 20,
			wantOffset:   40,
		},
	}

	for _, tt := range tests {
		gotPageNum, gotPageSize, gotOffset := NormalizePage(tt.pageNum, tt.pageSize)
		if gotPageNum != tt.wantPageNum || gotPageSize != tt.wantPageSize || gotOffset != tt.wantOffset {
			t.Fatalf("%s: got (%d, %d, %d), want (%d, %d, %d)",
				tt.name, gotPageNum, gotPageSize, gotOffset, tt.wantPageNum, tt.wantPageSize, tt.wantOffset)
		}
	}
}
