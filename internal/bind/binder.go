package bind

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/klauspost/compress/zstd"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"tailscale.com/types/key"
)

type Factory func(c echo.Context) (Binder, error)

type Binder interface {
	BindRequest(c echo.Context, v interface{}) error
	WriteResponse(c echo.Context, code int, v interface{}) error
	Marshal(compress string, v interface{}) ([]byte, error)
	Peer() key.MachinePublic
}

func DefaultBinder(machineKey key.MachinePublic) Factory {
	return func(c echo.Context) (Binder, error) {
		return &defaultBinder{machineKey: machineKey}, nil
	}
}

func BoxBinder(controlKey key.MachinePrivate) Factory {
	return func(c echo.Context) (Binder, error) {
		idParam := c.Param("id")

		id, err := util.ParseMachinePublicKey(idParam)

		if err != nil {
			return nil, err
		}

		return &boxBinder{
			controlKey: controlKey,
			machineKey: *id,
		}, nil
	}
}

type defaultBinder struct {
	machineKey key.MachinePublic
}

func (d *defaultBinder) BindRequest(c echo.Context, v interface{}) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, v)
}

func (d *defaultBinder) WriteResponse(c echo.Context, code int, v interface{}) error {
	marshalled, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.Response().WriteHeader(code)
	_, err = c.Response().Write(marshalled)

	return err
}

func (d *defaultBinder) Marshal(compress string, v interface{}) ([]byte, error) {
	var payload []byte

	marshalled, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	if compress == "zstd" {
		encoder, err := zstd.NewWriter(nil)
		if err != nil {
			return nil, err
		}

		payload = encoder.EncodeAll(marshalled, nil)
	} else {
		payload = marshalled
	}

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(len(payload)))
	data = append(data, payload...)

	return data, nil
}

func (d *defaultBinder) Peer() key.MachinePublic {
	return d.machineKey
}

type boxBinder struct {
	controlKey key.MachinePrivate
	machineKey key.MachinePublic
}

func (b *boxBinder) BindRequest(c echo.Context, v interface{}) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	decrypted, ok := b.controlKey.OpenFrom(b.machineKey, body)
	if !ok {
		return fmt.Errorf("unable to decrypt payload")
	}

	return json.Unmarshal(decrypted, v)
}

func (b *boxBinder) WriteResponse(c echo.Context, code int, v interface{}) error {
	marshalled, err := json.Marshal(v)
	if err != nil {
		return err
	}

	encrypted := b.controlKey.SealTo(b.machineKey, marshalled)

	c.Response().WriteHeader(code)
	_, err = c.Response().Write(encrypted)

	return err
}

func (b *boxBinder) Marshal(compress string, v interface{}) ([]byte, error) {
	var payload []byte

	marshalled, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	if compress == "zstd" {
		encoder, err := zstd.NewWriter(nil)
		if err != nil {
			return nil, err
		}

		encoded := encoder.EncodeAll(marshalled, nil)
		payload = b.controlKey.SealTo(b.machineKey, encoded)
	} else {
		payload = b.controlKey.SealTo(b.machineKey, marshalled)
	}

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(len(payload)))
	data = append(data, payload...)

	return data, nil
}

func (b *boxBinder) Peer() key.MachinePublic {
	return b.machineKey
}
