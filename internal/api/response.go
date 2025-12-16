package api

import "github.com/gin-gonic/gin"

type PaginatedResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondWithData(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"data": data})
}

func respondWithPagination(c *gin.Context, data interface{}, page, perPage, total int) {
	totalPages := (total + perPage - 1) / perPage
	c.JSON(200, PaginatedResponse{
		Data: data,
		Meta: Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func respondWithError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func respondNotFound(c *gin.Context, message string) {
	respondWithError(c, 404, "not_found", message)
}

func respondInternalError(c *gin.Context) {
	respondWithError(c, 500, "internal_error", "An unexpected error occurred")
}
