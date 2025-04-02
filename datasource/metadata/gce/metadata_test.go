// Copyright 2015 CoreOS, Inc.
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

package gce

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/flatcar/coreos-cloudinit/datasource"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata/test"
	"github.com/flatcar/coreos-cloudinit/pkg"
)

func TestType(t *testing.T) {
	want := "gce-metadata-service"
	if kind := (metadataService{}).Type(); kind != want {
		t.Fatalf("bad type: want %q, got %q", want, kind)
	}
}

func TestFetchMetadata(t *testing.T) {
	for _, tt := range []struct {
		root         string
		metadataPath string
		resources    map[string]string
		expect       datasource.Metadata
		clientErr    error
		expectErr    error
	}{
		{
			root:         "/",
			metadataPath: "computeMetadata/v1/instance/",
			resources:    map[string]string{},
		},
		{
			root:         "/",
			metadataPath: "computeMetadata/v1/instance/",
			resources: map[string]string{
				"/computeMetadata/v1/instance/hostname": "host",
			},
			expect: datasource.Metadata{
				Hostname: "host",
			},
		},
		{
			root:         "/",
			metadataPath: "computeMetadata/v1/instance/",
			resources: map[string]string{
				"/computeMetadata/v1/instance/hostname":                                          "host",
				"/computeMetadata/v1/instance/network-interfaces/0/ip":                           "1.2.3.4",
				"/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip": "5.6.7.8",
			},
			expect: datasource.Metadata{
				Hostname:    "host",
				PrivateIPv4: net.ParseIP("1.2.3.4"),
				PublicIPv4:  net.ParseIP("5.6.7.8"),
			},
		},
		{
			// test long host names are truncated on first `.`
			root:         "/",
			metadataPath: "computeMetadata/v1/instance/",
			resources: map[string]string{
				"/computeMetadata/v1/instance/hostname":                                          "this-hostname-is-larger-than-sixty-three-characters-long-and.will.be.truncated.local",
				"/computeMetadata/v1/instance/network-interfaces/0/ip":                           "1.2.3.4",
				"/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip": "5.6.7.8",
			},
			expect: datasource.Metadata{
				Hostname:    "this-hostname-is-larger-than-sixty-three-characters-long-and",
				PrivateIPv4: net.ParseIP("1.2.3.4"),
				PublicIPv4:  net.ParseIP("5.6.7.8"),
			},
		},
		{
			// test long host names are truncated if segment before first `.` is too long
			root:         "/",
			metadataPath: "computeMetadata/v1/instance/",
			resources: map[string]string{
				"/computeMetadata/v1/instance/hostname":                                          "this-hostname-is-larger-than-sixty-three-characters-long-and-will-be-truncated.local",
				"/computeMetadata/v1/instance/network-interfaces/0/ip":                           "1.2.3.4",
				"/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip": "5.6.7.8",
			},
			expect: datasource.Metadata{
				Hostname:    "this-hostname-is-larger-than-sixty-three-characters-long-and-wi",
				PrivateIPv4: net.ParseIP("1.2.3.4"),
				PublicIPv4:  net.ParseIP("5.6.7.8"),
			},
		},
		{
			clientErr: pkg.ErrTimeout{Err: fmt.Errorf("test error")},
			expectErr: pkg.ErrTimeout{Err: fmt.Errorf("test error")},
		},
	} {
		service := &metadataService{metadata.MetadataService{
			Root:         tt.root,
			Client:       &test.HttpClient{Resources: tt.resources, Err: tt.clientErr},
			MetadataPath: tt.metadataPath,
		}}
		metadata, err := service.FetchMetadata()
		if Error(err) != Error(tt.expectErr) {
			t.Fatalf("bad error (%q): want %q, got %q", tt.resources, tt.expectErr, err)
		}
		if !reflect.DeepEqual(tt.expect, metadata) {
			t.Fatalf("bad fetch (%q): want %#v, got %#v", tt.resources, tt.expect, metadata)
		}
	}
}

func Error(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
