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
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/flatcar/coreos-cloudinit/config"
	"github.com/flatcar/coreos-cloudinit/config/validate"
	"github.com/flatcar/coreos-cloudinit/datasource"
	"github.com/flatcar/coreos-cloudinit/datasource/configdrive"
	"github.com/flatcar/coreos-cloudinit/datasource/file"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata/cloudsigma"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata/digitalocean"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata/ec2"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata/gce"
	"github.com/flatcar/coreos-cloudinit/datasource/metadata/packet"
	"github.com/flatcar/coreos-cloudinit/datasource/proc_cmdline"
	"github.com/flatcar/coreos-cloudinit/datasource/url"
	"github.com/flatcar/coreos-cloudinit/datasource/vmware"
	"github.com/flatcar/coreos-cloudinit/datasource/waagent"
	"github.com/flatcar/coreos-cloudinit/initialize"
	"github.com/flatcar/coreos-cloudinit/network"
	"github.com/flatcar/coreos-cloudinit/pkg"
	"github.com/flatcar/coreos-cloudinit/system"
)

const (
	datasourceInterval    = 100 * time.Millisecond
	datasourceMaxInterval = 30 * time.Second
	datasourceTimeout     = 5 * time.Minute
)

var (
	flags = struct {
		printVersion  bool
		ignoreFailure bool
		sources       struct {
			file                        string
			configDrive                 string
			waagent                     string
			metadataService             bool
			ec2MetadataService          string
			gceMetadataService          string
			cloudSigmaMetadataService   bool
			digitalOceanMetadataService string
			packetMetadataService       string
			url                         string
			procCmdLine                 bool
			vmware                      bool
			ovfEnv                      string
		}
		convertNetconf string
		workspace      string
		sshKeyName     string
		oem            string
		validate       bool
	}{}
	version = "was not built properly"
)

func init() {
	flag.BoolVar(&flags.printVersion, "version", false, "Print the version and exit")
	flag.BoolVar(&flags.ignoreFailure, "ignore-failure", false, "Exits with 0 status in the event of malformed input from user-data")
	flag.StringVar(&flags.sources.file, "from-file", "", "Read user-data from provided file")
	flag.StringVar(&flags.sources.configDrive, "from-configdrive", "", "Read data from provided cloud-drive directory")
	flag.StringVar(&flags.sources.waagent, "from-waagent", "", "Read data from provided waagent directory")
	flag.BoolVar(&flags.sources.metadataService, "from-metadata-service", false, "[DEPRECATED - Use -from-ec2-metadata] Download data from metadata service")
	flag.StringVar(&flags.sources.ec2MetadataService, "from-ec2-metadata", "", "Download EC2 data from the provided url")
	flag.StringVar(&flags.sources.gceMetadataService, "from-gce-metadata", "", "Download GCE data from the provided url")
	flag.BoolVar(&flags.sources.cloudSigmaMetadataService, "from-cloudsigma-metadata", false, "Download data from CloudSigma server context")
	flag.StringVar(&flags.sources.digitalOceanMetadataService, "from-digitalocean-metadata", "", "Download DigitalOcean data from the provided url")
	flag.StringVar(&flags.sources.packetMetadataService, "from-packet-metadata", "", "Download Packet data from metadata service")
	flag.StringVar(&flags.sources.url, "from-url", "", "Download user-data from provided url")
	flag.BoolVar(&flags.sources.procCmdLine, "from-proc-cmdline", false, fmt.Sprintf("Parse %s for '%s=<url>', using the cloud-config served by an HTTP GET to <url>", proc_cmdline.ProcCmdlineLocation, proc_cmdline.ProcCmdlineCloudConfigFlag))
	flag.BoolVar(&flags.sources.vmware, "from-vmware-guestinfo", false, "Read data from VMware guestinfo")
	flag.StringVar(&flags.sources.ovfEnv, "from-vmware-ovf-env", "", "Read data from OVF Environment")
	flag.StringVar(&flags.oem, "oem", "", "Use the settings specific to the provided OEM")
	flag.StringVar(&flags.convertNetconf, "convert-netconf", "", "Read the network config provided in cloud-drive and translate it from the specified format into networkd unit files")
	flag.StringVar(&flags.workspace, "workspace", "/var/lib/coreos-cloudinit", "Base directory coreos-cloudinit should use to store data")
	flag.StringVar(&flags.sshKeyName, "ssh-key-name", initialize.DefaultSSHKeyName, "Add SSH keys to the system with the given name")
	flag.BoolVar(&flags.validate, "validate", false, "[EXPERIMENTAL] Validate the user-data but do not apply it to the system")
}

