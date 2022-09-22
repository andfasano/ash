package ash

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"
)

type Machine struct {
	Name    string `yaml:"name"`
	Profile string `yaml:"profile"`
	IP      string `yaml:"ip"`
	Mac     string `yaml:"mac"`

	//Profile overrides
	VCPUs   string `yaml:"vcpus"`
	Memory  string `yaml:"memory"`
	Disk    string `yaml:"disk"`
	Network string `yaml:"network"`
}

const (
	qemuOUI = "52:54:00"

	localBit    = 0x2
	unicastMask = 0xfe
)

func GenerateRandomMAC() string {
	buf := make([]byte, 3)
	rand.Read(buf)
	buf[0] = (buf[0] | localBit) & unicastMask

	return fmt.Sprintf("%s:%02x:%02x:%02x", qemuOUI, buf[0], buf[1], buf[2])
}

func (m Machine) getSpecsFrom(specs string) (uint64, string) {
	r := regexp.MustCompile(`(?m)^(\d+)\s?(\w+)$`)
	matches := r.FindStringSubmatch(specs)
	size, _ := strconv.Atoi(matches[1])
	return uint64(size), matches[2]
}

func (m Machine) GetDiskSpecs() (uint64, string) {
	return m.getSpecsFrom(m.Disk)
}

func (m Machine) GetMemorySpecs() (uint64, string) {
	return m.getSpecsFrom(m.Memory)
}
