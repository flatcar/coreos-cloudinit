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

package initialize

import (
	"os"
	"testing"

	"github.com/flatcar/coreos-cloudinit/config"
	"github.com/flatcar/coreos-cloudinit/datasource"

	"github.com/stretchr/testify/require"
)

func TestParseHeaderCRLF(t *testing.T) {
	configs := []string{
		"#cloud-config\nfoo: bar",
		"#cloud-config\r\nfoo: bar",
	}

	for i, config := range configs {
		_, err := ParseUserData(config)
		if err != nil {
			t.Errorf("Failed parsing config %d: %v", i, err)
		}
	}

	scripts := []string{
		"#!bin/bash\necho foo",
		"#!bin/bash\r\necho foo",
	}

	for i, script := range scripts {
		_, err := ParseUserData(script)
		if err != nil {
			t.Errorf("Failed parsing script %d: %v", i, err)
		}
	}
}

func TestParseConfigCRLF(t *testing.T) {
	contents := "#cloud-config \r\nhostname: foo\r\nssh_authorized_keys:\r\n  - foobar\r\n"
	ud, err := ParseUserData(contents)
	if err != nil {
		t.Fatalf("Failed parsing config: %v", err)
	}

	cfg := ud.(*config.CloudConfig)

	if cfg.Hostname != "foo" {
		t.Error("Failed parsing hostname from config")
	}

	if len(cfg.SSHAuthorizedKeys) != 1 {
		t.Error("Parsed incorrect number of SSH keys")
	}
}

func TestParseConfigEmpty(t *testing.T) {
	i, e := ParseUserData(``)
	if i != nil {
		t.Error("ParseUserData of empty string returned non-nil unexpectedly")
	} else if e != nil {
		t.Error("ParseUserData of empty string returned error unexpectedly")
	}
}

func TestParseMultipartMime(t *testing.T) {
	data, err := os.ReadFile("testdata/multipart_mime_userdata.txt")
	require.NoError(t, err)

	udata, err := NewUserData(string(data), getTestEnv())
	require.NoError(t, err)
	require.Equal(t, 8, len(udata.Parts))

	hostname := udata.FindHostname()
	require.Equal(t, "example", hostname)

	keys := udata.FindSSHKeys([]string{"bogus key"})
	require.Equal(t, 2, len(keys))
	require.Equal(t, "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEftQIHTRvUmyDCN7VGve4srz03Jmq6rPnqq+XMHMQUIL9c/b0l7B5tWfQvQecKyLte94HOPzAyMJlktWTVGQnY=", keys[0])
	require.Equal(t, "bogus key", keys[1])

	// The secong cloud-config defines a bunch of write_files, with varying encoding types and formats.
	// All of which will contain the string "42" in the content.
	require.NotNil(t, udata.Parts[2].cloudConfig)
	for _, val := range udata.Parts[2].cloudConfig.WriteFiles {
		require.Equal(t, "42", val.Content)
	}

	require.Equal(t, udata.Parts[3].userDataType, ScriptType)
	require.NotNil(t, udata.Parts[3].script)

	// This cloud config contains only a comment. We should get back a valid
	// CloudConfig with no fields set.
	require.Equal(t, udata.Parts[4].userDataType, CloudConfigType)
	require.NotNil(t, udata.Parts[4].cloudConfig)

	// The next two parts are an UnknownType and an Ignition config.
	// We don't process these, but we should still be able to find them.
	require.Equal(t, udata.Parts[5].userDataType, UnknownType)
	require.Equal(t, udata.Parts[6].userDataType, IgnitionType)

	// The last part is a cloud-config, but the content-type is set to an unknown type.
	// We try to parse types we don't directly handle, in the hopes they actually contain
	// valid user-data. This adds some tolerance for misconfigured content-types.
	require.Equal(t, udata.Parts[7].userDataType, CloudConfigType)
	require.Equal(t, udata.Parts[7].cloudConfig.Hostname, "undercover")
}

func getTestEnv() *Environment {
	metadata := datasource.Metadata{}
	return NewEnvironment("./", "./", "./", "", metadata)
}

func TestNewUserDataParsesIgnition(t *testing.T) {
	data, err := os.ReadFile("testdata/ignition_userdata.txt")
	require.NoError(t, err)

	udata, err := NewUserData(string(data), getTestEnv())
	require.NoError(t, err)
	require.Equal(t, 1, len(udata.Parts))
	require.Equal(t, udata.Parts[0].userDataType, IgnitionType)
}

func TestNewUserDataParsesCloudConfig(t *testing.T) {
	data, err := os.ReadFile("testdata/cloudconfig_userdata.txt")
	require.NoError(t, err)

	udata, err := NewUserData(string(data), getTestEnv())
	require.NoError(t, err)
	require.Equal(t, 1, len(udata.Parts))
	require.Equal(t, udata.Parts[0].userDataType, CloudConfigType)
	require.NotNil(t, udata.Parts[0].cloudConfig)
}

func TestNewUserDataParsesScript(t *testing.T) {
	data, err := os.ReadFile("testdata/script_userdata.txt")
	require.NoError(t, err)

	udata, err := NewUserData(string(data), getTestEnv())
	require.NoError(t, err)
	require.Equal(t, 1, len(udata.Parts))
	require.Equal(t, udata.Parts[0].userDataType, ScriptType)
	require.NotNil(t, udata.Parts[0].script)
}

func TestNewUserDataParsesUnknown(t *testing.T) {
	data, err := os.ReadFile("testdata/unknown_userdata.txt")
	require.NoError(t, err)

	udata, err := NewUserData(string(data), getTestEnv())
	require.NoError(t, err)
	require.Equal(t, 1, len(udata.Parts))
	require.Equal(t, udata.Parts[0].userDataType, UnknownType)
}
