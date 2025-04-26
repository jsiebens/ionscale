package util

import (
	"errors"
	"fmt"
	"github.com/sony/sonyflake"
	"go.uber.org/zap"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	sf     provider
	sfOnce sync.Once
)

func NextID() uint64 {
	EnsureIDProvider()
	id, err := sf.NextID()
	if err != nil {
		panic(err)
	}
	return id
}

type provider interface {
	NextID() (uint64, error)
}

type errorProvider struct {
	err error
}

func (e errorProvider) NextID() (uint64, error) {
	return 0, fmt.Errorf("unable to generate ID, sonyflake not configured properly: %w", e.err)
}

func EnsureIDProvider() {
	sfOnce.Do(func() {
		sf = createIDProvider()
	})
}

func createIDProvider() provider {
	startTime := time.Date(2022, 05, 01, 00, 0, 0, 0, time.UTC)
	
	sfInstance, err := sonyflake.New(sonyflake.Settings{
		MachineID: machineID(),
		StartTime: startTime,
	})

	if err != nil && errors.Is(err, sonyflake.ErrNoPrivateAddress) {
		id := RandUint16()
		zap.L().Warn("failed to generate sonyflake machine id from private ip address, using a random machine id", zap.Uint16("id", id))

		sfInstance, err = sonyflake.New(sonyflake.Settings{
			MachineID: func() (uint16, error) { return id, nil },
			StartTime: startTime,
		})

		if err != nil {
			return errorProvider{err}
		}
	}

	if err != nil {
		return errorProvider{err}
	}

	return sfInstance
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
