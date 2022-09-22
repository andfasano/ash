package ash

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

var (
	ashPool    = "ash_pool"
	poolFolder = "pool"
)

type libvirtProvider struct {
	log      logr.Logger
	agentISO string
	cacheDir string
}

func NewLibvirtProvider(log logr.Logger, agentISO string, cacheDir string) *libvirtProvider {
	return &libvirtProvider{
		log:      log,
		agentISO: agentISO,
		cacheDir: cacheDir,
	}
}

func (p *libvirtProvider) Setup(scenario Scenario) error {

	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return err
	}
	defer conn.Close()

	network := scenario.Networks[0]

	pool, err := p.createPool(conn)
	if err != nil {
		return err
	}

	_, err = p.createNetwork(conn, network, scenario.Machines)
	if err != nil {
		return err
	}

	for _, machine := range scenario.Machines {
		_, err = p.createVolume(conn, pool, machine)
		if err != nil {
			return err
		}

		_, err = p.createDomain(conn, network, machine)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *libvirtProvider) createPool(conn *libvirt.Connect) (*libvirt.StoragePool, error) {

	pool, err := conn.LookupStoragePoolByName(ashPool)
	if err == nil {
		log.Println("skipping pool creation, already exists", ashPool)
		return pool, nil
	}
	lverr, ok := err.(libvirt.Error)
	if ok && lverr.Code != libvirt.ERR_NO_STORAGE_POOL {
		return nil, err
	}

	poolPath := filepath.Join(p.cacheDir, poolFolder)
	err = os.MkdirAll(poolPath, 0755)
	if err != nil {
		return nil, err
	}

	poolCfg := &libvirtxml.StoragePool{
		Name: ashPool,
		Type: "dir",
		Target: &libvirtxml.StoragePoolTarget{
			Path: poolPath,
			Permissions: &libvirtxml.StoragePoolTargetPermissions{
				Owner: "0",
				Group: "0",
				Mode:  "0755",
			},
		},
	}

	poolDoc, err := poolCfg.Marshal()
	if err != nil {
		return nil, err
	}

	pool, err = conn.StoragePoolCreateXML(poolDoc, libvirt.STORAGE_POOL_CREATE_NORMAL)
	if err != nil {
		return nil, err
	}

	log.Println("storage pool created", ashPool)
	return pool, nil
}

func (p *libvirtProvider) createVolume(conn *libvirt.Connect, pool *libvirt.StoragePool, machine Machine) (*libvirt.StorageVol, error) {

	volumeName := fmt.Sprintf("%s.qcow2", machine.Name)

	volume, err := conn.LookupStorageVolByKey(filepath.Join(p.cacheDir, poolFolder, volumeName))
	if err == nil {
		log.Println("skipping volume creation, already exists", volumeName)
		return volume, nil
	}
	lverr, ok := err.(libvirt.Error)
	if ok && lverr.Code != libvirt.ERR_NO_STORAGE_VOL {
		return nil, err
	}

	diskSize, diskUnit := machine.GetDiskSpecs()

	volCfg := &libvirtxml.StorageVolume{
		Type: "file",
		Name: volumeName,
		Capacity: &libvirtxml.StorageVolumeSize{
			Value: diskSize,
			Unit:  diskUnit,
		},
		Allocation: &libvirtxml.StorageVolumeSize{
			Value: 51318784,
			Unit:  "bytes",
		},
		Target: &libvirtxml.StorageVolumeTarget{
			Format: &libvirtxml.StorageVolumeTargetFormat{
				Type: "qcow2",
			},
			Permissions: &libvirtxml.StorageVolumeTargetPermissions{
				Mode:  "0600",
				Owner: "0",
				Group: "0",
				Label: "system_u:object_r:virt_image_t:s0",
			},
		},
	}

	volDoc, err := volCfg.Marshal()
	if err != nil {
		return nil, err
	}

	volume, err = pool.StorageVolCreateXML(volDoc, 0)
	if err != nil {
		return nil, err
	}

	log.Println("storage volume created", volumeName)
	return volume, nil
}

