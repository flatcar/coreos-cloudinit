package ionoscloud

import (
	"log"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/flatcar/coreos-cloudinit/datasource"
)

const (
	networkconfig = "/etc/cloud/cloud.cfg.d/99_custom_networking.cfg"
)

type ionoscloud struct {
	seed     string
	readFile func(filename string) ([]byte, error)
}

func NewDatasource(seed string) *ionoscloud {
	return &ionoscloud{seed, os.ReadFile}
}

func (ic *ionoscloud) IsAvailable() bool {
	_, err := os.Stat(ic.seed)
	return !os.IsNotExist(err)
}

func (ic *ionoscloud) AvailabilityChanges() bool {
	return true
}

func (ic *ionoscloud) ConfigRoot() string {
	return ic.seed
}

func (ic *ionoscloud) FetchMetadata() (metadata datasource.Metadata, err error) {
	var data []byte
	var m struct {
		DSMode        string            `json:"dsmode"`
		SSHPublicKeys map[string]string `json:"public_keys"`
	}

	if data, err = ic.tryReadFile(path.Join(ic.seed, "meta-data")); err != nil || len(data) == 0 {
		return
	}
	if err = yaml.Unmarshal([]byte(data), &m); err != nil {
		return
	}

	metadata.SSHPublicKeys = m.SSHPublicKeys
	metadata.NetworkConfig, _ = ic.tryReadFile(networkconfig)

	return
}

func (ic *ionoscloud) FetchUserdata() ([]byte, error) {
	return ic.tryReadFile(path.Join(ic.seed, "user-data"))
}

func (ic *ionoscloud) Type() string {
	return "ionoscloud"
}

func (ic *ionoscloud) tryReadFile(filename string) ([]byte, error) {
	log.Printf("Attempting to read from %q\n", filename)
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		err = nil
	}
	return data, err
}
