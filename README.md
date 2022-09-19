**NOTE**: This project overlaps in purpose with [Ignition][ignition] which is where most active development is taking place. However, the Flatcar Container Linux team also continues to support and maintain this project to maintain compatibility with cloudinit based environments.

[ignition]: https://github.com/flatcar/ignition

coreos-cloudinit enables a user to customize Flatcar Container Linux machines by providing either a cloud-config document or an executable script through user-data.

## Configuration with cloud-config

A subset of the [official cloud-config spec][official-cloud-config] is implemented by coreos-cloudinit.
Additionally, several [Flatcar-specific options][custom-cloud-config] have been implemented to support interacting with unit files, bootstrapping etcd clusters, and more.
All supported cloud-config parameters are [documented here][all-cloud-config]. 

[official-cloud-config]: http://cloudinit.readthedocs.org/en/latest/topics/format.html#cloud-config-data
[custom-cloud-config]: ./Documentation/cloud-config.md#coreos-parameters
[all-cloud-config]: ./Documentation/cloud-config.md

The following is an example cloud-config document:

```
#cloud-config

coreos:
    units:
      - name: etcd.service
        command: start

users:
  - name: core
    passwd: $1$allJZawX$00S5T756I5PGdQga5qhqv1

write_files:
  - path: /etc/resolv.conf
    content: |
        nameserver 192.0.2.2
        nameserver 192.0.2.3
```

## Executing a Script

coreos-cloudinit supports executing user-data as a script instead of parsing it as a cloud-config document.
Make sure the first line of your user-data is a shebang and coreos-cloudinit will attempt to execute it:

```
#!/bin/bash

echo 'Hello, world!'
```

## user-data Field Substitution

coreos-cloudinit will replace the following set of tokens in your user-data with system-generated values.

| Token         | Description |
| ------------- | ----------- |
| $public_ipv4  | Public IPv4 address of machine |
| $private_ipv4 | Private IPv4 address of machine |

These values are determined by Flatcar Container Linux based on the given provider on which your machine is running.
Read more about provider-specific functionality in the [Flatcar Container Linux OEM documentation][oem-doc].

[oem-doc]: https://www.flatcar.org/docs/latest/installing/cloud/

For example, submitting the following user-data...

```
#cloud-config
coreos:
    etcd:
        addr: $public_ipv4:4001
        peer-addr: $private_ipv4:7001
```

...will result in this cloud-config document being executed:

```
#cloud-config
coreos:
    etcd:
        addr: 203.0.113.29:4001
        peer-addr: 192.0.2.13:7001
```

## Bugs

Please use the [Flatcar Container Linux issue tracker][bugs] to report all bugs, issues, and feature requests.

[bugs]: https://github.com/flatcar/flatcar/issues
