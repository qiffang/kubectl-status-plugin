// +build !linux

package clusterstatus

func SelinuxEnabled() bool {
	return true
}
