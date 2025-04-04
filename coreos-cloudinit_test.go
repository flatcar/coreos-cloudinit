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

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/flatcar/coreos-cloudinit/datasource"
	"github.com/flatcar/coreos-cloudinit/initialize"
	"net"
	"reflect"
	"testing"
)

func mustDecode(in string) []byte {
	out, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return out
}

func TestDecompressIfGzip(t *testing.T) {
	tests := []struct {
		in []byte

		out []byte
		err error
	}{
		{
			in: nil,

			out: nil,
			err: nil,
		},
		{
			in: []byte{},

			out: []byte{},
			err: nil,
		},
		{
			in: mustDecode("H4sIAJWV/VUAA1NOzskvTdFNzs9Ly0wHABt6mQENAAAA"),

			out: []byte("#cloud-config"),
			err: nil,
		},
		{
			in: []byte("#cloud-config"),

			out: []byte("#cloud-config"),
			err: nil,
		},
		{
			in: mustDecode("H4sCORRUPT=="),

			out: nil,
			err: errors.New("any error"),
		},
	}
	for i, tt := range tests {
		out, err := decompressIfGzip(tt.in)
		if !bytes.Equal(out, tt.out) || (tt.err != nil && err == nil) {
			t.Errorf("bad gzip (%d): want (%s, %#v), got (%s, %#v)", i, string(tt.out), tt.err, string(out), err)
		}
	}

}

func TestDetermineHostname(t *testing.T) {
	for _, tt := range []struct {
		metaData datasource.Metadata
		uData    *initialize.UserData
		expect   string
	}{
		{
			metaData: datasource.Metadata{
				PublicIPv4:    net.ParseIP("1.2.3.4"),
				PublicIPv6:    net.ParseIP("5.6.7.8"),
				PrivateIPv4:   net.ParseIP("1.2.3.4"),
				PrivateIPv6:   net.ParseIP("5.6.7.8"),
				Hostname:      "regular-name",
				SSHPublicKeys: map[string]string{"my": "key"},
				NetworkConfig: net.Interface{
					Index:        0,
					MTU:          0,
					Name:         "some-interface",
					HardwareAddr: nil,
					Flags:        0,
				},
			},
			uData:  nil,
			expect: "regular-name",
		},
		{
			metaData: datasource.Metadata{
				PublicIPv4:    net.ParseIP("1.2.3.4"),
				PublicIPv6:    net.ParseIP("5.6.7.8"),
				PrivateIPv4:   net.ParseIP("1.2.3.4"),
				PrivateIPv6:   net.ParseIP("5.6.7.8"),
				Hostname:      "regular-name.domain",
				SSHPublicKeys: map[string]string{"my": "key"},
				NetworkConfig: net.Interface{
					Index:        0,
					MTU:          0,
					Name:         "some-interface",
					HardwareAddr: nil,
					Flags:        0,
				},
			},
			uData:  nil,
			expect: "regular-name",
		},
		{
			metaData: datasource.Metadata{
				PublicIPv4:    net.ParseIP("1.2.3.4"),
				PublicIPv6:    net.ParseIP("5.6.7.8"),
				PrivateIPv4:   net.ParseIP("1.2.3.4"),
				PrivateIPv6:   net.ParseIP("5.6.7.8"),
				Hostname:      "this-hostname-is-larger-than-sixty-three-characters-long-and.will.be.truncated.locale",
				SSHPublicKeys: map[string]string{"my": "key"},
				NetworkConfig: net.Interface{
					Index:        0,
					MTU:          0,
					Name:         "some-interface",
					HardwareAddr: nil,
					Flags:        0,
				},
			},
			uData:  nil,
			expect: "this-hostname-is-larger-than-sixty-three-characters-long-and",
		},
		{
			metaData: datasource.Metadata{
				PublicIPv4:    net.ParseIP("1.2.3.4"),
				PublicIPv6:    net.ParseIP("5.6.7.8"),
				PrivateIPv4:   net.ParseIP("1.2.3.4"),
				PrivateIPv6:   net.ParseIP("5.6.7.8"),
				Hostname:      "this-hostname-is-larger-than-sixty-three-characters-long-and-will-be-truncated.local",
				SSHPublicKeys: map[string]string{"my": "key"},
				NetworkConfig: net.Interface{
					Index:        0,
					MTU:          0,
					Name:         "some-interface",
					HardwareAddr: nil,
					Flags:        0,
				},
			},
			uData:  nil,
			expect: "this-hostname-is-larger-than-sixty-three-characters-long-and-wi",
		},
	} {
		hostname := determineHostname(tt.metaData, tt.uData)
		if !reflect.DeepEqual(tt.expect, hostname) {
			t.Fatalf("Bad hostname, want %s, got %s", tt.expect, hostname)
		}

	}
}
