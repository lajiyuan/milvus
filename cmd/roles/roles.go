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

package roles

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/milvus-io/milvus/cmd/components"
	"github.com/milvus-io/milvus/internal/datanode"
	"github.com/milvus-io/milvus/internal/dataservice"
	"github.com/milvus-io/milvus/internal/indexnode"
	"github.com/milvus-io/milvus/internal/indexservice"
	"github.com/milvus-io/milvus/internal/log"
	"github.com/milvus-io/milvus/internal/logutil"
	"github.com/milvus-io/milvus/internal/metrics"
	"github.com/milvus-io/milvus/internal/msgstream"
	"github.com/milvus-io/milvus/internal/proxynode"
	"github.com/milvus-io/milvus/internal/querynode"
	"github.com/milvus-io/milvus/internal/queryservice"
	"github.com/milvus-io/milvus/internal/rootcoord"
	"github.com/milvus-io/milvus/internal/util/paramtable"
	"github.com/milvus-io/milvus/internal/util/trace"
)

func newMsgFactory(localMsg bool) msgstream.Factory {
	if localMsg {
		return msgstream.NewRmsFactory()
	}
	return msgstream.NewPmsFactory()
}

type MilvusRoles struct {
	EnableRootCoord      bool `env:"ENABLE_ROOT_COORD"`
	EnableProxy          bool `env:"ENABLE_PROXY"`
	EnableQueryCoord     bool `env:"ENABLE_QUERY_COORD"`
	EnableQueryNode      bool `env:"ENABLE_QUERY_NODE"`
	EnableDataCoord      bool `env:"ENABLE_DATA_COORD"`
	EnableDataNode       bool `env:"ENABLE_DATA_NODE"`
	EnableIndexCoord     bool `env:"ENABLE_INDEX_COORD"`
	EnableIndexNode      bool `env:"ENABLE_INDEX_NODE"`
	EnableMsgStreamCoord bool `env:"ENABLE_MSGSTREAM_COORD"`
}

func (mr *MilvusRoles) EnvValue(env string) bool {
	env = strings.ToLower(env)
	env = strings.Trim(env, " ")
	return env == "1" || env == "true"
}

func (mr *MilvusRoles) setLogConfigFilename(filename string) *log.Config {
	paramtable.Params.Init()
	cfg := paramtable.Params.LogConfig
	if len(cfg.File.RootPath) != 0 {
		cfg.File.Filename = path.Join(cfg.File.RootPath, filename)
	} else {
		cfg.File.Filename = ""
	}
	return cfg
}

func (mr *MilvusRoles) runRootCoord(ctx context.Context, localMsg bool) *components.RootCoord {
	var rc *components.RootCoord
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		rootcoord.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&rootcoord.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		rc, err = components.NewRootCoord(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = rc.Run()
	}()
	wg.Wait()

	metrics.RegisterRootCoord()
	return rc
}

func (mr *MilvusRoles) runProxy(ctx context.Context, localMsg bool, alias string) *components.Proxy {
	var pn *components.Proxy
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		proxynode.Params.InitAlias(alias)
		proxynode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&proxynode.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		pn, err = components.NewProxy(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = pn.Run()
	}()
	wg.Wait()

	metrics.RegisterProxyNode()
	return pn
}

func (mr *MilvusRoles) runQueryCoord(ctx context.Context, localMsg bool) *components.QueryCoord {
	var qs *components.QueryCoord
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		queryservice.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&queryservice.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		qs, err = components.NewQueryCoord(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = qs.Run()
	}()
	wg.Wait()

	metrics.RegisterQueryCoord()
	return qs
}

func (mr *MilvusRoles) runQueryNode(ctx context.Context, localMsg bool, alias string) *components.QueryNode {
	var qn *components.QueryNode
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		querynode.Params.InitAlias(alias)
		querynode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&querynode.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		qn, err = components.NewQueryNode(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = qn.Run()
	}()
	wg.Wait()

	metrics.RegisterQueryNode()
	return qn
}

func (mr *MilvusRoles) runDataCoord(ctx context.Context, localMsg bool) *components.DataCoord {
	var ds *components.DataCoord
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		dataservice.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&dataservice.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		ds, err = components.NewDataCoord(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = ds.Run()
	}()
	wg.Wait()

	metrics.RegisterDataCoord()
	return ds
}

