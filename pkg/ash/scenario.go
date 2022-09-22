package ash

const (
	scenarioFilename = "scenario.yaml"
)

type Scenario struct {
	Name     string    `yaml:"name"`
	Machines []Machine `yaml:"machines"`

	MachineProfiles []MachineProfile
	Networks        []Network
}
