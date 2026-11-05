package analytics

import "time"

type NodeAccessDaily struct {
	Day           string `gorm:"primaryKey"`
	NodeID        string `gorm:"primaryKey"`
	StatusCode    int32  `gorm:"primaryKey"`
	Source        string `gorm:"primaryKey"`
	RequestCount  uint64
	BytesSent     uint64
	RequestLength uint64
}

func (NodeAccessDaily) TableName() string { return "of_node_access_daily" }

type NodeAccessQuota struct {
	Day            string `gorm:"primaryKey"`
	NodeID         string `gorm:"primaryKey"`
	UsedBytes      uint64
	SeenBytes      uint64
	SampleBoundary string
}

func (NodeAccessQuota) TableName() string { return "of_node_access_quota" }

type NodeAccessBatch struct {
	BatchKey  string `gorm:"primaryKey"`
	CreatedAt time.Time
}

func (NodeAccessBatch) TableName() string { return "of_node_access_batches" }

type NodeAccessCheckpoint struct {
	Name      string `gorm:"primaryKey"`
	ThroughID uint64
}

func (NodeAccessCheckpoint) TableName() string { return "of_node_access_checkpoints" }

type NodeAccessOutbox struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement:false"`
	Payload   string
	CreatedAt time.Time
}

func (NodeAccessOutbox) TableName() string { return "of_node_access_outbox" }
