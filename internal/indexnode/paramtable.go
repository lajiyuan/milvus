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

package indexnode

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/milvus-io/milvus/internal/log"
	"github.com/milvus-io/milvus/internal/proto/internalpb"
	"github.com/milvus-io/milvus/internal/util/paramtable"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	StartParamsKey = "START_PARAMS"
)

type ParamTable struct {
	paramtable.BaseTable

	IP      string
	Address string
	Port    int

	NodeID int64
	Alias  string

	MasterAddress string

	EtcdEndpoints []string
	MetaRootPath  string

	MinIOAddress         string
	MinIOAccessKeyID     string
	MinIOSecretAccessKey string
	MinIOUseSSL          bool
	MinioBucketName      string

	Log log.Config
}

var Params ParamTable
var once sync.Once

func (pt *ParamTable) InitAlias(alias string) {
	pt.Alias = alias
}

func (pt *ParamTable) Init() {
	once.Do(func() {
		pt.BaseTable.Init()
		pt.initLogCfg()
		pt.initParams()
	})
}

func (pt *ParamTable) initParams() {
	pt.initMinIOAddress()
	pt.initMinIOAccessKeyID()
	pt.initMinIOSecretAccessKey()
	pt.initMinIOUseSSL()
	pt.initMinioBucketName()
	pt.initEtcdEndpoints()
	pt.initMetaRootPath()
}

func (pt *ParamTable) LoadConfigFromInitParams(initParams *internalpb.InitParams) error {
	pt.NodeID = initParams.NodeID

	config := viper.New()
	config.SetConfigType("yaml")
	for _, pair := range initParams.StartParams {
		if pair.Key == StartParamsKey {
			err := config.ReadConfig(bytes.NewBuffer([]byte(pair.Value)))
			if err != nil {
				return err
			}
			break
		}
	}

	for _, key := range config.AllKeys() {
		val := config.Get(key)
		str, err := cast.ToStringE(val)
		if err != nil {
			switch val := val.(type) {
			case []interface{}:
				str = str[:0]
				for _, v := range val {
					ss, err := cast.ToStringE(v)
					if err != nil {
						log.Debug("indexnode", zap.String("error", err.Error()))
					}
					if len(str) == 0 {
						str = ss
					} else {
						str = str + "," + ss
					}
				}

			default:
				log.Debug("indexnode", zap.String("undefine config type, key=", key))
			}
		}
		err = pt.Save(key, str)
		if err != nil {
			panic(err)
		}

	}

	pt.initParams()
	return nil
}

func (pt *ParamTable) initMinIOAddress() {
	ret, err := pt.Load("_MinioAddress")
	if err != nil {
		panic(err)
	}
	pt.MinIOAddress = ret
}

func (pt *ParamTable) initMinIOAccessKeyID() {
	ret, err := pt.Load("minio.accessKeyID")
	if err != nil {
		panic(err)
	}
	pt.MinIOAccessKeyID = ret
}

func (pt *ParamTable) initMinIOSecretAccessKey() {
	ret, err := pt.Load("minio.secretAccessKey")
	if err != nil {
		panic(err)
	}
	pt.MinIOSecretAccessKey = ret
}

func (pt *ParamTable) initMinIOUseSSL() {
	ret, err := pt.Load("minio.useSSL")
	if err != nil {
		panic(err)
	}
	pt.MinIOUseSSL, err = strconv.ParseBool(ret)
	if err != nil {
		panic(err)
	}
}

func (pt *ParamTable) initEtcdEndpoints() {
	endpoints, err := pt.Load("_EtcdEndpoints")
	if err != nil {
		panic(err)
	}
	pt.EtcdEndpoints = strings.Split(endpoints, ",")
}

func (pt *ParamTable) initMetaRootPath() {
	rootPath, err := pt.Load("etcd.rootPath")
	if err != nil {
		panic(err)
	}
	subPath, err := pt.Load("etcd.metaSubPath")
	if err != nil {
		panic(err)
	}
	pt.MetaRootPath = path.Join(rootPath, subPath)
}

func (pt *ParamTable) initMinioBucketName() {
	bucketName, err := pt.Load("minio.bucketName")
	if err != nil {
		panic(err)
	}
	pt.MinioBucketName = bucketName
}

func (pt *ParamTable) initLogCfg() {
	pt.Log = log.Config{}
	format, err := pt.Load("log.format")
	if err != nil {
		panic(err)
	}
	pt.Log.Format = format
	level, err := pt.Load("log.level")
	if err != nil {
		panic(err)
	}
	pt.Log.Level = level
	pt.Log.File.MaxSize = pt.ParseInt("log.file.maxSize")
	pt.Log.File.MaxBackups = pt.ParseInt("log.file.maxBackups")
	pt.Log.File.MaxDays = pt.ParseInt("log.file.maxAge")
	rootPath, err := pt.Load("log.file.rootPath")
	if err != nil {
		panic(err)
	}
	if len(rootPath) != 0 {
		pt.Log.File.Filename = path.Join(rootPath, fmt.Sprintf("indexnode-%s.log", pt.Alias))
	} else {
		pt.Log.File.Filename = ""
	}
}
