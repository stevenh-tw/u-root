// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race
// +build !race

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hugelgupf/vmtest"
	"github.com/hugelgupf/vmtest/qemu"
	"github.com/hugelgupf/vmtest/testtmp"
	"github.com/u-root/u-root/pkg/uroot"
)

// TestMountKexec tests that kexec occurs correctly by checking the kernel cmdline.
// This is possible because the generic initramfs ensures that we mount the
// testdata directory containing the initramfs and kernel used in the VM.
func TestMountKexec(t *testing.T) {
	vmtest.SkipIfNotArch(t, qemu.ArchAMD64, qemu.ArchArm64)

	testCmds := []string{
		"var CMDLINE = (cat /proc/cmdline)",
		"var SUFFIX = $CMDLINE[-7..]",
		"echo SAW $SUFFIX",
		"kexec -i /testdata/initramfs.cpio -c $CMDLINE' KEXEC=Y' /kernel",
	}

	vm := vmtest.StartVMAndRunCmds(t, testCmds,
		vmtest.WithMergedInitramfs(uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/core/kexec",
				"github.com/u-root/u-root/cmds/core/shutdown",
			),
			ExtraFiles: []string{
				fmt.Sprintf("%s:kernel", os.Getenv("VMTEST_KERNEL")),
			},
		}),
		vmtest.WithQEMUFn(
			qemu.WithVMTimeout(time.Minute),
			qemu.ArbitraryArgs("-m", "8192"),
		),
		// The initramfs will be placed in shared dir, so in the VM
		// it's available at /testdata/initramfs.cpio.
		vmtest.WithSharedDir(testtmp.TempDir(t)),
	)

	if _, err := vm.Console.ExpectString("SAW KEXEC=Y"); err != nil {
		t.Fatal(err)
	}
	if err := vm.Kill(); err != nil {
		t.Errorf("Kill: %v", err)
	}
	_ = vm.Wait()
}

// TestMountKexecLoad is same as TestMountKexec except it test calling
// kexec_load syscall than file load.
func TestMountKexecLoad(t *testing.T) {
	vmtest.SkipIfNotArch(t, qemu.ArchAMD64, qemu.ArchArm64)

	gzipP, err := exec.LookPath("gzip")
	if err != nil {
		t.Skipf("no gzip found, skip it as it won't be able to de-compress kernel")
	}

	testCmds := []string{
		"var CMDLINE = (cat /proc/cmdline)",
		"var SUFFIX = $CMDLINE[-7..]",
		"echo SAW $SUFFIX",
		"kexec -d -i /testdata/initramfs.cpio --loadsyscall -c $CMDLINE' KEXEC=Y' /kernel",
	}
	vm := vmtest.StartVMAndRunCmds(t, testCmds,
		vmtest.WithMergedInitramfs(uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/core/kexec",
				"github.com/u-root/u-root/cmds/core/shutdown",
			),
			ExtraFiles: []string{
				fmt.Sprintf("%s:kernel", os.Getenv("VMTEST_KERNEL")),
				gzipP,
			},
		}),
		vmtest.WithQEMUFn(
			qemu.WithVMTimeout(time.Minute),
			qemu.ArbitraryArgs("-m", "8192"),
		),
		// The initramfs will be placed in shared dir, so in the VM
		// it's available at /testdata/initramfs.cpio.
		vmtest.WithSharedDir(testtmp.TempDir(t)),
	)

	if _, err := vm.Console.ExpectString("SAW KEXEC=Y"); err != nil {
		t.Error(err)
	}
	if err := vm.Kill(); err != nil {
		t.Errorf("Kill: %v", err)
	}
	_ = vm.Wait()
}

