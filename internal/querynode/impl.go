// Copyright (C) 2019-2020 Zilliz. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under the License.

package querynode

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/milvus-io/milvus/internal/log"
	"github.com/milvus-io/milvus/internal/proto/commonpb"
	"github.com/milvus-io/milvus/internal/proto/internalpb"
	"github.com/milvus-io/milvus/internal/proto/milvuspb"
	queryPb "github.com/milvus-io/milvus/internal/proto/querypb"
	"github.com/milvus-io/milvus/internal/util/typeutil"
)

func (node *QueryNode) GetComponentStates(ctx context.Context) (*internalpb.ComponentStates, error) {
	stats := &internalpb.ComponentStates{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
		},
	}
	code, ok := node.stateCode.Load().(internalpb.StateCode)
	if !ok {
		errMsg := "unexpected error in type assertion"
		stats.Status = &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
			Reason:    errMsg,
		}
		return stats, errors.New(errMsg)
	}
	info := &internalpb.ComponentInfo{
		NodeID:    Params.QueryNodeID,
		Role:      typeutil.QueryNodeRole,
		StateCode: code,
	}
	stats.State = info
	return stats, nil
}

func (node *QueryNode) GetTimeTickChannel(ctx context.Context) (*milvuspb.StringResponse, error) {
	return &milvuspb.StringResponse{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
			Reason:    "",
		},
		Value: Params.QueryTimeTickChannelName,
	}, nil
}

func (node *QueryNode) GetStatisticsChannel(ctx context.Context) (*milvuspb.StringResponse, error) {
	return &milvuspb.StringResponse{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
			Reason:    "",
		},
		Value: Params.StatsChannelName,
	}, nil
}

func (node *QueryNode) AddQueryChannel(ctx context.Context, in *queryPb.AddQueryChannelRequest) (*commonpb.Status, error) {
	collectionID := in.CollectionID
	if node.searchService == nil {
		errMsg := "null search service, collectionID = " + fmt.Sprintln(collectionID)
		status := &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
			Reason:    errMsg,
		}
		return status, errors.New(errMsg)
	}

	//if _, ok := node.searchService.searchCollections[in.CollectionID]; !ok {
	//	errMsg := "null search collection, collectionID = " + fmt.Sprintln(collectionID)
	//	status := &commonpb.Status{
	//		ErrorCode: commonpb.ErrorCode_UnexpectedError,
	//		Reason:    errMsg,
	//	}
	//	return status, errors.New(errMsg)
	//}

	// add search collection
	if !node.searchService.hasSearchCollection(collectionID) {
		node.searchService.addSearchCollection(collectionID)
		log.Debug("add search collection", zap.Any("collectionID", collectionID))
	}

	// add request channel
	sc := node.searchService.searchCollections[in.CollectionID]
	consumeChannels := []string{in.RequestChannelID}
	//consumeSubName := Params.MsgChannelSubName
	consumeSubName := Params.MsgChannelSubName + "-" + strconv.FormatInt(collectionID, 10) + "-" + strconv.Itoa(rand.Int())
	sc.searchMsgStream.AsConsumer(consumeChannels, consumeSubName)
	node.retrieveService.retrieveMsgStream.AsConsumer(consumeChannels, "RetrieveSubName")
	log.Debug("querynode AsConsumer: " + strings.Join(consumeChannels, ", ") + " : " + consumeSubName)

	// add result channel
	producerChannels := []string{in.ResultChannelID}
	sc.searchResultMsgStream.AsProducer(producerChannels)
	node.retrieveService.retrieveResultMsgStream.AsProducer(producerChannels)
	log.Debug("querynode AsProducer: " + strings.Join(producerChannels, ", "))

	// message stream need to asConsumer before start
	// add search collection
	if !node.searchService.hasSearchCollection(collectionID) {
		node.searchService.addSearchCollection(collectionID)
		log.Debug("add search collection", zap.Any("collectionID", collectionID))
	}
	sc.start()
	log.Debug("start search collection", zap.Any("collectionID", collectionID))

	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	return status, nil
}