type oemConfig map[string]string

var (
	oemConfigs = map[string]oemConfig{
		"digitalocean": {
			"from-digitalocean-metadata": "http://169.254.169.254/",
		},
		"ec2-compat": {
			"from-ec2-metadata": "http://169.254.169.254/",
			"from-configdrive":  "/media/configdrive",
		},
		"gce": {
			"from-gce-metadata": "http://metadata.google.internal/",
		},
		"rackspace-onmetal": {
			"from-configdrive": "/media/configdrive",
			"convert-netconf":  "debian",
		},
		"azure": {
			"from-waagent": "/var/lib/waagent",
		},
		"cloudsigma": {
			"from-cloudsigma-metadata": "true",
		},
		"packet": {
			"from-packet-metadata": "https://metadata.packet.net/",
		},
		"vmware": {
			"from-vmware-guestinfo": "true",
			"convert-netconf":       "vmware",
		},
	}
)

func main() {
	failure := false

	// Conservative Go 1.5 upgrade strategy:
	// keep GOMAXPROCS' default at 1 for now.
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(1)
	}

	flag.Parse()

	if c, ok := oemConfigs[flags.oem]; ok {
		for k, v := range c {
			flag.Set(k, v)
		}
	} else if flags.oem != "" {
		oems := make([]string, 0, len(oemConfigs))
		for k := range oemConfigs {
			oems = append(oems, k)
		}
		fmt.Printf("Invalid option to -oem: %q. Supported options: %q\n", flags.oem, oems)
		os.Exit(2)
	}

	if flags.printVersion {
		fmt.Printf("coreos-cloudinit %s\n", version)
		os.Exit(0)
	}

	switch flags.convertNetconf {
	case "":
	case "debian":
	case "packet":
	case "vmware":
	default:
		fmt.Printf("Invalid option to -convert-netconf: '%s'. Supported options: 'debian, packet, vmware'\n", flags.convertNetconf)
		os.Exit(2)
	}

	dss := getDatasources()
	if len(dss) == 0 {
		fmt.Println("Provide at least one of --from-file, --from-configdrive, --from-ec2-metadata, --from-gce-metadata, --from-cloudsigma-metadata, --from-packet-metadata, --from-digitalocean-metadata, --from-vmware-guestinfo, --from-waagent, --from-url or --from-proc-cmdline")
		os.Exit(2)
	}

	ds := selectDatasource(dss)
	if ds == nil {
		log.Println("No datasources available in time")
		os.Exit(1)
	}

	log.Printf("Fetching meta-data from datasource of type %q\n", ds.Type())
	metadata, err := ds.FetchMetadata()
	if err != nil {
		log.Printf("Failed fetching meta-data from datasource: %v\n", err)
		os.Exit(1)
	}
	env := initialize.NewEnvironment("/", ds.ConfigRoot(), flags.workspace, flags.sshKeyName, metadata)

	// Setup networking units
	if flags.convertNetconf != "" {
		if err := setupNetworkUnits(metadata.NetworkConfig, env, flags.convertNetconf); err != nil {
			log.Printf("Failed to setup network units: %v\n", err)
			os.Exit(1)
		}
	}

	log.Printf("Fetching user-data from datasource of type %q\n", ds.Type())
	userdataBytes, err := ds.FetchUserdata()
	if err != nil {
		log.Printf("Failed fetching user-data from datasource: %v. Continuing...\n", err)
		failure = true
	}
	userdataBytes, err = decompressIfGzip(userdataBytes)
	if err != nil {
		log.Printf("Failed decompressing user-data from datasource: %v. Continuing...\n", err)
		failure = true
	}

	if report, err := validate.Validate(userdataBytes); err == nil {
		ret := 0
		for _, e := range report.Entries() {
			log.Println(e)
			ret = 1
		}
		if flags.validate {
			os.Exit(ret)
		}
	} else {
		log.Printf("Failed while validating user_data (%q)\n", err)
		if flags.validate {
			os.Exit(1)
		}
	}

	udata, err := initialize.NewUserData(string(userdataBytes), env)
	if err != nil {
		log.Printf("Failed to parse user-data: %v\nContinuing...\n", err)
		failure = true
	}

	mustStop := false
	hostname := determineHostname(metadata, udata)
	if err := initialize.ApplyHostname(hostname); err != nil {
		log.Printf("Failed to set hostname: %v", err)
		mustStop = true
	}

	mergedKeys := mergeSSHKeysFromSources(metadata, udata)
	if err := initialize.ApplyCoreUserSSHKeys(mergedKeys, env); err != nil {
		log.Printf("Failed to apply SSH keys: %v", err)
		mustStop = true
	}

	if mustStop {
		// We try to set both the hostname and SSH keys. If either fails, we stop.
		// We don't stop if hostname fails to be set, because we may still be able to set
		// the SSH keys and access the server to debug. However, if an error is encountered
		// in either of the two operations, we exit with a non-zero status.
		os.Exit(1)
	}

	if !failure && udata != nil {
		for _, part := range udata.Parts {
			log.Printf("Running part %q (%s)", part.PartName(), part.PartType())
			if err := part.RunPart(env); err != nil {
				log.Printf("Failed to run part %q: %v", part.PartName(), err)
				failure = true
			}
		}
	}

	if failure && !flags.ignoreFailure {
		os.Exit(1)
	}
}

