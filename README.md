# Agent Scenario Helper PoC

This poc shows how to setup a virtual environment (tailored on the agent scenarios) using the
official [Libvirt Go packages](https://libvirt.org/go/libvirt). Currently works just for SNO
scenario.

1. Copy the scenario files from the /scenarios/[name] folder in /cluster
2. Add your ssh-key and pull-secret in /cluster/install-config.yaml
3. Run `openshift-install agent create image --dir cluster`
4. Run `go run /cmd/ash/main.go setup` to setup the environment
5. Run `go run /cmd/ash/main.go teardown` for cleaning it 