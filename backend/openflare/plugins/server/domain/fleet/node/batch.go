// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package node

import (
	"context"
	"errors"

	ofws "Wavelet/openflare/plugins/server/domain/fleet/websocket"
	"Wavelet/openflare/plugins/server/kernel/repository"
	"gorm.io/gorm"
)

type BatchNodeRequest struct {
	IDs     []uint `json:"ids"`
	Channel string `json:"channel"`
	TagName string `json:"tag_name"`
}

type BatchNodeResult struct {
	ID      uint   `json:"id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type BatchNodeResponse struct {
	Results []BatchNodeResult `json:"results"`
}

func RequestAgentUpdateBatch(ctx context.Context, input BatchNodeRequest) (*BatchNodeResponse, error) {
	input.IDs = normalizeBatchIDs(input.IDs)
	if len(input.IDs) == 0 || len(input.IDs) > 100 {
		return nil, errors.New("节点数量必须介于 1 到 100 之间")
	}
	result := &BatchNodeResponse{Results: make([]BatchNodeResult, 0, len(input.IDs))}
	for _, id := range input.IDs {
		view, err := RequestAgentUpdate(ctx, id, AgentUpdateInput{Channel: input.Channel, TagName: input.TagName})
		item := BatchNodeResult{ID: id, Success: err == nil}
		if err != nil {
			item.Message = safeBatchError(err)
		} else {
			item.Message = view.UpdateChannel
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func RequestForceSyncBatch(ctx context.Context, input BatchNodeRequest) (*BatchNodeResponse, error) {
	input.IDs = normalizeBatchIDs(input.IDs)
	if len(input.IDs) == 0 || len(input.IDs) > 100 {
		return nil, errors.New("节点数量必须介于 1 到 100 之间")
	}
	activeConfig, err := repository.GetActiveConfigVersion(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errNoActiveConfigVersion)
		}
		return nil, err
	}
	result := &BatchNodeResponse{Results: make([]BatchNodeResult, 0, len(input.IDs))}
	for _, id := range input.IDs {
		node, getErr := repository.GetOpenFlareNodeByID(ctx, id)
		item := BatchNodeResult{ID: id}
		if getErr != nil {
			item.Message = safeBatchError(getErr)
			result.Results = append(result.Results, item)
			continue
		}
		if !ofws.SendForceSyncConfig(node.NodeID, forceSyncConfigPayload{Version: activeConfig.Version, Checksum: activeConfig.Checksum}) {
			item.Message = errNodeForceSyncFailed
		} else {
			item.Success = true
			item.Message = "同步指令已下发"
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func normalizeBatchIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func safeBatchError(err error) string {
	if err == nil {
		return "節點操作失敗"
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errNodeNotFound
	}
	return "節點操作失敗"
}