func (p *libvirtProvider) createNetwork(conn *libvirt.Connect, scenarioNetwork Network, scenarionMachines []Machine) (*libvirt.Network, error) {

	network, err := conn.LookupNetworkByName(scenarioNetwork.Name)
	if err == nil {
		log.Println("skipping network creation, already exists", scenarioNetwork.Name)
		return network, nil
	}
	lverr, ok := err.(libvirt.Error)
	if ok && lverr.Code != libvirt.ERR_NO_NETWORK {
		return nil, err
	}

	networkCfg := libvirtxml.Network{
		Name: scenarioNetwork.Name,
		UUID: uuid.New().String(),
		Forward: &libvirtxml.NetworkForward{
			Mode: "nat",
			NAT: &libvirtxml.NetworkForwardNAT{
				Ports: []libvirtxml.NetworkForwardNATPort{
					{
						Start: 1024,
						End:   65535,
					},
				},
			},
		},
		Bridge: &libvirtxml.NetworkBridge{
			Name:  scenarioNetwork.Name,
			STP:   "on",
			Delay: "0",
		},
		Domain: &libvirtxml.NetworkDomain{
			Name: fmt.Sprintf("%s.%s", scenarioNetwork.Name, defaultBaseDomain),
		},
		DNS: &libvirtxml.NetworkDNS{
			Forwarders: []libvirtxml.NetworkDNSForwarder{
				{
					Domain: fmt.Sprintf("apps.%s.%s", scenarioNetwork.Name, defaultBaseDomain),
					Addr:   "127.0.0.1",
				},
			},
		},
	}

	// IP network configuration
	ip, ipnet, err := net.ParseCIDR(scenarioNetwork.Cidr)
	if err != nil {
		return nil, err
	}

	baseNetworkIp := ip.Mask(ipnet.Mask)
	baseNetworkIp[3] += 1
	dhcpStart := ip.Mask(ipnet.Mask)
	dhcpStart[3] += 20
	dhcpEnd := ip.Mask(ipnet.Mask)
	dhcpEnd[3] += 60

	ips := []libvirtxml.NetworkIP{
		{
			Address: baseNetworkIp.String(),
			Netmask: net.IP(ipnet.Mask).String(),
			DHCP: &libvirtxml.NetworkDHCP{
				Ranges: []libvirtxml.NetworkDHCPRange{
					{
						Start: dhcpStart.String(),
						End:   dhcpEnd.String(),
					},
				},
				Hosts: []libvirtxml.NetworkDHCPHost{},
			},
		},
	}
	// DHCP configuration, auto assign an ip to each machine
	for _, m := range scenarionMachines {
		ips[0].DHCP.Hosts = append(ips[0].DHCP.Hosts, libvirtxml.NetworkDHCPHost{
			MAC:  m.Mac,
			IP:   m.IP,
			Name: m.Name,
		})
	}
	networkCfg.IPs = ips

	networkDoc, err := networkCfg.Marshal()
	if err != nil {
		return nil, err
	}

	network, err = conn.NetworkCreateXML(networkDoc)
	if err != nil {
		return nil, err
	}

	log.Println("network created", scenarioNetwork.Name)
	return network, nil
}

