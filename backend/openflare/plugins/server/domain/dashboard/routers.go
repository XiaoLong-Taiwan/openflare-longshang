// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"net/http"

	"Wavelet/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetOverviewHandler 获取仪表盘概览数据。
// @Summary 获取仪表盘概览
// @Description 聚合节点与可观测性数据，需要管理员权限。traffic 返回滚动 24 小时起止、保留存取明细口径、4xx/5xx 百分比与平均 QPS（请求数/86400，estimated_qps 保留同值）；capacity 返回最近 24 小时快照覆盖、新鲜/过期/缺失节点数与最新快照时间，新鲜阈值为 120 秒。趋势包含首尾部分小时共 25 桶；压缩数组列顺序不变。summary_fallback 表示使用小时趋势回退，UV 与流量节点数不可用。
// @Tags openflare-dashboard
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=dashboard.OverviewPayload} "仪表盘概览"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 404 {object} response.Any "无权限或不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/d/dashboard/overview [get]
func GetOverviewHandler(c *gin.Context) {
	overview, err := GetOverview(c.Request.Context())
	if response.AbortBadRequestOnError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(overview))
}
