// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"testing"
	"time"

	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/vmtest"
)

func TestDhclient(t *testing.T) {
	// TODO: support arm
	if vmtest.TestArch() != "amd64" {
		t.Skipf("test not supported on %s", vmtest.TestArch())
	}

	network := qemu.NewNetwork()
	_, scleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name:         "TestDhclient_Server",
		SerialOutput: vmtest.TestLineWriter(t, "server"),
		Cmds: []string{
			"github.com/u-root/u-root/cmds/core/echo",
			"github.com/u-root/u-root/cmds/core/ip",
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-root/cmds/core/sleep",
			"github.com/u-root/u-root/cmds/core/shutdown",
			"github.com/u-root/u-root/integration/testcmd/pxeserver",
		},
		Uinit: []string{
			"ip link set eth0 up",
			"ip addr add 192.168.0.1/24 dev eth0",
			"ip route add 0.0.0.0/0 dev eth0",
			"pxeserver",
		},
		Network: network,
	})
	defer scleanup()

	dhcpClient, ccleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name:         "TestDhclient_Client",
		SerialOutput: vmtest.TestLineWriter(t, "client"),
		Cmds: []string{
			"github.com/u-root/u-root/cmds/core/ip",
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-root/cmds/core/dhclient",
			"github.com/u-root/u-root/cmds/core/shutdown",
		},
		Uinit: []string{
			"dhclient -ipv6=false -v",
			"ip a",
			"sleep 5",
			"shutdown -h",
		},
		Network: network,
		Timeout: 30 * time.Second,
	})
	defer ccleanup()

	if err := dhcpClient.Expect("Configured eth0 with IPv4 DHCP Lease"); err != nil {
		t.Error(err)
	}
	if err := dhcpClient.Expect("inet 192.168.0.2"); err != nil {
		t.Error(err)
	}
}

func TestPxeboot(t *testing.T) {
	// TODO: support arm
	if vmtest.TestArch() != "amd64" {
		t.Skipf("test not supported on %s", vmtest.TestArch())
	}

	network := qemu.NewNetwork()
	dhcpServer, scleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name:         "TestPxeboot_Server",
		SerialOutput: vmtest.TestLineWriter(t, "server"),
		Cmds: []string{
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-root/cmds/core/ip",
			"github.com/u-root/u-root/cmds/core/ls",
			"github.com/u-root/u-root/integration/testcmd/pxeserver",
		},
		Uinit: []string{
			"ip addr add 192.168.0.1/24 dev eth0",
			"ip link set eth0 up",
			"ip route add 0.0.0.0/0 dev eth0",
			"ls -l /pxeroot",
			"pxeserver -dir=/pxeroot",
		},
		Files: []string{
			"./testdata/pxe:pxeroot",
		},
		Network: network,
		Timeout: 15 * time.Second,
	})
	defer scleanup()

	dhcpClient, ccleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name:         "TestPxeboot_Client",
		SerialOutput: vmtest.TestLineWriter(t, "client"),
		Cmds: []string{
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-root/cmds/core/ip",
			"github.com/u-root/u-root/cmds/core/shutdown",
			"github.com/u-root/u-root/cmds/boot/pxeboot",
		},
		Uinit: []string{
			"pxeboot --dry-run --no-load -v",
			"shutdown -h",
		},
		Network: network,
		Timeout: 15 * time.Second,
	})
	defer ccleanup()

	if err := dhcpServer.Expect("starting file server"); err != nil {
		t.Error(err)
	}
	if err := dhcpClient.Expect("Got DHCPv4 lease on eth0:"); err != nil {
		t.Error(err)
	}
	if err := dhcpClient.Expect("Boot URI: tftp://192.168.0.1/pxelinux.0"); err != nil {
		t.Error(err)
	}
}
