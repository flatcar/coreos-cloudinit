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

package validate

import (
	"reflect"
	"testing"
)

func TestCheckDiscoveryUrl(t *testing.T) {
	tests := []struct {
		config string

		entries []Entry
	}{
		{},
		{
			config: "coreos:\n  etcd:\n    discovery: https://discovery.etcd.io/00000000000000000000000000000000",
		},
		{
			config: "coreos:\n  etcd:\n    discovery: http://custom.domain/mytoken",
		},
		{
			config:  "coreos:\n  etcd:\n    discovery: disco",
			entries: []Entry{{entryWarning, "discovery URL is not valid", 3}},
		},
	}

	for i, tt := range tests {
		r := Report{}
		n, err := parseCloudConfig([]byte(tt.config), &r)
		if err != nil {
			panic(err)
		}
		checkDiscoveryUrl(n, &r)

		if e := r.Entries(); !reflect.DeepEqual(tt.entries, e) {
			t.Errorf("bad report (%d, %q): want %#v, got %#v", i, tt.config, tt.entries, e)
		}
	}
}

func TestCheckEncoding(t *testing.T) {
	tests := []struct {
		config string

		entries []Entry
	}{
		{},
		{
			config: "write_files:\n  - encoding: base64\n    content: aGVsbG8K",
		},
		{
			config: "write_files:\n  - content: !!binary aGVsbG8K",
		},
		{
			config:  "write_files:\n  - encoding: base64\n    content: !!binary aGVsbG8K",
			entries: []Entry{{entryError, `content cannot be decoded as "base64"`, 3}},
		},
		{
			config: "write_files:\n  - encoding: base64\n    content: !!binary YUdWc2JHOEsK",
		},
		{
			config: "write_files:\n  - encoding: gzip\n    content: !!binary H4sIAOC3tVQAA8tIzcnJ5wIAIDA6NgYAAAA=",
		},
		{
			config: "write_files:\n  - encoding: gzip+base64\n    content: H4sIAOC3tVQAA8tIzcnJ5wIAIDA6NgYAAAA=",
		},
		{
			config:  "write_files:\n  - encoding: custom\n    content: hello",
			entries: []Entry{{entryError, `content cannot be decoded as "custom"`, 3}},
		},
	}

	for i, tt := range tests {
		r := Report{}
		n, err := parseCloudConfig([]byte(tt.config), &r)
		if err != nil {
			panic(err)
		}
		checkEncoding(n, &r)

		if e := r.Entries(); !reflect.DeepEqual(tt.entries, e) {
			t.Errorf("bad report (%d, %q): want %#v, got %#v", i, tt.config, tt.entries, e)
		}
	}
}

func TestCheckStructure(t *testing.T) {
	tests := []struct {
		config string

		entries []Entry
	}{
		{},

		// Test for unrecognized keys
		{
			config:  "test:",
			entries: []Entry{{entryWarning, "unrecognized key \"test\"", 1}},
		},
		{
			config:  "coreos:\n  flannel:\n    bad:",
			entries: []Entry{{entryWarning, "unrecognized key \"bad\"", 3}},
		},
		{
			config: "coreos:\n  flannel:\n    interface: good",
		},

		// Test for deprecated keys
		{
			config:  "coreos:\n  etcd:\n    proxy: hi",
			entries: []Entry{{entryWarning, "deprecated key \"etcd\" (etcd is no longer shipped in Container Linux)", 2}, {entryWarning, "deprecated key \"proxy\" (etcd2 options no longer work for etcd)", 3}},
		},

		// Test for incorrect types
		// Want boolean
		{
			config: "coreos:\n  units:\n    - enable: true",
		},
		// Want string
		{
			config: "hostname: true",
		},
		{
			config: "hostname: 4",
		},
		{
			config: "hostname: host",
		},
		{
			config: "ssh_authorized_keys:\n  - key",
		},
		{
			config: "users:\n  - name: good",
		},
	}

	for i, tt := range tests {
		r := Report{}
		n, err := parseCloudConfig([]byte(tt.config), &r)
		if err != nil {
			panic(err)
		}
		checkStructure(n, &r)

		if e := r.Entries(); !reflect.DeepEqual(tt.entries, e) {
			t.Errorf("bad report (%d, %q): want %#v, got %#v", i, tt.config, tt.entries, e)
		}
	}
}

func TestCheckValidity(t *testing.T) {
	tests := []struct {
		config string

		entries []Entry
	}{
		// string
		{
			config: "hostname: test",
		},

		// int
		{
			config: "coreos:\n  fleet:\n    verbosity: 2",
		},

		// bool
		{
			config: "coreos:\n  units:\n    - enable: true",
		},

		// slice
		{
			config: "coreos:\n  units:\n    - command: start\n    - name: stop",
		},
		{
			config:  "coreos:\n  units:\n    - command: lol",
			entries: []Entry{{entryError, "invalid value lol", 3}},
		},

		// struct
		{
			config: "coreos:\n  update:\n    reboot_strategy: off",
		},
		{
			config:  "coreos:\n  update:\n    reboot_strategy: always",
			entries: []Entry{{entryError, "invalid value always", 3}},
		},

		// unknown
		{
			config: "unknown: hi",
		},
	}

	for i, tt := range tests {
		r := Report{}
		n, err := parseCloudConfig([]byte(tt.config), &r)
		if err != nil {
			panic(err)
		}
		checkValidity(n, &r)

		if e := r.Entries(); !reflect.DeepEqual(tt.entries, e) {
			t.Errorf("bad report (%d, %q): want %#v, got %#v", i, tt.config, tt.entries, e)
		}
	}
}

func TestCheckWriteFiles(t *testing.T) {
	tests := []struct {
		config string

		entries []Entry
	}{
		{},
		{
			config: "write_files:\n  - path: /valid",
		},
		{
			config: "write_files:\n  - path: /tmp/usr/valid",
		},
		{
			config:  "write_files:\n  - path: /usr/invalid",
			entries: []Entry{{entryError, "file cannot be written to a read-only filesystem", 2}},
		},
		{
			config:  "write-files:\n  - path: /tmp/../usr/invalid",
			entries: []Entry{{entryError, "file cannot be written to a read-only filesystem", 2}},
		},
	}

	for i, tt := range tests {
		r := Report{}
		n, err := parseCloudConfig([]byte(tt.config), &r)
		if err != nil {
			panic(err)
		}
		checkWriteFiles(n, &r)

		if e := r.Entries(); !reflect.DeepEqual(tt.entries, e) {
			t.Errorf("bad report (%d, %q): want %#v, got %#v", i, tt.config, tt.entries, e)
		}
	}
}

func TestCheckWriteFilesUnderCoreos(t *testing.T) {
	tests := []struct {
		config string

		entries []Entry
	}{
		{},
		{
			config: "write_files:\n  - path: /hi",
		},
		{
			config:  "coreos:\n  write_files:\n    - path: /hi",
			entries: []Entry{{entryInfo, "write_files doesn't belong under coreos", 2}},
		},
		{
			config:  "coreos:\n  write-files:\n    - path: /hyphen",
			entries: []Entry{{entryInfo, "write_files doesn't belong under coreos", 2}},
		},
	}

	for i, tt := range tests {
		r := Report{}
		n, err := parseCloudConfig([]byte(tt.config), &r)
		if err != nil {
			panic(err)
		}
		checkWriteFilesUnderCoreos(n, &r)

		if e := r.Entries(); !reflect.DeepEqual(tt.entries, e) {
			t.Errorf("bad report (%d, %q): want %#v, got %#v", i, tt.config, tt.entries, e)
		}
	}
}
