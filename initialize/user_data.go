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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"

	"github.com/flatcar/coreos-cloudinit/config"
	"github.com/flatcar/coreos-cloudinit/system"
)

type UserDataType string

const (
	CloudConfigType UserDataType = "cloud-config"
	ScriptType      UserDataType = "script"
	IgnitionType    UserDataType = "ignition"
	UnknownType     UserDataType = "unknown"
)

var (
	ErrIgnitionConfig = errors.New("not a config (found Ignition)")
)

func ParseUserData(contents string) (interface{}, error) {
	if len(contents) == 0 {
		return nil, nil
	}

	switch {
	case config.IsScript(contents):
		log.Printf("Parsing user-data as script")
		return config.NewScript(contents)
	case config.IsCloudConfig(contents):
		log.Printf("Parsing user-data as cloud-config")
		cc, err := config.NewCloudConfig(contents)
		if err != nil {
			return nil, err
		}

		if err := cc.Decode(); err != nil {
			return nil, err
		}

		return cc, nil
	case config.IsIgnitionConfig(contents):
		return nil, ErrIgnitionConfig
	default:
		return nil, errors.New("Unrecognized user-data format")
	}
}

func NewUserData(payload string, env *Environment) (*UserData, error) {
	if len(payload) == 0 {
		return &UserData{}, nil
	}

	if env == nil {
		return nil, fmt.Errorf("environment is nil")
	}

	parts, err := partsFromUserData(payload, env)
	if err != nil {
		return nil, fmt.Errorf("error parsing user-data: %w", err)
	}
	return &UserData{
		Parts: parts,
		env:   env,
	}, nil
}

func multipartToUserDataParts(payload string, env *Environment) ([]UserDataPart, error) {
	if env == nil {
		return nil, fmt.Errorf("environment is nil")
	}
	reader := strings.NewReader(payload)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		return []UserDataPart{}, fmt.Errorf("error parsing multipart MIME: %w", err)
	}

	contentType := m.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return []UserDataPart{}, fmt.Errorf("error parsing header: %w", err)
	}
	if mediaType != "multipart/mixed" {
		// We expect a multipart/mixed message.
		return []UserDataPart{}, fmt.Errorf("expected multipart/mixed, got %s", mediaType)
	}
	// get the boundary from the Content-Type header
	boundary, ok := params["boundary"]
	if !ok {
		return []UserDataPart{}, errors.New("no boundary found in Content-Type header")
	}

	multipartReader := multipart.NewReader(m.Body, boundary)
	udParts := []UserDataPart{}
	for {
		part, err := multipartReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []UserDataPart{}, fmt.Errorf("error reading part: %w", err)
		}

		partContentType := part.Header.Get("Content-Type")
		partMediaType, _, err := mime.ParseMediaType(partContentType)
		if err != nil {
			return []UserDataPart{}, fmt.Errorf("error parsing header: %w", err)
		}
		partTransferEncoding := part.Header.Get("Content-Transfer-Encoding")

		body, err := io.ReadAll(part)
		if err != nil {
			return []UserDataPart{}, fmt.Errorf("error reading part: %w", err)
		}

		if partTransferEncoding == "base64" {
			body, err = base64.StdEncoding.DecodeString(string(body))
			if err != nil {
				return []UserDataPart{}, fmt.Errorf("error decoding base64: %w", err)
			}
		}
		switch partMediaType {
		case "text/cloud-config":
			part, err := payloadAsCloudConfigPart(string(body), env)
			if err != nil {
				return []UserDataPart{}, fmt.Errorf("error parsing cloud-config: %w", err)
			}

			udParts = append(udParts, part)
			continue
		case "text/x-shellscript":
			part, err := payloadAsScriptPart(string(body), env)
			if err != nil {
				return []UserDataPart{}, fmt.Errorf("error parsing script: %w", err)
			}
			udParts = append(udParts, part)
			continue
		case "application/gzip",
			"application/gzip-compressed",
			"application/gzipped",
			"application/x-compress",
			"application/x-compressed",
			"application/x-gunzip",
			"application/x-gzip",
			"application/x-gzip-compressed":

			gzr, err := gzip.NewReader(bytes.NewReader(body))
			if err != nil {
				return []UserDataPart{}, fmt.Errorf("error reading gzip: %w", err)
			}
			body, err = ioutil.ReadAll(gzr)
			if err != nil {
				gzr.Close()
				return []UserDataPart{}, fmt.Errorf("error reading gzip: %w", err)
			}
			gzr.Close()
			// with the gzip wrapper removed, we can now parse the part. Fallthrough to
			// the default condition, which will attempt to detect the part type and return
			// a UserDataPart.
			fallthrough
		default:
			parsedParts, err := partsFromUserData(string(body), env)
			if err != nil {
				return []UserDataPart{}, fmt.Errorf("error parsing part: %w", err)
			}
			udParts = append(udParts, parsedParts...)
		}
	}

	return udParts, nil
}

func payloadAsScriptPart(payload string, env *Environment) (UserDataPart, error) {
	if env == nil {
		return UserDataPart{}, fmt.Errorf("environment is nil")
	}

	userdata := env.Apply(payload)
	if !config.IsScript(userdata) {
		return UserDataPart{}, fmt.Errorf("payload is not a script")
	}
	script, err := config.NewScript(userdata)
	if err != nil {
		return UserDataPart{}, err
	}
	return UserDataPart{
		userDataType: ScriptType,
		contents:     userdata,
		script:       script,
	}, nil
}

