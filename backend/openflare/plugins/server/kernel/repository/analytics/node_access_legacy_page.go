package analytics

import (
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"context"
)

func LegacyNodeAccessPage(ctx context.Context, limit int) ([]analyticsmodel.NodeAccessLog, error) {
	return nodeAccessIndexPage(ctx, 0, limit, true)
}

func NodeAccessIndexPage(ctx context.Context, after uint64, limit int) ([]analyticsmodel.NodeAccessLog, error) {
	return nodeAccessIndexPage(ctx, after, limit, false)
}

func nodeAccessIndexPage(ctx context.Context, after uint64, limit int, pending bool) ([]analyticsmodel.NodeAccessLog, error) {
	conn, err := ChConn(ctx)
	if err != nil {
		return nil, err
	}
	condition := "id > ?"
	if pending {
		condition += " AND summary_tracked = false"
	}
	rows, err := conn.Query(ctx, "SELECT id,node_id,logged_at,remote_addr,region,host,path,user_agent,cache_status,status_code,bytes_sent,request_length,request_time_ms,created_at,summary_tracked FROM of_node_access_logs WHERE "+condition+" ORDER BY id LIMIT ?", after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []analyticsmodel.NodeAccessLog
	for rows.Next() {
		var row analyticsmodel.NodeAccessLog
		if err := rows.Scan(&row.ID, &row.NodeID, &row.LoggedAt, &row.RemoteAddr, &row.Region, &row.Host, &row.Path, &row.UserAgent, &row.CacheStatus, &row.StatusCode, &row.BytesSent, &row.RequestLength, &row.RequestTimeMs, &row.CreatedAt, &row.SummaryTracked); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
