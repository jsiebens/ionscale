package util

import (
	"strings"
	"tailscale.com/types/key"
)

const (
	discoPublicHexPrefix   = "discokey:"
	nodePublicHexPrefix    = "nodekey:"
	machinePublicHexPrefix = "mkey:"
	privateHexPrefix       = "privkey:"
)

func ParseMachinePrivateKey(machineKey string) (*key.MachinePrivate, error) {
	if !strings.HasPrefix(machineKey, privateHexPrefix) {
		machineKey = privateHexPrefix + machineKey
	}

	var mp key.MachinePrivate

	if err := mp.UnmarshalText([]byte(machineKey)); err != nil {
		return nil, err
	}

	return &mp, nil
}

func ParseMachinePublicKey(machineKey string) (*key.MachinePublic, error) {
	if !strings.HasPrefix(machineKey, machinePublicHexPrefix) {
		machineKey = machinePublicHexPrefix + machineKey
	}

	var mp key.MachinePublic

	if err := mp.UnmarshalText([]byte(machineKey)); err != nil {
		return nil, err
	}

	return &mp, nil
}

func ParseNodePublicKey(machineKey string) (*key.NodePublic, error) {
	if !strings.HasPrefix(machineKey, nodePublicHexPrefix) {
		machineKey = nodePublicHexPrefix + machineKey
	}

	var mp key.NodePublic

	if err := mp.UnmarshalText([]byte(machineKey)); err != nil {
		return nil, err
	}

	return &mp, nil
}

func ParseDiscoPublicKey(machineKey string) (*key.DiscoPublic, error) {
	if !strings.HasPrefix(machineKey, discoPublicHexPrefix) {
		machineKey = discoPublicHexPrefix + machineKey
	}

	var mp key.DiscoPublic

	if err := mp.UnmarshalText([]byte(machineKey)); err != nil {
		return nil, err
	}

	return &mp, nil
}
