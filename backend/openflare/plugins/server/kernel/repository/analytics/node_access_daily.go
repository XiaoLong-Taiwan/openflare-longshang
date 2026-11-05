package analytics

import (
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"context"
)

func LegacyNodeAccessDaily(ctx context.Context) ([]analyticsmodel.NodeAccessDaily, error) {
	conn, err := ChConn(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx, "SELECT formatDateTime(logged_at, '%Y-%m-%d', 'UTC'), node_id, status_code, count(), sum(bytes_sent), sum(request_length) FROM of_node_access_logs WHERE summary_tracked = false GROUP BY 1, 2, 3")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []analyticsmodel.NodeAccessDaily
	for rows.Next() {
		var row analyticsmodel.NodeAccessDaily
		if err := rows.Scan(&row.Day, &row.NodeID, &row.StatusCode, &row.RequestCount, &row.BytesSent, &row.RequestLength); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