// TestMountKexecLoadOnly test kexec loads without a kexec reboot.
func TestMountKexecLoadOnly(t *testing.T) {
	vmtest.SkipIfNotArch(t, qemu.ArchAMD64, qemu.ArchArm64)

	gzipP, err := exec.LookPath("gzip")
	if err != nil {
		t.Skipf("no gzip found, skip it as it won't be able to de-compress kernel")
	}

	testCmds := []string{
		"var CMDLINE = (cat /proc/cmdline)",
		"echo kexecloadresult ?(kexec -d -l -i /testdata/initramfs.cpio --loadsyscall -c $CMDLINE /kernel)",
	}
	vm := vmtest.StartVMAndRunCmds(t, testCmds,
		vmtest.WithMergedInitramfs(uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/core/kexec",
			),
			ExtraFiles: []string{
				fmt.Sprintf("%s:kernel", os.Getenv("VMTEST_KERNEL")),
				gzipP,
			},
		}),
		vmtest.WithQEMUFn(
			qemu.WithVMTimeout(time.Minute),
			qemu.ArbitraryArgs("-m", "8192"),
		),
		// The initramfs will be placed in shared dir, so in the VM
		// it's available at /testdata/initramfs.cpio.
		vmtest.WithSharedDir(testtmp.TempDir(t)),
	)

	if _, err := vm.Console.ExpectString("kexecloadresult $ok"); err != nil {
		t.Error(err)
	}
	if err := vm.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
}

// TestMountKexecLoadCustomDTB test kexec_load a Arm64 Image with a user provided dtb.
func TestMountKexecLoadCustomDTB(t *testing.T) {
	vmtest.SkipIfNotArch(t, qemu.ArchArm64)

	testCmds := []string{
		"var CMDLINE = (cat /proc/cmdline)",
		"var SUFFIX = $CMDLINE[-7..]",
		"echo SAW $SUFFIX",
		"cp /sys/firmware/fdt /tmp/userfdt",
		"kexec -d --dtb /tmp/userfdt -i /testdata/initramfs.cpio --loadsyscall -c $CMDLINE' KEXEC=Y' /kernel",
	}
	vm := vmtest.StartVMAndRunCmds(t, testCmds,
		vmtest.WithMergedInitramfs(uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/core/cp",
				"github.com/u-root/u-root/cmds/core/kexec",
			),
			ExtraFiles: []string{
				fmt.Sprintf("%s:kernel", os.Getenv("VMTEST_KERNEL")),
			},
		}),
		vmtest.WithQEMUFn(
			qemu.WithVMTimeout(time.Minute),
			qemu.ArbitraryArgs("-m", "8192"),
		),
		// The initramfs will be placed in shared dir, so in the VM
		// it's available at /testdata/initramfs.cpio.
		vmtest.WithSharedDir(testtmp.TempDir(t)),
	)

	if _, err := vm.Console.ExpectString("SAW KEXEC=Y"); err != nil {
		t.Error(err)
	}
	if err := vm.Kill(); err != nil {
		t.Errorf("Kill: %v", err)
	}
	_ = vm.Wait()
}

func TestKexecLinuxImageCfgFile(t *testing.T) {
	vmtest.SkipIfNotArch(t, qemu.ArchAMD64, qemu.ArchArm64)

	dir := t.TempDir()
	cfg := []byte("{ \"InitrdPath\": \"/testdata/initramfs.cpio\", \"KernelPath\": \"/kernel\", \"Cmdline\": \"/proc/cmdline\", \"Name\": \"testloadconfig\" }")
	cfgFile := filepath.Join(dir, "linux_image_cfg.json")
	if err := os.WriteFile(cfgFile, cfg, 0777); err != nil {
		t.Fatalf("Failed to setup test cfg file: %v", err)
	}

	testCmds := []string{
		"echo kexecloadresult ?(kexec -d -l -I /linux_image_cfg.json)",
	}
	vm := vmtest.StartVMAndRunCmds(t, testCmds,
		vmtest.WithMergedInitramfs(uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/core/echo",
				"github.com/u-root/u-root/cmds/core/kexec",
			),
			ExtraFiles: []string{
				fmt.Sprintf("%s:kernel", os.Getenv("VMTEST_KERNEL")),
				fmt.Sprintf("%s:linux_image_cfg.json", cfgFile),
			},
		}),
		vmtest.WithQEMUFn(
			qemu.WithVMTimeout(time.Minute),
			qemu.ArbitraryArgs("-m", "8192"),
		),
		// The initramfs will be placed in shared dir, so in the VM
		// it's available at /testdata/initramfs.cpio.
		vmtest.WithSharedDir(testtmp.TempDir(t)),
	)

	if _, err := vm.Console.ExpectString("kexecloadresult $ok"); err != nil {
		t.Error(err)
	}
	if err := vm.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
}
