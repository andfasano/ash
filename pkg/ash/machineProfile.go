package ash

type MachineProfile struct {
	Name string `yaml:"name"`

	VCPUs   string `yaml:"vcpus"`
	Memory  string `yaml:"memory"`
	Disk    string `yaml:"disk"`
	Network string `yaml:"network"`
}
