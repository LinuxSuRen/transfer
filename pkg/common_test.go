package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFillContainer(t *testing.T) {
	tests := []struct {
		name string
		txt  string
		size int
		want string
	}{{
		name: "same size",
		txt:  "hello",
		size: 5,
		want: "hello",
	}, {
		name: "smaller size",
		txt:  "hello",
		size: 3,
		want: "hello",
	}, {
		name: "lagger size",
		txt:  "hello",
		size: 8,
		want: "   hello",
	}}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillContainer(tt.txt, tt.size)
			assert.Equal(t, tt.want, result, "failed in case [%d]", i)
		})
	}
}

func TestFillContainerWithNumber(t *testing.T) {
	tests := []struct {
		name string
		num  int
		size int
		want string
	}{{
		name: "same size",
		num:  12345,
		size: 5,
		want: "12345",
	}, {
		num:  1089,
		size: 10,
		want: "      1089",
	}, {
		name: "smaller size",
		num:  12345,
		size: 3,
		want: "12345",
	}, {
		name: "lagger size",
		num:  12345,
		size: 8,
		want: "   12345",
	}}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillContainerWithNumber(tt.num, tt.size)
			assert.Equal(t, tt.want, result, "failed in case [%d]", i)
		})
	}
}