func (p *libvirtProvider) createDomain(conn *libvirt.Connect, network Network, machine Machine) (*libvirt.Domain, error) {

	domain, err := conn.LookupDomainByName(machine.Name)
	if err == nil {
		log.Println("skipping domain creation, already exists", machine.Name)
		return domain, nil
	}
	lverr, ok := err.(libvirt.Error)
	if ok && lverr.Code != libvirt.ERR_NO_DOMAIN {
		return nil, err
	}

	memSize, memUnit := machine.GetMemorySpecs()
	vcpus, err := strconv.Atoi(machine.VCPUs)
	if err != nil {
		return nil, err
	}

	domCfg := &libvirtxml.Domain{
		Type: "kvm",
		Name: machine.Name,
		Metadata: &libvirtxml.DomainMetadata{
			XML: "<libosinfo:libosinfo xmlns:libosinfo=\"http://libosinfo.org/xmlns/libvirt/domain/1.0\"><libosinfo:os id=\"http://fedoraproject.org/coreos/stable\"/></libosinfo:libosinfo>",
		},
		Memory: &libvirtxml.DomainMemory{
			Value: uint(memSize),
			Unit:  memUnit,
		},
		CurrentMemory: &libvirtxml.DomainCurrentMemory{
			Value: uint(memSize),
			Unit:  memUnit,
		},
		VCPU: &libvirtxml.DomainVCPU{
			Value: uint(vcpus),
		},
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch:    "x86_64",
				Machine: "pc-q35-rhel8.6.0",
				Type:    "hvm",
			},
			BootDevices: []libvirtxml.DomainBootDevice{
				{
					Dev: "hd",
				},
				{
					Dev: "cdrom",
				},
			},
		},
		CPU: &libvirtxml.DomainCPU{
			Mode: "host-passthrough",
		},
		Devices: &libvirtxml.DomainDeviceList{
			Emulator: "/usr/libexec/qemu-kvm",
			Disks: []libvirtxml.DomainDisk{
				{
					Device: "disk",
					Driver: &libvirtxml.DomainDiskDriver{
						Name: "qemu",
						Type: "qcow2",
					},
					Source: &libvirtxml.DomainDiskSource{
						Volume: &libvirtxml.DomainDiskSourceVolume{
							Pool:   ashPool,
							Volume: fmt.Sprintf("%s.qcow2", machine.Name),
						},
						Index: 2,
					},
					BackingStore: &libvirtxml.DomainDiskBackingStore{},
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "vda",
						Bus: "virtio",
					},
				},
				{
					Device: "cdrom",
					Driver: &libvirtxml.DomainDiskDriver{
						Name: "qemu",
						Type: "raw",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: p.agentISO,
						},
						Index: 1,
					},
					BackingStore: &libvirtxml.DomainDiskBackingStore{},
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "sdb",
						Bus: "sata",
					},
					ReadOnly: &libvirtxml.DomainDiskReadOnly{},
				},
			},
			Interfaces: []libvirtxml.DomainInterface{
				{
					MAC: &libvirtxml.DomainInterfaceMAC{
						Address: machine.Mac,
					},
					Source: &libvirtxml.DomainInterfaceSource{
						Network: &libvirtxml.DomainInterfaceSourceNetwork{
							Network: network.Name,
						},
					},
					Model: &libvirtxml.DomainInterfaceModel{
						Type: "virtio",
					},
				},
			},
			Graphics: []libvirtxml.DomainGraphic{
				{
					VNC: &libvirtxml.DomainGraphicVNC{
						Port:     -1,
						AutoPort: "yes",
					},
				},
			},
			Videos: []libvirtxml.DomainVideo{
				{
					Model: libvirtxml.DomainVideoModel{
						Type:  "cirrus",
						VRam:  16384,
						Heads: 1,
					},
				},
			},
		},
	}

	domDoc, err := domCfg.Marshal()
	if err != nil {
		return nil, err
	}

	domain, err = conn.DomainCreateXML(domDoc, libvirt.DOMAIN_NONE)
	if err != nil {
		return nil, err
	}

	log.Println("domain created", machine.Name)
	return domain, nil
}

func (p *libvirtProvider) Teardown(scenario Scenario, cacheDir string) error {

	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Remove all the scenario machines and their volumes
	for _, machine := range scenario.Machines {
		domain, err := conn.LookupDomainByName(machine.Name)
		if err == nil {
			err = domain.Destroy()
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println("domain deleted", machine.Name)
			}
		}

		volumeName := fmt.Sprintf("%s.qcow2", machine.Name)
		volume, err := conn.LookupStorageVolByKey(filepath.Join(cacheDir, poolFolder, volumeName))
		if err == nil {
			err = volume.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL)
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Println("volume deleted", volumeName)
			}
		}
	}

	// Remove the pool
	pool, err := conn.LookupStoragePoolByName(ashPool)
	if err == nil {
		err = pool.Destroy()
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Println("pool deleted", ashPool)
		}
	}

	// Remove the network
	network, err := conn.LookupNetworkByName(scenario.Networks[0].Name)
	if err == nil {
		err = network.Destroy()
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Println("network deleted", scenario.Networks[0].Name)
		}
	}

	return nil
}