func (node *QueryNode) RemoveQueryChannel(ctx context.Context, in *queryPb.RemoveQueryChannelRequest) (*commonpb.Status, error) {
	// if node.searchService == nil || node.searchService.searchMsgStream == nil {
	// 	errMsg := "null search service or null search result message stream"
	// 	status := &commonpb.Status{
	// 		ErrorCode: commonpb.ErrorCode_UnexpectedError,
	// 		Reason:    errMsg,
	// 	}

	// 	return status, errors.New(errMsg)
	// }

	// searchStream, ok := node.searchService.searchMsgStream.(*pulsarms.PulsarMsgStream)
	// if !ok {
	// 	errMsg := "type assertion failed for search message stream"
	// 	status := &commonpb.Status{
	// 		ErrorCode: commonpb.ErrorCode_UnexpectedError,
	// 		Reason:    errMsg,
	// 	}

	// 	return status, errors.New(errMsg)
	// }

	// resultStream, ok := node.searchService.searchResultMsgStream.(*pulsarms.PulsarMsgStream)
	// if !ok {
	// 	errMsg := "type assertion failed for search result message stream"
	// 	status := &commonpb.Status{
	// 		ErrorCode: commonpb.ErrorCode_UnexpectedError,
	// 		Reason:    errMsg,
	// 	}

	// 	return status, errors.New(errMsg)
	// }

	// // remove request channel
	// consumeChannels := []string{in.RequestChannelID}
	// consumeSubName := Params.MsgChannelSubName
	// // TODO: searchStream.RemovePulsarConsumers(producerChannels)
	// searchStream.AsConsumer(consumeChannels, consumeSubName)

	// // remove result channel
	// producerChannels := []string{in.ResultChannelID}
	// // TODO: resultStream.RemovePulsarProducer(producerChannels)
	// resultStream.AsProducer(producerChannels)

	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	return status, nil
}

func (node *QueryNode) WatchDmChannels(ctx context.Context, in *queryPb.WatchDmChannelsRequest) (*commonpb.Status, error) {
	dct := &watchDmChannelsTask{
		baseTask: baseTask{
			ctx:  ctx,
			done: make(chan error),
		},
		req:  in,
		node: node,
	}

	err := node.scheduler.queue.Enqueue(dct)
	if err != nil {
		status := &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
			Reason:    err.Error(),
		}
		log.Error(err.Error())
		return status, err
	}
	log.Debug("watchDmChannelsTask Enqueue done", zap.Any("collectionID", in.CollectionID))

	func() {
		err = dct.WaitToFinish()
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Debug("watchDmChannelsTask WaitToFinish done", zap.Any("collectionID", in.CollectionID))
	}()

	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	return status, nil
}

func (node *QueryNode) LoadSegments(ctx context.Context, in *queryPb.LoadSegmentsRequest) (*commonpb.Status, error) {
	dct := &loadSegmentsTask{
		baseTask: baseTask{
			ctx:  ctx,
			done: make(chan error),
		},
		req:  in,
		node: node,
	}

	err := node.scheduler.queue.Enqueue(dct)
	if err != nil {
		status := &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
			Reason:    err.Error(),
		}
		log.Error(err.Error())
		return status, err
	}
	segmentIDs := make([]UniqueID, 0)
	for _, info := range in.Infos {
		segmentIDs = append(segmentIDs, info.SegmentID)
	}
	log.Debug("loadSegmentsTask Enqueue done", zap.Int64s("segmentIDs", segmentIDs))

	func() {
		err = dct.WaitToFinish()
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Debug("loadSegmentsTask WaitToFinish done", zap.Int64s("segmentIDs", segmentIDs))
	}()

	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	return status, nil
}

func (node *QueryNode) ReleaseCollection(ctx context.Context, in *queryPb.ReleaseCollectionRequest) (*commonpb.Status, error) {
	dct := &releaseCollectionTask{
		baseTask: baseTask{
			ctx:  ctx,
			done: make(chan error),
		},
		req:  in,
		node: node,
	}

	err := node.scheduler.queue.Enqueue(dct)
	if err != nil {
		status := &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
			Reason:    err.Error(),
		}
		log.Error(err.Error())
		return status, err
	}
	log.Debug("releaseCollectionTask Enqueue done", zap.Any("collectionID", in.CollectionID))

	func() {
		err = dct.WaitToFinish()
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Debug("releaseCollectionTask WaitToFinish done", zap.Any("collectionID", in.CollectionID))
	}()

	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	return status, nil
}

