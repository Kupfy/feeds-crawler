package response

type PagedResponse[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Total int `json:"total"`
}
