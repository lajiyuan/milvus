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

package components

import (
	"context"

	grpcqueryservice "github.com/milvus-io/milvus/internal/distributed/queryservice"
	"github.com/milvus-io/milvus/internal/msgstream"
)

type QueryCoord struct {
	ctx context.Context
	svr *grpcqueryservice.Server
}

// NewQueryCoord creates a new QueryCoord
func NewQueryCoord(ctx context.Context, factory msgstream.Factory) (*QueryCoord, error) {
	svr, err := grpcqueryservice.NewServer(ctx, factory)
	if err != nil {
		panic(err)
	}

	return &QueryCoord{
		ctx: ctx,
		svr: svr,
	}, nil
}

// Run starts service
func (qs *QueryCoord) Run() error {
	if err := qs.svr.Run(); err != nil {
		panic(err)
	}
	return nil
}

// Stop terminates service
func (qs *QueryCoord) Stop() error {
	if err := qs.svr.Stop(); err != nil {
		return err
	}
	return nil
}