func (node *QueryNode) ReleasePartitions(ctx context.Context, in *queryPb.ReleasePartitionsRequest) (*commonpb.Status, error) {
	dct := &releasePartitionsTask{
		baseTask: baseTask{
			ctx:  ctx,
			done: make(chan error),
		},
		req:  in,
		node: node,
	}

	err := node.scheduler.queue.Enqueue(dct)
	if err != nil {
		status := &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
			Reason:    err.Error(),
		}
		log.Error(err.Error())
		return status, err
	}
	log.Debug("releasePartitionsTask Enqueue done", zap.Any("collectionID", in.CollectionID))

	func() {
		err = dct.WaitToFinish()
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Debug("releasePartitionsTask WaitToFinish done", zap.Any("collectionID", in.CollectionID))
	}()

	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	return status, nil
}

// ReleaseSegments deprecated
func (node *QueryNode) ReleaseSegments(ctx context.Context, in *queryPb.ReleaseSegmentsRequest) (*commonpb.Status, error) {
	status := &commonpb.Status{
		ErrorCode: commonpb.ErrorCode_Success,
	}
	for _, id := range in.SegmentIDs {
		err := node.historical.replica.removeSegment(id)
		if err != nil {
			// not return, try to release all segments
			status.ErrorCode = commonpb.ErrorCode_UnexpectedError
			status.Reason = err.Error()
		}
		err = node.streaming.replica.removeSegment(id)
		if err != nil {
			// not return, try to release all segments
			status.ErrorCode = commonpb.ErrorCode_UnexpectedError
			status.Reason = err.Error()
		}
	}
	return status, nil
}

func (node *QueryNode) GetSegmentInfo(ctx context.Context, in *queryPb.GetSegmentInfoRequest) (*queryPb.GetSegmentInfoResponse, error) {
	infos := make([]*queryPb.SegmentInfo, 0)
	getSegmentInfo := func(segment *Segment) *queryPb.SegmentInfo {
		var indexName string
		var indexID int64
		// TODO:: segment has multi vec column
		if len(segment.indexInfos) > 0 {
			for fieldID := range segment.indexInfos {
				indexName = segment.getIndexName(fieldID)
				indexID = segment.getIndexID(fieldID)
				break
			}
		}
		info := &queryPb.SegmentInfo{
			SegmentID:    segment.ID(),
			CollectionID: segment.collectionID,
			PartitionID:  segment.partitionID,
			MemSize:      segment.getMemSize(),
			NumRows:      segment.getRowCount(),
			IndexName:    indexName,
			IndexID:      indexID,
		}
		return info
	}
	// get info from historical
	for _, id := range in.SegmentIDs {
		log.Debug("QueryNode::Impl::GetSegmentInfo for historical", zap.Any("SegmentID", id))
		segment, err := node.historical.replica.getSegmentByID(id)
		if err != nil {
			log.Debug("QueryNode::Impl::GetSegmentInfo, for historical segmentID not exist", zap.Any("SegmentID", id))
			continue
		}
		info := getSegmentInfo(segment)
		log.Debug("QueryNode::Impl::GetSegmentInfo for historical", zap.Any("SegmentID", id), zap.Any("info", info))

		infos = append(infos, info)
	}
	// get info from streaming
	for _, id := range in.SegmentIDs {
		log.Debug("QueryNode::Impl::GetSegmentInfo for streaming", zap.Any("SegmentID", id))
		segment, err := node.streaming.replica.getSegmentByID(id)
		if err != nil {
			log.Debug("QueryNode::Impl::GetSegmentInfo, for streaming segmentID not exist", zap.Any("SegmentID", id))
			continue
		}
		info := getSegmentInfo(segment)
		log.Debug("QueryNode::Impl::GetSegmentInfo for streaming", zap.Any("SegmentID", id), zap.Any("info", info))
		infos = append(infos, info)
	}
	return &queryPb.GetSegmentInfoResponse{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
		},
		Infos: infos,
	}, nil
}