func payloadAsCloudConfigPart(payload string, env *Environment) (UserDataPart, error) {
	if env == nil {
		return UserDataPart{}, fmt.Errorf("environment is nil")
	}

	userdata := env.Apply(payload)
	cc, err := config.NewCloudConfig(userdata)
	if err != nil {
		return UserDataPart{}, err
	}

	if err := cc.Decode(); err != nil {
		return UserDataPart{}, err
	}

	if userdata[:len("#cloud-config")] != "#cloud-config" {
		// add the header if it's missing. When parsing multipart MIME, we get the type
		// of the userdata from the Content-Type header, so we don't require that the body contain
		// the header, but it simplifies our lives if we add it here, as there are functions that look
		// for it in other parts of the codebase.
		userdata += "#cloud-config\n\n"
	}
	return UserDataPart{
		userDataType: CloudConfigType,
		contents:     userdata,
		cloudConfig:  cc,
	}, nil
}

func partsFromUserData(payload string, env *Environment) ([]UserDataPart, error) {
	if env == nil {
		return []UserDataPart{}, fmt.Errorf("environment is nil")
	}

	var parts []UserDataPart
	switch {
	case config.IsScript(payload):
		part, err := payloadAsScriptPart(payload, env)
		if err != nil {
			return nil, fmt.Errorf("error parsing script: %w", err)
		}
		parts = append(parts, part)
	case config.IsCloudConfig(payload):
		part, err := payloadAsCloudConfigPart(payload, env)
		if err != nil {
			return nil, fmt.Errorf("error parsing cloud-config: %w", err)
		}
		parts = append(parts, part)
	case config.IsIgnitionConfig(payload):
		// we don't actually do anything with it, but we add it as a part
		// and log a warning later.
		part := UserDataPart{
			userDataType: IgnitionType,
			contents:     payload,
		}
		parts = append(parts, part)
	case config.IsMultipartMime(payload):
		udParts, err := multipartToUserDataParts(payload, env)
		if err != nil {
			return nil, fmt.Errorf("error parsing multipart MIME: %w", err)
		}
		parts = append(parts, udParts...)
	default:
		parts = append(parts, UserDataPart{
			userDataType: UnknownType,
			contents:     payload,
		})
	}

	return parts, nil
}

type UserDataPart struct {
	userDataType UserDataType
	contents     string

	cloudConfig *config.CloudConfig
	script      *config.Script
}

func (udp *UserDataPart) PartType() UserDataType {
	return udp.userDataType
}

func (udp *UserDataPart) IsCloudConfig() bool {
	return udp.userDataType == CloudConfigType
}

func (udp *UserDataPart) IsScript() bool {
	return udp.userDataType == ScriptType
}

func (udp *UserDataPart) IsIgnition() bool {
	return udp.userDataType == IgnitionType
}

func (udp *UserDataPart) IsUnknown() bool {
	return udp.userDataType == UnknownType
}

func (udp *UserDataPart) runScript(env *Environment) error {
	if env == nil {
		return fmt.Errorf("environment is nil")
	}

	err := PrepWorkspace(env.Workspace())
	if err != nil {
		log.Printf("Failed preparing workspace: %v\n", err)
		return err
	}
	path, err := PersistScriptInWorkspace(*udp.script, env.Workspace())
	if err == nil {
		var name string
		name, err = system.ExecuteScript(path)
		PersistUnitNameInWorkspace(name, env.Workspace())
	}
	return err
}

func (udp *UserDataPart) runCloudConfig(env *Environment) error {
	if err := Apply(*udp.cloudConfig, env); err != nil {
		return fmt.Errorf("error applying cloud-config: %w", err)
	}
	return nil
}

func (udp *UserDataPart) RunPart(env *Environment) error {
	switch udp.userDataType {
	case ScriptType:
		return udp.runScript(env)
	case CloudConfigType:
		return udp.runCloudConfig(env)
	default:
		log.Printf("ignoring part of type %s", udp.userDataType)
	}
	return nil
}

type UserData struct {
	Parts []UserDataPart

	env *Environment
}

func (ud *UserData) FindHostname() string {
	for _, part := range ud.Parts {
		if part.cloudConfig != nil && part.cloudConfig.Hostname != "" {
			return part.cloudConfig.Hostname
		}
	}
	return ""
}

func (ud *UserData) FindSSHKeys(additionalKeys []string) []string {
	keys := []string{}
	hasKey := func(key string) bool {
		for _, k := range keys {
			if k == key {
				return true
			}
		}
		return false
	}

	for _, part := range ud.Parts {
		if part.cloudConfig != nil {
			keys = append(keys, part.cloudConfig.SSHAuthorizedKeys...)

			if part.cloudConfig.Users != nil {
				for _, user := range part.cloudConfig.Users {
					if user.Name != "core" {
						continue
					}

					// The "core" user is the default user on coreos systems. Append these keys to the list.
					for _, key := range user.SSHAuthorizedKeys {
						if !hasKey(key) {
							keys = append(keys, key)
						}
					}
				}
			}
		}
	}
	return append(keys, additionalKeys...)
}
