package clusterstatus

import (
	"fmt"
	"github.com/moby/sys/mountinfo"
	"syscall"
	"os"
)

const (
	stRdOnly         = 0x01
)

func SelinuxEnabled() bool {
	selinuxEnabled := false
	if fs := getSelinuxMountPoint(); fs != "" {
		if con, _ := Getcon(); con != "kernel" {
			selinuxEnabled = true
		}
	}
	return selinuxEnabled
}

func readCon(name string) (string, error) {
	var val string

	in, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer in.Close()

	_, err = fmt.Fscanf(in, "%s", &val)
	return val, err
}

// Getcon returns the SELinux label of the current process thread, or an error.
func Getcon() (string, error) {
	return readCon(fmt.Sprintf("/proc/self/task/%d/attr/current", syscall.Gettid()))
}

// getSelinuxMountPoint returns the path to the mountpoint of an selinuxfs
// filesystem or an empty string if no mountpoint is found.  Selinuxfs is
// a proc-like pseudo-filesystem that exposes the selinux policy API to
// processes.  The existence of an selinuxfs mount is used to determine
// whether selinux is currently enabled or not.
func getSelinuxMountPoint() string {
	selinuxfs := ""

	mounts, err := mountinfo.GetMounts(nil)
	if err != nil {
		return selinuxfs
	}
	for _, mount := range mounts {
		if mount.FSType == "selinuxfs" {
			selinuxfs = mount.Mountpoint
			break
		}
	}
	if selinuxfs != "" {
		var buf syscall.Statfs_t
		syscall.Statfs(selinuxfs, &buf)
		if (buf.Flags & stRdOnly) == 1 {
			selinuxfs = ""
		}
	}
	return selinuxfs
}