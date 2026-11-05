package analytics

import "context"

func DeleteNodeAccessIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	conn, err := ChConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, "ALTER TABLE of_node_access_logs DELETE WHERE id IN ? SETTINGS mutations_sync = 1", ids)
}

func MarkNodeAccessIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	conn, err := ChConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, "ALTER TABLE of_node_access_logs UPDATE summary_tracked = true WHERE id IN ? SETTINGS mutations_sync = 1", ids)
}
