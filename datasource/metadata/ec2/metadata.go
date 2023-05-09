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

package ec2

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/flatcar/coreos-cloudinit/datasource"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata"
	"github.com/flatcar/coreos-cloudinit/pkg"
)

const (
	DefaultAddress = "http://169.254.169.254/"
	apiVersion     = "2009-04-04/"
	userdataPath   = apiVersion + "user-data"
	metadataPath   = apiVersion + "meta-data"
)

type metadataService struct {
	metadata.MetadataService
}

func NewDatasource(root string) *metadataService {
	if token, err := fetchToken(); err == nil {
		tokenHeader := http.Header(map[string][]string{"X-aws-ec2-metadata-token": {string(token)}})
		return &metadataService{metadata.NewDatasource(root, apiVersion, userdataPath, metadataPath, tokenHeader)}
	} else {
		log.Printf("error: %v", err)
	}
	return &metadataService{metadata.NewDatasource(root, apiVersion, userdataPath, metadataPath, nil)}
}

func (ms metadataService) FetchMetadata() (datasource.Metadata, error) {
	metadata := datasource.Metadata{}

	if keynames, err := ms.fetchAttributes(fmt.Sprintf("%s/public-keys", ms.MetadataUrl())); err == nil {
		keyIDs := make(map[string]string)
		for _, keyname := range keynames {
			tokens := strings.SplitN(keyname, "=", 2)
			if len(tokens) != 2 {
				return metadata, fmt.Errorf("malformed public key: %q", keyname)
			}
			keyIDs[tokens[1]] = tokens[0]
		}

		metadata.SSHPublicKeys = map[string]string{}
		for name, id := range keyIDs {
			sshkey, err := ms.fetchAttribute(fmt.Sprintf("%s/public-keys/%s/openssh-key", ms.MetadataUrl(), id))
			if err != nil {
				return metadata, err
			}
			metadata.SSHPublicKeys[name] = sshkey
			log.Printf("Found SSH key for %q\n", name)
		}
	} else if _, ok := err.(pkg.ErrNotFound); !ok {
		return metadata, err
	}

	if hostname, err := ms.fetchAttribute(fmt.Sprintf("%s/hostname", ms.MetadataUrl())); err == nil {
		hostname := strings.Split(hostname, ".")[0]
		if len(hostname) > 63 {
			hostname = hostname[:63]
		}
		metadata.Hostname = hostname
	} else if _, ok := err.(pkg.ErrNotFound); !ok {
		return metadata, err
	}

	if localAddr, err := ms.fetchAttribute(fmt.Sprintf("%s/local-ipv4", ms.MetadataUrl())); err == nil {
		metadata.PrivateIPv4 = net.ParseIP(localAddr)
	} else if _, ok := err.(pkg.ErrNotFound); !ok {
		return metadata, err
	}

	if publicAddr, err := ms.fetchAttribute(fmt.Sprintf("%s/public-ipv4", ms.MetadataUrl())); err == nil {
		metadata.PublicIPv4 = net.ParseIP(publicAddr)
	} else if _, ok := err.(pkg.ErrNotFound); !ok {
		return metadata, err
	}

	return metadata, nil
}

func (ms metadataService) Type() string {
	return "ec2-metadata-service"
}

// This is separate from the normal HTTP client because it is needed to configure that client.
func fetchToken() ([]byte, error) {
	c := &http.Client{
		Timeout: 10 * time.Second,
	}
	log.Print("fetching token...")
	req, err := http.NewRequest("PUT", DefaultAddress+"latest/api/token", nil)
	if err != nil {
		return nil, err
	}
	// 6 hours
	req.Header.Add("X-aws-ec2-metadata-token-ttl-seconds", "21600")
	if resp, err := c.Do(req); err == nil {
		if resp.StatusCode == 200 {
			return ioutil.ReadAll(resp.Body)
		} else {
			return nil, fmt.Errorf("token response status code %v", resp.StatusCode)
		}
	} else {
		return nil, fmt.Errorf("Unable to fetch data: %s", err.Error())
	}
}

func (ms metadataService) fetchAttributes(url string) ([]string, error) {
	resp, err := ms.FetchData(url)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(resp))
	data := make([]string, 0)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	return data, scanner.Err()
}

func (ms metadataService) fetchAttribute(url string) (string, error) {
	if attrs, err := ms.fetchAttributes(url); err == nil && len(attrs) > 0 {
		return attrs[0], nil
	} else {
		return "", err
	}
}
