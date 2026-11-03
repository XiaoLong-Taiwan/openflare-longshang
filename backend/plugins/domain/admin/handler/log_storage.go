// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"Wavelet/pkg/response"
	"Wavelet/plugins/domain/admin/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type logStorageCleanupRequest struct {
	Category string `json:"category"`
	Days     *int   `json:"days"`
}

// GetLogStorage returns log category row counts and storage sizes.
// @Summary Get log storage overview
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 403 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/v1/admin/logs/storage [get]
func GetLogStorage(c *gin.Context) {
	result, err := service.LogStorageOverview(c.Request.Context())
	if err != nil {
		response.AbortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OK(result))
}

// CleanupLogStorage clears one or all log categories, optionally before a day cutoff.
// @Summary Cleanup log storage
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body logStorageCleanupRequest true "Cleanup request"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 403 {object} response.Any
// @Router /api/v1/admin/logs/storage/cleanup [post]
func CleanupLogStorage(c *gin.Context) {
	var request logStorageCleanupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.AbortWithError(c, http.StatusBadRequest, "invalid cleanup request")
		return
	}
	if request.Days != nil && *request.Days <= 0 {
		response.AbortWithError(c, http.StatusBadRequest, "days must be greater than zero")
		return
	}
	results, err := service.CleanupLogStorage(c.Request.Context(), request.Category, request.Days)
	if err != nil {
		response.AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.OK(results))
}
