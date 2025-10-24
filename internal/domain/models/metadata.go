package models

import (
	"errors"
	"math"
	"strings"

	"slices"

	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

// Filters represents pagination and sorting options for list endpoints.
// It encapsulates the requested page number, number of items per page,
// the requested sort expression, and a safelist of allowed sort keys.
// Use this type to pass client-supplied pagination and sorting parameters
// through application layers and to validate/normalize them before use.
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

func NewFilters(page int, pageSize int, sort string, sortSafelist []string) (Filters, error) {
	if len(sortSafelist) == 0 {
		return Filters{}, errors.New("length of sortSafeList must be greater than 0")
	}
	return Filters{
		Page:         page,
		PageSize:     pageSize,
		Sort:         sort,
		SortSafelist: sortSafelist,
	}, nil
}

func (f Filters) Validate(v *validator.Validator) {
	// Check that the page and page_size parameters contain sensible values.
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	// Check that the sort parameter matches a value in the safelist.
	v.Check(validator.PermittedValue(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

// Check that the client-provided Sort field matches one of the entries in our safelist
// and if it does, extract the column name from the Sort field by stripping the leading
// hyphen character (if one exists).
func (f Filters) SortColumn() string {
	if slices.Contains(f.SortSafelist, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-")
	}
	// WARNING may result panic
	return f.SortSafelist[0]
}

// Return the sort direction ("ASC" or "DESC") depending on the prefix character of the
// Sort field.
func (f Filters) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

func (f Filters) Limit() int {
	return f.PageSize
}

func (f Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type Metadata struct {
	CurrentPage  int `json:"current_page"`
	PageSize     int `json:"page_size"`
	FirstPage    int `json:"first_page"`
	LastPage     int `json:"last_page"`
	TotalRecords int `json:"total_records"`
}

// The CalculateMetadata() function calculates the appropriate pagination metadata
// values given the total number of records, current page, and page size values. Note
// that the last page value is calculated using the math.Ceil() function, which rounds
// up a float to the nearest integer. So, for example, if there were 12 records in total
// and a page size of 5, the last page value would be math.Ceil(12/5) = 3.
func CalculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		// Note that we return an empty Metadata struct if there are no records.
		return Metadata{
			CurrentPage:  page,
			PageSize:     pageSize,
			FirstPage:    0,
			LastPage:     0,
			TotalRecords: 0,
		}
	}
	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}
