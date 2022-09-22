package ash

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
)

type AgentScenarioHelper struct {
	Scenario  Scenario
	assetsDir string
	cacheDir  string
	log       logr.Logger
}

func NewAgentScenarioHelper(cacheDir string) *AgentScenarioHelper {

	log, _ := logr.FromContext(context.TODO())
	workingDir, _ := os.Getwd()

	return &AgentScenarioHelper{
		assetsDir: filepath.Join(workingDir, "cluster"),
		cacheDir:  cacheDir,
		log:       log,
	}
}

//-----------------------------------------------------------------------------

func (ash *AgentScenarioHelper) Setup() error {

	err := ash.defineScenario()
	if err != nil {
		return err
	}

	log.Println("Setting up scenario", ash.Scenario.Name)

	provider := NewLibvirtProvider(ash.log, filepath.Join(ash.assetsDir, "agent.iso"), ash.cacheDir)
	err = provider.Setup(ash.Scenario)
	if err != nil {
		return err
	}

	return nil
}

//-----------------------------------------------------------------------------

func (ash *AgentScenarioHelper) Teardown() error {

	err := ash.defineScenario()
	if err != nil {
		return err
	}

	log.Println("Cleaning up scenario", ash.Scenario.Name)

	provider := NewLibvirtProvider(ash.log, filepath.Join(ash.assetsDir, "agent.iso"), ash.cacheDir)
	err = provider.Teardown(ash.Scenario, ash.cacheDir)
	if err != nil {
		return err
	}

	return nil
}

//-----------------------------------------------------------------------------

func (ash *AgentScenarioHelper) defineScenario() error {

	scenarioFile, err := os.ReadFile(filepath.Join(ash.assetsDir, scenarioFilename))
	if err != nil {
		return err
	}

	scenario := Scenario{}
	err = yaml.Unmarshal(scenarioFile, &scenario)
	if err != nil {
		return err
	}
	err = ash.setDefaults(&scenario)
	if err != nil {
		return err
	}
	err = ash.setNetworking(&scenario)
	if err != nil {
		return err
	}
	ash.Scenario = scenario

	return nil
}

func (ash *AgentScenarioHelper) setDefaults(scenario *Scenario) error {

	// network and profile currently hardcoded
	scenario.Networks = append(scenario.Networks, defaultNetwork)
	scenario.MachineProfiles = append(scenario.MachineProfiles, defaultProfile)

	for i := 0; i < len(scenario.Machines); i++ {
		machine := scenario.Machines[i]

		// network and profile currently hardcoded
		profile := defaultProfile
		machine.Network = defaultNetwork.Name
		machine.Profile = profile.Name

		// Copy values from profile if missing
		if machine.VCPUs == "" {
			machine.VCPUs = profile.VCPUs
		}
		if machine.Memory == "" {
			machine.Memory = profile.Memory
		}
		if machine.Disk == "" {
			machine.Disk = profile.Disk
		}

		scenario.Machines[i] = machine
	}

	return nil
}

func (ash *AgentScenarioHelper) setNetworking(scenario *Scenario) error {

	ip, ipnet, err := net.ParseCIDR(scenario.Networks[0].Cidr)
	if err != nil {
		return err
	}
	ip = ip.Mask(ipnet.Mask)
	ip[3] += machinesBaseAddrStart

	for i := 0; i < len(scenario.Machines); i++ {
		machine := scenario.Machines[i]

		// Generate a mac if missing
		if machine.Mac == "" {
			machine.Mac = GenerateRandomMAC()
		}

		// Auto assign an IP
		if machine.IP == "" {
			machine.IP = ip.To4().String()
			ip[3]++
		}

		scenario.Machines[i] = machine
	}

	return nil
}
