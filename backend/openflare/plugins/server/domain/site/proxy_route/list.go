package proxy_route

import (
	"context"
	"errors"
	"strings"

	"Wavelet/openflare/plugins/server/kernel/repository"
)

type ListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
}

type ListResult struct {
	Total    int64   `json:"total"`
	Results  []*View `json:"results"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

func ListPage(ctx context.Context, query ListQuery) (*ListResult, error) {
	query.Search = strings.TrimSpace(query.Search)
	if query.Page < 0 || query.PageSize < 0 || len(query.Search) > 256 {
		return nil, errors.New("代理列表查詢參數無效")
	}
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	if query.Page-1 > int(^uint(0)>>1)/query.PageSize {
		return nil, errors.New("代理列表查詢參數無效")
	}
	routes, total, err := repository.ListProxyRoutePage(ctx, query.Page, query.PageSize, query.Search)
	if err != nil {
		return nil, err
	}
	views, err := buildProxyRouteViews(ctx, routes)
	if err != nil {
		return nil, err
	}
	return &ListResult{Total: total, Results: views, Page: query.Page, PageSize: query.PageSize}, nil
}
