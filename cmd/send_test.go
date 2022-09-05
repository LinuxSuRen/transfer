package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckMissing(t *testing.T) {
	tests := []struct {
		name      string
		message   []byte
		wantIndex int
		wantOK    bool
	}{{
		name:      "missing",
		message:   []byte("miss000123"),
		wantIndex: 123,
		wantOK:    true,
	}, {
		name:      "missing",
		message:   []byte("miss   123"),
		wantIndex: 123,
		wantOK:    true,
	}, {
		name:      "missing with invalid number",
		message:   []byte("miss000aaa"),
		wantIndex: 0,
		wantOK:    false,
	}, {
		name:      "done",
		message:   []byte("done0000"),
		wantIndex: -1,
		wantOK:    true,
	}}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, ok := checkMissing(tt.message)
			assert.Equal(t, tt.wantIndex, index, "failed in case [%d]", i)
			assert.Equal(t, tt.wantOK, ok, "failed in case [%d]", i)
		})
	}
}
