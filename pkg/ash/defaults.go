package ash

var (
	defaultLabel = "ash"

	defaultProfile = MachineProfile{
		Name:    defaultLabel,
		VCPUs:   "8",
		Memory:  "16GiB",
		Disk:    "120GiB",
		Network: defaultLabel,
	}

	defaultNetwork = Network{
		Name:         defaultLabel,
		Cidr:         "192.168.200.0/24",
		Disconnected: "no",
	}

	defaultBaseDomain = "test.agent.org"
)
