# Cloud-Config Locations

---

**NOTE**: This project overlaps in purpose with [Ignition][ignition] which is where most active development is taking place. However, the Flatcar Container Linux team also continues to support and maintain this project to maintain compatibility with cloudinit based environments.

[ignition]: https://docs.flatcar-linux.org/ignition/what-is-ignition/#what-is-ignition
[provisioning]: https://docs.flatcar-linux.org/os/provisioning/

---

On every boot, coreos-cloudinit looks for a config file to configure your host. Here is a list of locations which are used by the Cloud-Config utility, depending on your Flatcar Container Linux platform:

| Location | Description |
| --- | --- |
| `/media/configvirtfs/openstack/latest/user_data` | `/media/configvirtfs` mount point with [config-2](config-drive.md#contents-and-format) label. It should contain a `openstack/latest/user_data` relative path. Usually used by cloud providers or in VM installations. |
| `/media/configdrive/openstack/latest/user_data` | FAT or ISO9660 filesystem with [config-2](config-drive.md#qemu-virtfs) label and `/media/configdrive/` mount point. It should also contain a `openstack/latest/user_data` relative path. Usually used in installations which are configured by USB Flash sticks or CDROM media. |
| Kernel command line: `cloud-config-url=http://example.com/user_data`. | You can find this string using this command `cat /proc/cmdline`. Usually used in [PXE](https://docs.flatcar-linux.org/os/booting-with-pxe/) or [iPXE](https://docs.flatcar-linux.org/os/booting-with-ipxe/) boots. |
| `/var/lib/coreos-install/user_data` | When you install Flatcar COntainer Linux manually using the [flatcar-install](https://docs.flatcar-linux.org/os/installing-to-disk/#installing-flatcar-container-linux-to-disk) tool. Usually used in bare metal installations. |
| `/usr/share/oem/cloud-config.yml` | Path for OEM images. |
| `/var/lib/coreos-vagrant/vagrantfile-user-data`| Vagrant OEM scripts automatically store Cloud-Config into this path. |
| `/var/lib/waagent/CustomData`| Azure platform uses OEM path for first Cloud-Config initialization and then `/var/lib/waagent/CustomData` to apply your settings. |
| `http://169.254.169.254/metadata/v1/user-data` `http://169.254.169.254/2009-04-04/user-data` `https://metadata.packet.net/userdata`|DigitalOcean, EC2 and Packet cloud providers correspondingly use these URLs to download Cloud-Config.|
| `/usr/share/oem/bin/vmtoolsd --cmd "info-get guestinfo.coreos.config.data"` | Cloud-Config provided by [VMware Guestinfo][VMware Guestinfo] |
| `/usr/share/oem/bin/vmtoolsd --cmd "info-get guestinfo.coreos.config.url"` | Cloud-Config URL provided by [VMware Guestinfo][VMware Guestinfo] |

[VMware Guestinfo]: vmware-guestinfo.md

You can also run the `coreos-cloudinit` tool manually and provide a path to your custom Cloud-Config file:

```sh
sudo coreos-cloudinit --from-file=/home/core/cloud-config.yaml
```

This command will apply your custom cloud-config.
