// +build !linux

package clusterstatus

func SwapDisabled() (bool, error) {
	return false, nil
}