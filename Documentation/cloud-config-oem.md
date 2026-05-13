## OEM configuration

---

**NOTE**: cloud-init overlaps in purpose with [Ignition][ignition] which is where most active development is taking place. However, the Flatcar Container Linux team also continues to support and maintain this project to maintain compatibility with cloudinit based environments.

[ignition]: https://www.flatcar.org/docs/latest/provisioning/ignition/
[provisioning]: https://www.flatcar.org/docs/latest/provisioning/

---

The `coreos.oem.*` parameters follow the [os-release spec][os-release], but have been repurposed as a way for coreos-cloudinit to know about the OEM partition on this machine. Customizing this section is only needed when generating a new OEM of Flatcar Container Linux from the SDK. The fields include:

- **id**: Lowercase string identifying the OEM
- **name**: Human-friendly string representing the OEM
- **version-id**: Lowercase string identifying the version of the OEM
- **home-url**: Link to the homepage of the provider or OEM
- **bug-report-url**: Link to a place to file bug reports about this OEM

coreos-cloudinit renders these fields to `/etc/oem-release`.
If no **id** field is provided, coreos-cloudinit will ignore this section.

For example, the following cloud-config document...

```yaml
#cloud-config
coreos:
  oem:
    id: "cloudstack"
    name: "CloudStack"
    version-id: "168.0.0"
    home-url: "https://cloudstack.apache.org/"
    bug-report-url: "https://github.com/flatcar/flatcar/issues"
```

...would be rendered to the following `/etc/oem-release`:

```yaml
ID=cloudstack
NAME="CloudStack"
VERSION_ID=168.0.0
HOME_URL="https://cloudstack.apache.org/"
BUG_REPORT_URL="https://github.com/flatcar/flatcar/issues"
```

[os-release]: http://www.freedesktop.org/software/systemd/man/os-release.html