// determineHostname returns either the hostname from the metadata, or the hostname from the
// supplied cloud-config. The cloud-config hostname takes precedence, and we stop after the first
// cloud-config that gives us a hostname.
func determineHostname(md datasource.Metadata, udata *initialize.UserData) string {
	hostname := md.Hostname
	if udata != nil {
		udataHostname := udata.FindHostname()
		if udataHostname != "" {
			hostname = udataHostname
		}
	}
	// Always truncate hostnames to everything before the first `.`
	hostname = strings.Split(hostname, ".")[0]

	// Truncate after 63 characters if the hostname exceeds that
	if len(hostname) > 63 {
		log.Printf("Hostname too long. Truncating hostname %s to 63 bytes (%s)", hostname, hostname[:63])
		hostname = hostname[:63]

	}
	return hostname
}

// mergeSSHKeysFromSources creates a list of all SSH keys from meta-data and the supplied
// cloud-config sources.
func mergeSSHKeysFromSources(md datasource.Metadata, udata *initialize.UserData) []string {
	keys := []string{}
	for _, key := range md.SSHPublicKeys {
		keys = append(keys, key)
	}

	if udata != nil {
		return udata.FindSSHKeys(keys)
	}

	return keys
}

func setupNetworkUnits(netConfig interface{}, env *initialize.Environment, netconf string) error {
	var ifaces []network.InterfaceGenerator
	var err error
	switch netconf {
	case "debian":
		ifaces, err = network.ProcessDebianNetconf(netConfig.([]byte))
	case "packet":
		ifaces, err = network.ProcessPacketNetconf(netConfig.(packet.NetworkData))
	case "vmware":
		ifaces, err = network.ProcessVMwareNetconf(netConfig.(map[string]string))
	default:
		err = fmt.Errorf("Unsupported network config format %q", netconf)
	}
	if err != nil {
		return fmt.Errorf("error generating interfaces: %w", err)
	}

	if err := initialize.ApplyNetworkConfig(ifaces, env); err != nil {
		return fmt.Errorf("error applying network config: %w", err)
	}
	return nil
}

