# Deprecated Cloud-Config Features

---

**NOTE**: This project overlaps in purpose with [Ignition][ignition] which is where most active development is taking place. However, the Flatcar Container Linux team also continues to support and maintain this project to maintain compatibility with cloudinit based environments.

[ignition]: https://www.flatcar.org/docs/latest/provisioning/ignition/
[provisioning]: https://www.flatcar.org/docs/latest/provisioning/

---

## Retrieving SSH Authorized Keys

### From a GitHub User

Using the `coreos-ssh-import-github` field, we can import public SSH keys from a GitHub user to use as authorized keys to a server.

```yaml
#cloud-config

users:
  - name: elroy
    coreos-ssh-import-github: elroy
```

### From an HTTP Endpoint

We can also pull public SSH keys from any HTTP endpoint which matches [GitHub's API response format](https://developer.github.com/v3/users/keys/#list-public-keys-for-a-user).
For example, if you have an installation of GitHub Enterprise, you can provide a complete URL with an authentication token:

```yaml
#cloud-config

users:
  - name: elroy
    coreos-ssh-import-url: https://github-enterprise.example.com/api/v3/users/elroy/keys?access_token=<TOKEN>
```

You can also specify any URL whose response matches the JSON format for public keys:

```yaml
#cloud-config

users:
  - name: elroy
    coreos-ssh-import-url: https://example.com/public-keys
```
