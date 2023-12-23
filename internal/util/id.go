package util

import (
	"fmt"
	"github.com/sony/sonyflake"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	sf     *sonyflake.Sonyflake
	sfOnce sync.Once
)

func NextID() uint64 {
	ensureProvider()
	id, _ := sf.NextID()
	return id
}

func ensureProvider() {
	sfOnce.Do(func() {
		sfInstance, err := sonyflake.New(sonyflake.Settings{
			MachineID: machineID(),
			StartTime: time.Date(2022, 05, 01, 00, 0, 0, 0, time.UTC),
		})
		if err != nil {
			panic("unable to initialize sonyflake: " + err.Error())
		}

		sf = sfInstance
	})
}

func machineID() func() (uint16, error) {
	envMachineID := os.Getenv("IONSCALE_MACHINE_ID")
	if len(envMachineID) != 0 {
		return func() (uint16, error) {
			id, err := strconv.ParseInt(envMachineID, 0, 16)
			if err != nil {
				return 0, err
			}
			return uint16(id), nil
		}
	}

	envMachineIP := os.Getenv("IONSCALE_MACHINE_IP")
	if len(envMachineIP) != 0 {
		return func() (uint16, error) {
			ip := net.ParseIP(envMachineIP).To4()
			if len(ip) < 4 {
				return 0, fmt.Errorf("invalid IP")
			}
			return uint16(ip[2])<<8 + uint16(ip[3]), nil
		}
	}

	return nil
}
