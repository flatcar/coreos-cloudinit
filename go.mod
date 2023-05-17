module github.com/flatcar/coreos-cloudinit

go 1.18

require (
	github.com/cloudsigma/cepgo v0.0.0-20140805094338-1bfc4895bf5c
	github.com/coreos/go-systemd v0.0.0-20140326023052-4fbc5060a317
	github.com/coreos/yaml v0.0.0-20141224210557-6b16a5714269
	github.com/dotcloud/docker v0.11.2-0.20140522020950-55d41c3e21e1
	github.com/sigma/vmw-ovflib v0.0.0-20150531125353-56b4f44581ca
	github.com/stretchr/testify v1.8.2
	github.com/vmware/vmw-guestinfo v0.0.0-20170622145319-ab8497750719
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/guelfey/go.dbus v0.0.0-20131113121618-f6a3a2366cc3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/smartystreets/goconvey v1.7.2 // indirect
	github.com/tarm/goserial v0.0.0-20140420040555-cdabc8d44e8e // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vmware/vmw-guestinfo => github.com/sigma/vmw-guestinfo v0.0.0-20170622145319-ab8497750719
