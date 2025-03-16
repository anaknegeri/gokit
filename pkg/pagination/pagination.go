// Package pagination provides utilities for pagination in database queries
package pagination

import (
	"math"

	"gorm.io/gorm"
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page" query:"page"`
	PageSize int `json:"pageSize" query:"pageSize"`
}

// PaginationMeta contains metadata about pagination results
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// PaginationResult represents paginated results with data and metadata
type PaginationResult struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// Paginator handles paginating database queries
type Paginator struct {
	db *gorm.DB
}

// NewPaginator creates a new paginator with the provided database connection
func NewPaginator(db *gorm.DB) *Paginator {
	return &Paginator{
		db: db,
	}
}

// Paginate performs pagination on a database query
func (p *Paginator) Paginate(params PaginationParams, result interface{}) (*PaginationResult, error) {
	// Default to page 1 if page is invalid
	if params.Page <= 0 {
		params.Page = 1
	}

	// Default to 10 items per page if page size is invalid
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// Calculate offset for the query
	offset := (params.Page - 1) * params.PageSize

	// Get total count of records
	var total int64
	if err := p.db.Count(&total).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))

	// Execute the query with pagination
	if err := p.db.Limit(params.PageSize).Offset(offset).Find(result).Error; err != nil {
		return nil, err
	}

	// Create and return the pagination result
	return &PaginationResult{
		Data: result,
		Meta: PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// GetPaginationFromRequest extracts pagination parameters from a request context
func GetPaginationFromRequest(c interface {
	QueryInt(string, int) int
}) PaginationParams {
	return PaginationParams{
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("pageSize", 10),
	}
}