// getDatasources creates a slice of possible Datasources for cloudinit based
// on the different source command-line flags.
func getDatasources() []datasource.Datasource {
	dss := make([]datasource.Datasource, 0, 5)
	if flags.sources.file != "" {
		dss = append(dss, file.NewDatasource(flags.sources.file))
	}
	if flags.sources.url != "" {
		dss = append(dss, url.NewDatasource(flags.sources.url))
	}
	if flags.sources.configDrive != "" {
		dss = append(dss, configdrive.NewDatasource(flags.sources.configDrive))
	}
	if flags.sources.metadataService {
		dss = append(dss, ec2.NewDatasource(ec2.DefaultAddress))
	}
	if flags.sources.ec2MetadataService != "" {
		dss = append(dss, ec2.NewDatasource(flags.sources.ec2MetadataService))
	}
	if flags.sources.gceMetadataService != "" {
		dss = append(dss, gce.NewDatasource(flags.sources.gceMetadataService))
	}
	if flags.sources.cloudSigmaMetadataService {
		dss = append(dss, cloudsigma.NewServerContextService())
	}
	if flags.sources.digitalOceanMetadataService != "" {
		dss = append(dss, digitalocean.NewDatasource(flags.sources.digitalOceanMetadataService))
	}
	if flags.sources.waagent != "" {
		dss = append(dss, waagent.NewDatasource(flags.sources.waagent))
	}
	if flags.sources.packetMetadataService != "" {
		dss = append(dss, packet.NewDatasource(flags.sources.packetMetadataService))
	}
	if flags.sources.procCmdLine {
		dss = append(dss, proc_cmdline.NewDatasource())
	}
	if flags.sources.vmware {
		dss = append(dss, vmware.NewDatasource(""))
	}
	if flags.sources.ovfEnv != "" {
		dss = append(dss, vmware.NewDatasource(flags.sources.ovfEnv))
	}
	return dss
}

// selectDatasource attempts to choose a valid Datasource to use based on its
// current availability. The first Datasource to report to be available is
// returned. Datasources will be retried if possible if they are not
// immediately available. If all Datasources are permanently unavailable or
// datasourceTimeout is reached before one becomes available, nil is returned.
func selectDatasource(sources []datasource.Datasource) datasource.Datasource {
	ds := make(chan datasource.Datasource)
	stop := make(chan struct{})
	var wg sync.WaitGroup

	for _, s := range sources {
		wg.Add(1)
		go func(s datasource.Datasource) {
			defer wg.Done()

			duration := datasourceInterval
			for {
				log.Printf("Checking availability of %q\n", s.Type())
				if s.IsAvailable() {
					ds <- s
					return
				} else if !s.AvailabilityChanges() {
					return
				}
				select {
				case <-stop:
					return
				case <-time.After(duration):
					duration = pkg.ExpBackoff(duration, datasourceMaxInterval)
				}
			}
		}(s)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var s datasource.Datasource
	select {
	case s = <-ds:
	case <-done:
	case <-time.After(datasourceTimeout):
	}

	close(stop)
	return s
}

// TODO(jonboulle): this should probably be refactored and moved into a different module
func runScript(script config.Script, env *initialize.Environment) error {
	err := initialize.PrepWorkspace(env.Workspace())
	if err != nil {
		log.Printf("Failed preparing workspace: %v\n", err)
		return err
	}
	path, err := initialize.PersistScriptInWorkspace(script, env.Workspace())
	if err == nil {
		var name string
		name, err = system.ExecuteScript(path)
		initialize.PersistUnitNameInWorkspace(name, env.Workspace())
	}
	return err
}

const gzipMagicBytes = "\x1f\x8b"

func decompressIfGzip(userdataBytes []byte) ([]byte, error) {
	if !bytes.HasPrefix(userdataBytes, []byte(gzipMagicBytes)) {
		return userdataBytes, nil
	}
	gzr, err := gzip.NewReader(bytes.NewReader(userdataBytes))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	return ioutil.ReadAll(gzr)
}
