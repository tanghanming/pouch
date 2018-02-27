package main

import (
	"runtime"
	"strings"

	"github.com/alibaba/pouch/test/command"
	"github.com/alibaba/pouch/test/environment"
	"github.com/go-check/check"
	"github.com/gotestyourself/gotestyourself/icmd"
)

// PouchNetworkSuite is the test suite fo network CLI.
type PouchNetworkSuite struct{}

func init() {
	check.Suite(&PouchNetworkSuite{})
}

// SetUpSuite does common setup in the beginning of each test suite.
func (suite *PouchNetworkSuite) SetUpSuite(c *check.C) {
	SkipIfFalse(c, environment.IsLinux)
	command.PouchRun("pull", busyboxImage).Assert(c, icmd.Success)

	// Remove all Containers, in case there are legacy containers connecting network.
	environment.PruneAllContainers(apiClient)
}

// TestNetworkDefault tests the creation of default bridge/none/host network.
func (suite *PouchNetworkSuite) TestNetworkDefault(c *check.C) {
	pc, _, _, _ := runtime.Caller(0)
	tmpname := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	var funcname string
	for i := range tmpname {
		funcname = tmpname[i]
	}

	// After pouchd is launched, default netowrk bridge is created
	// check the existence of default network: bridge
	command.PouchRun("network", "inspect", "bridge").Assert(c, icmd.Success)

	command.PouchRun("network", "inspect", "none").Assert(c, icmd.Success)

	command.PouchRun("network", "inspect", "host").Assert(c, icmd.Success)

	// Check the existence of link: p0
	icmd.RunCommand("ip", "link", "show", "dev", "p0").Assert(c, icmd.Success)

	{
		// Assign the none network to a container.
		expct := icmd.Expected{
			Out: "",
		}
		command.PouchRun("run", "--name", funcname, "--net", funcname, busyboxImage, "ip", "r").Compare(expct)
		command.PouchRun("rm", "-f", funcname)
	}
	{
		routeOnHost := icmd.RunCommand("ip", "r").Stdout()
		// Assign the host network to a container.
		expct := icmd.Expected{
			Out: routeOnHost,
		}
		command.PouchRun("run", "--name", funcname, "--net", funcname, busyboxImage, "ip", "r").Compare(expct)
		command.PouchRun("rm", "-f", funcname)
	}
}

// TestNetworkBridgeWorks tests bridge network works.
func (suite *PouchNetworkSuite) TestNetworkBridgeWorks(c *check.C) {
	pc, _, _, _ := runtime.Caller(0)
	tmpname := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	var funcname string
	for i := range tmpname {
		funcname = tmpname[i]
	}

	// Remove network in case there is legacy network which may impacts test.
	defer command.PouchRun("network", "remove", funcname)

	gateway := "192.168.4.1"
	subnet := "192.168.4.0/24"

	command.PouchRun("network", "create", "--name", funcname,
		"-d", "bridge",
		"--gateway", gateway,
		"--subnet", subnet).Assert(c, icmd.Success)
	command.PouchRun("network", "inspect", funcname).Assert(c, icmd.Success)

	// Assign network to a container works
	expct := icmd.Expected{
		Out: "eth0",
	}
	command.PouchRun("run", "--name", funcname, "--net", funcname, busyboxImage, "ip", "link", "ls", "eth0").Compare(expct)
	command.PouchRun("rm", "-f", funcname)

	// remove network should fail
	expct = icmd.Expected{
		ExitCode: 1,
		Err:      "has active endpoints",
	}
	command.PouchRun("run", "--name", funcname, "--net", funcname, busyboxImage, "top").Assert(c, icmd.Success)
	command.PouchRun("network", "remove", funcname).Compare(expct)
	command.PouchRun("rm", "-f", funcname).Assert(c, icmd.Success)

	// TODO: check when remove container, the corresponding veth device on host should also be deleted

	// TODO: remove this network when function EndpointRemove in mgr/network.go is implemented.
	//command.PouchRun("network", "remove", funcname).Assert(c, icmd.Success)
}

// TestNetworkCreateWrongDriver tests using wrong driver returns error.
func (suite *PouchNetworkSuite) TestNetworkCreateWrongDriver(c *check.C) {
	pc, _, _, _ := runtime.Caller(0)
	tmpname := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	var funcname string
	for i := range tmpname {
		funcname = tmpname[i]
	}

	expct := icmd.Expected{
		ExitCode: 1,
		Err:      "not found",
	}

	command.PouchRun("network", "create", "--name", funcname, "--driver", "wrongdriver").Compare(expct)
	command.PouchRun("network", "remove", funcname)
}

// TestNetworkCreateWithLabel tests creating network with label.
func (suite *PouchNetworkSuite) TestNetworkCreateWithLabel(c *check.C) {
	pc, _, _, _ := runtime.Caller(0)
	tmpname := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	var funcname string
	for i := range tmpname {
		funcname = tmpname[i]
	}

	gateway := "192.168.3.1"
	subnet := "192.168.3.0/24"

	command.PouchRun("network", "create",
		"--name", funcname,
		"-d", "bridge",
		"--gateway", gateway,
		"--subnet", subnet,
		"--label", "test=foo").Assert(c, icmd.Success)
	command.PouchRun("network", "remove", funcname)
}

// TestNetworkCreateWithOption tests creating network with option.
func (suite *PouchNetworkSuite) TestNetworkCreateWithOption(c *check.C) {
	pc, _, _, _ := runtime.Caller(0)
	tmpname := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	var funcname string
	for i := range tmpname {
		funcname = tmpname[i]
	}

	gateway := "192.168.100.1"
	subnet := "192.168.100.0/24"

	command.PouchRun("network", "create",
		"--name", funcname,
		"-d", "bridge",
		"--gateway", gateway,
		"--subnet", subnet,
		"--option", "test=foo").Assert(c, icmd.Success)
	command.PouchRun("network", "remove", funcname)
}

// TestNetworkCreateDup tests creating duplicate network return error.
func (suite *PouchNetworkSuite) TestNetworkCreateDup(c *check.C) {
	pc, _, _, _ := runtime.Caller(0)
	tmpname := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	var funcname string
	for i := range tmpname {
		funcname = tmpname[i]
	}

	expct := icmd.Expected{
		ExitCode: 1,
		Err:      "already exist",
	}

	gateway1 := "192.168.101.1"
	subnet1 := "192.168.101.0/24"
	gateway2 := "192.168.102.1"
	subnet2 := "192.168.102.0/24"

	command.PouchRun("network", "create",
		"--name", funcname,
		"-d", "bridge",
		"--gateway", gateway1,
		"--subnet", subnet1).Assert(c, icmd.Success)

	command.PouchRun("network", "create",
		"--name", funcname,
		"-d", "bridge",
		"--gateway", gateway2,
		"--subnet", subnet2).Compare(expct)

	command.PouchRun("network", "remove", funcname)
}