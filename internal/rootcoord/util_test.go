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

package rootcoord

import (
	"testing"

	"github.com/milvus-io/milvus/internal/proto/commonpb"
	"github.com/milvus-io/milvus/internal/proto/etcdpb"
	"github.com/milvus-io/milvus/internal/proto/schemapb"
	"github.com/stretchr/testify/assert"
)

func TestEqualKeyPairArray(t *testing.T) {
	p1 := []*commonpb.KeyValuePair{
		{
			Key:   "k1",
			Value: "v1",
		},
	}

	p2 := []*commonpb.KeyValuePair{}
	assert.False(t, EqualKeyPairArray(p1, p2))

	p2 = append(p2, &commonpb.KeyValuePair{
		Key:   "k2",
		Value: "v2",
	})
	assert.False(t, EqualKeyPairArray(p1, p2))
	p2 = []*commonpb.KeyValuePair{
		{
			Key:   "k1",
			Value: "v2",
		},
	}
	assert.False(t, EqualKeyPairArray(p1, p2))

	p2 = []*commonpb.KeyValuePair{
		{
			Key:   "k1",
			Value: "v1",
		},
	}
	assert.True(t, EqualKeyPairArray(p1, p2))
}

func Test_GetFieldSchemaByID(t *testing.T) {
	coll := &etcdpb.CollectionInfo{
		Schema: &schemapb.CollectionSchema{
			Fields: []*schemapb.FieldSchema{
				{
					FieldID: 1,
				},
			},
		},
	}
	_, err := GetFieldSchemaByID(coll, 1)
	assert.Nil(t, err)
	_, err = GetFieldSchemaByID(coll, 2)
	assert.NotNil(t, err)
}

func Test_GetFieldSchemaByIndexID(t *testing.T) {
	coll := &etcdpb.CollectionInfo{
		Schema: &schemapb.CollectionSchema{
			Fields: []*schemapb.FieldSchema{
				{
					FieldID: 1,
				},
			},
		},
		FieldIndexes: []*etcdpb.FieldIndexInfo{
			{
				FiledID: 1,
				IndexID: 2,
			},
		},
	}
	_, err := GetFieldSchemaByIndexID(coll, 2)
	assert.Nil(t, err)
	_, err = GetFieldSchemaByIndexID(coll, 3)
	assert.NotNil(t, err)
}
