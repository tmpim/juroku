package retro

import (
	"sync"

	"github.com/libretro/ludo/libretro"
)

// 	glfw.KeyX:          libretro.DeviceIDJoypadA,
// 	glfw.KeyZ:          libretro.DeviceIDJoypadB,
// 	glfw.KeyA:          libretro.DeviceIDJoypadY,
// 	glfw.KeyS:          libretro.DeviceIDJoypadX,
// 	glfw.KeyQ:          libretro.DeviceIDJoypadL,
// 	glfw.KeyW:          libretro.DeviceIDJoypadR,
// 	glfw.KeyUp:         libretro.DeviceIDJoypadUp,
// 	glfw.KeyDown:       libretro.DeviceIDJoypadDown,
// 	glfw.KeyLeft:       libretro.DeviceIDJoypadLeft,
// 	glfw.KeyRight:      libretro.DeviceIDJoypadRight,
// 	glfw.KeyEnter:      libretro.DeviceIDJoypadStart,
// 	glfw.KeyRightShift: libretro.DeviceIDJoypadSelect,

// #define RETRO_DEVICE_ID_JOYPAD_B        0
// #define RETRO_DEVICE_ID_JOYPAD_Y        1
// #define RETRO_DEVICE_ID_JOYPAD_SELECT   2
// #define RETRO_DEVICE_ID_JOYPAD_START    3
// #define RETRO_DEVICE_ID_JOYPAD_UP       4
// #define RETRO_DEVICE_ID_JOYPAD_DOWN     5
// #define RETRO_DEVICE_ID_JOYPAD_LEFT     6
// #define RETRO_DEVICE_ID_JOYPAD_RIGHT    7
// #define RETRO_DEVICE_ID_JOYPAD_A        8
// #define RETRO_DEVICE_ID_JOYPAD_X        9
// #define RETRO_DEVICE_ID_JOYPAD_L       10
// #define RETRO_DEVICE_ID_JOYPAD_R       11
// #define RETRO_DEVICE_ID_JOYPAD_L2      12
// #define RETRO_DEVICE_ID_JOYPAD_R2      13
// #define RETRO_DEVICE_ID_JOYPAD_L3      14
// #define RETRO_DEVICE_ID_JOYPAD_R3      15

type InputDevice struct {
	state   [1024]int16
	mutated bool
	mutex   *sync.Mutex
}

type input struct {
	devices [16]*InputDevice
	mutex   *sync.Mutex

	presentingState [16][1024]int16
}

func (d *InputDevice) Set(id int, value int16) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state[id] == value {
		return
	}

	d.state[id] = value
	d.mutated = true
}

func (n *input) connectDevice(id int) *InputDevice {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.devices[id] = &InputDevice{
		mutated: false,
		mutex:   new(sync.Mutex),
	}

	return n.devices[id]
}

func (n *input) getState(port uint, device uint32, index uint, id uint) int16 {
	if id >= 255 || index > 0 || device != libretro.DeviceJoypad {
		return 0
	}

	return n.presentingState[port][id]
}

func (n *input) poll() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	for port, dev := range n.devices {
		if dev == nil {
			continue
		}
		func() {
			dev.mutex.Lock()
			defer dev.mutex.Unlock()
			copy(n.presentingState[port][:], dev.state[:])
			dev.mutated = true
		}()
	}
}