func (mr *MilvusRoles) runDataNode(ctx context.Context, localMsg bool, alias string) *components.DataNode {
	var dn *components.DataNode
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		datanode.Params.InitAlias(alias)
		datanode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&datanode.Params.Log)
			defer log.Sync()
		}

		factory := newMsgFactory(localMsg)
		var err error
		dn, err = components.NewDataNode(ctx, factory)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = dn.Run()
	}()
	wg.Wait()

	metrics.RegisterDataNode()
	return dn
}

func (mr *MilvusRoles) runIndexCoord(ctx context.Context, localMsg bool) *components.IndexCoord {
	var is *components.IndexCoord
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		indexservice.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&indexservice.Params.Log)
			defer log.Sync()
		}

		var err error
		is, err = components.NewIndexCoord(ctx)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = is.Run()
	}()
	wg.Wait()

	metrics.RegisterIndexCoord()
	return is
}

func (mr *MilvusRoles) runIndexNode(ctx context.Context, localMsg bool, alias string) *components.IndexNode {
	var in *components.IndexNode
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		indexnode.Params.InitAlias(alias)
		indexnode.Params.Init()

		if !localMsg {
			logutil.SetupLogger(&indexnode.Params.Log)
			defer log.Sync()
		}

		var err error
		in, err = components.NewIndexNode(ctx)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = in.Run()
	}()
	wg.Wait()

	metrics.RegisterIndexNode()
	return in
}

func (mr *MilvusRoles) runMsgStreamCoord(ctx context.Context) *components.MsgStream {
	var mss *components.MsgStream
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		var err error
		mss, err = components.NewMsgStreamCoord(ctx)
		if err != nil {
			panic(err)
		}
		wg.Done()
		_ = mss.Run()
	}()
	wg.Wait()

	metrics.RegisterMsgStreamCoord()
	return mss
}

func (mr *MilvusRoles) Run(localMsg bool, alias string) {
	if os.Getenv("DEPLOY_MODE") == "STANDALONE" {
		closer := trace.InitTracing("standalone")
		if closer != nil {
			defer closer.Close()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// only standalone enable localMsg
	if localMsg {
		os.Setenv("DEPLOY_MODE", "STANDALONE")
		cfg := mr.setLogConfigFilename("standalone.log")
		logutil.SetupLogger(cfg)
		defer log.Sync()
	}

	var rc *components.RootCoord
	if mr.EnableRootCoord {
		rc = mr.runRootCoord(ctx, localMsg)
		if rc != nil {
			defer rc.Stop()
		}
	}

	var pn *components.Proxy
	if mr.EnableProxy {
		pn = mr.runProxy(ctx, localMsg, alias)
		if pn != nil {
			defer pn.Stop()
		}
	}

	var qs *components.QueryCoord
	if mr.EnableQueryCoord {
		qs = mr.runQueryCoord(ctx, localMsg)
		if qs != nil {
			defer qs.Stop()
		}
	}

	var qn *components.QueryNode
	if mr.EnableQueryNode {
		qn = mr.runQueryNode(ctx, localMsg, alias)
		if qn != nil {
			defer qn.Stop()
		}
	}

	var ds *components.DataCoord
	if mr.EnableDataCoord {
		ds = mr.runDataCoord(ctx, localMsg)
		if ds != nil {
			defer ds.Stop()
		}
	}

	var dn *components.DataNode
	if mr.EnableDataNode {
		dn = mr.runDataNode(ctx, localMsg, alias)
		if dn != nil {
			defer dn.Stop()
		}
	}

	var is *components.IndexCoord
	if mr.EnableIndexCoord {
		is = mr.runIndexCoord(ctx, localMsg)
		if is != nil {
			defer is.Stop()
		}
	}

	var in *components.IndexNode
	if mr.EnableIndexNode {
		in = mr.runIndexNode(ctx, localMsg, alias)
		if in != nil {
			defer in.Stop()
		}
	}

	var mss *components.MsgStream
	if mr.EnableMsgStreamCoord {
		mss = mr.runMsgStreamCoord(ctx)
		if mss != nil {
			defer mss.Stop()
		}
	}

	metrics.ServeHTTP()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-sc
	fmt.Printf("Get %s signal to exit\n", sig.String())

	// some deferred Stop has race with context cancel
	cancel()
}
