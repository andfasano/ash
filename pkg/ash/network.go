package ash

const (
	// Machines address start from CIDR + 80
	machinesBaseAddrStart = 80
)

type Network struct {
	Name         string `yaml:"name"`
	Cidr         string `yaml:"cidr"`
	Disconnected string `yaml:"disconnected"`
}
