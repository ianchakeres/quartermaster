// Copyright 2016 The quartermaster Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"encoding/json"
	"time"

	"github.com/coreos-inc/quartermaster/pkg/spec"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const resyncPeriod = 5 * time.Minute

func newQuartermasterRESTClient(c rest.Config) (*rest.RESTClient, error) {
	c.APIPath = "/apis"
	c.GroupVersion = &schema.GroupVersion{
		Group:   "storage.coreos.com",
		Version: "v1alpha1",
	}
	// TODO(fabxc): is this even used with our custom list/watch functions?
	c.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	return rest.RESTClientFor(&c)
}

type storageNodeDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *storageNodeDecoder) Close() {
	d.close()
}

func (d *storageNodeDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object spec.StorageNode
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}

// NewStorageNodeListWatch returns a new ListWatch on the StorageNode resource.
func NewStorageNodeListWatch(client *rest.RESTClient) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			req := client.Get().
				Namespace(api.NamespaceAll).
				Resource("storagenodes").
				// VersionedParams(&options, api.ParameterCodec)
				FieldsSelectorParam(nil)

			b, err := req.DoRaw()
			if err != nil {
				return nil, err
			}
			var p spec.StorageNodeList
			return &p, json.Unmarshal(b, &p)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			r, err := client.Get().
				Prefix("watch").
				Namespace(api.NamespaceAll).
				Resource("storagenodes").
				// VersionedParams(&options, api.ParameterCodec).
				FieldsSelectorParam(nil).
				Stream()
			if err != nil {
				return nil, err
			}
			return watch.NewStreamWatcher(&storageNodeDecoder{
				dec:   json.NewDecoder(r),
				close: r.Close,
			}), nil
		},
	}
}
