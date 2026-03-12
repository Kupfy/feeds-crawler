package request

type SearchRequest struct {
	Query string `form:"q"`
	Page  int    `form:"page"`
	Size  int    `form:"size,default=10" validate:"lte=50"`
}
