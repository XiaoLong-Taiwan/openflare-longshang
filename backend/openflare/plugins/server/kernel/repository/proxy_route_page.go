package repository

import (
	"context"
	"strings"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/pkg/util"
)

func ListProxyRoutePage(ctx context.Context, page, pageSize int, search string) ([]*model.ProxyRoute, int64, error) {
	query := DB(ctx).Model(&model.ProxyRoute{})
	if search != "" {
		pattern := "%" + util.EscapeLike(strings.ToLower(search)) + "%"
		domains := DB(ctx).Model(&model.ZoneDomain{}).Select("proxy_route_id").Where("LOWER(domain) LIKE ? ESCAPE '\\'", pattern)
		query = query.Where("LOWER(site_name) LIKE ? ESCAPE '\\' OR LOWER(origin_url) LIKE ? ESCAPE '\\' OR id IN (?)", pattern, pattern, domains)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	routes := make([]*model.ProxyRoute, 0)
	err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&routes).Error
	return routes, total, err
}
