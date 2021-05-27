// +build !linux

package clusterstatus

func EnableBridgeNfCallIptables() bool{
	return true
}
