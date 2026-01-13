package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPagination(t *testing.T) {
	p := DefaultPagination()
	assert.Equal(t, int64(1), p.Page)
	assert.Equal(t, int64(20), p.PageSize)
}

func TestPaginationSkipLimit(t *testing.T) {
	p := Pagination{Page: 2, PageSize: 25}
	assert.Equal(t, int64(25), p.Skip())
	assert.Equal(t, int64(25), p.Limit())
}
