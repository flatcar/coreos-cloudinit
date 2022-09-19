# Debian Interfaces #

---

**NOTE**: This project overlaps in purpose with [Ignition][ignition] which is where most active development is taking place. However, the Flatcar Container Linux team also continues to support and maintain this project to maintain compatibility with cloudinit based environments.

[ignition]: https://www.flatcar.org/docs/latest/provisioning/ignition/
[provisioning]: https://www.flatcar.org/docs/latest/provisioning/

---

**WARNING**: This option is EXPERIMENTAL and may change or be removed at any point.

There is basic support for converting from a Debian network configuration to networkd unit files. The -convert-netconf=debian option is used to activate this feature.

# convert-netconf #

Default: ""

Read the network config provided in cloud-drive and translate it from the specified format into networkd unit files (requires the -from-configdrive flag). Currently only supports "debian" which provides support for a small subset of the [Debian network configuration](https://wiki.debian.org/NetworkConfiguration). These options include:

- interface config methods
	- static
		- address/netmask
		- gateway
		- hwaddress
		- dns-nameservers
	- dhcp
		- hwaddress
	- manual
	- loopback
- vlan_raw_device
- bond-slaves
