package retro

import (
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"github.com/libretro/ludo/libretro"
)

type Environment struct {
	gameInfo        libretro.GameInfo
	pixelFormat     uint32
	avInfo          libretro.SystemAVInfo
	variablesUpdate bool
}

type RetroCore struct {
	core         *libretro.Core
	opts         *Options
	env          *Environment
	input        *input
	execQueue    chan func(core *RetroCore)
	drawCallback func(img image.Image)

	audioCallback func(data []int16)
	audioSamples  int
	audioBuffer   []int16

	running bool
}

type Options struct {
	Username  string
	SystemDir string
	SaveDir   string
	Variables map[string]string
}

func NewCore(sofile string, opts *Options) (*RetroCore, error) {
	core, err := libretro.Load(sofile)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(opts.SaveDir, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("juroku retro: failed to create SaveDir %q: %w", opts.SaveDir, err)
	}

	err = os.MkdirAll(opts.SystemDir, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("juroku retro: failed to create SystemDir %q: %w", opts.SystemDir, err)
	}

	retroCore := &RetroCore{
		core: core,
		opts: opts,
		env: &Environment{
			variablesUpdate: false,
		},
		input: &input{
			mutex: new(sync.Mutex),
		},
		execQueue: make(chan func(core *RetroCore), 10),
	}

	retroCore.initCore()

	return retroCore, nil
}

func (c *RetroCore) Run() {
	c.running = true
	tick := time.Duration(float64(time.Second) / c.env.avInfo.Timing.FPS)
	t := time.NewTicker(tick)
	for {
		select {
		case <-t.C:
			if !c.running {
				return
			}
			c.core.Run()
		case op := <-c.execQueue:
			op(c)
			if !c.running {
				return
			}
		}
	}
}

func (c *RetroCore) AVInfo() libretro.SystemAVInfo {
	result := make(chan libretro.SystemAVInfo)
	c.execQueue <- func(core *RetroCore) {
		result <- core.env.avInfo
	}
	return <-result
}

func (c *RetroCore) OnFrameDraw(drawFunc func(img image.Image)) {
	c.drawCallback = drawFunc
}

func (c *RetroCore) OnAudioBuffer(bufferAmount int, audioFunc func(data []int16)) {
	c.audioSamples = bufferAmount
	c.audioCallback = audioFunc
}

func (c *RetroCore) ConnectDevice(id int) *InputDevice {
	c.execQueue <- func(core *RetroCore) {
		core.core.SetControllerPortDevice(uint(id), libretro.DeviceJoypad)
	}

	return c.input.connectDevice(id)
}

func (c *RetroCore) Save(name string) error {
	var err error
	var data []byte
	waitChan := make(chan struct{})

	c.execQueue <- func(core *RetroCore) {
		data, err = c.core.Serialize(c.core.SerializeSize())
		close(waitChan)
	}

	<-waitChan

	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(c.opts.SaveDir, name), data, 0644)
}

func (c *RetroCore) Load(name string) error {
	data, err := ioutil.ReadFile(filepath.Join(c.opts.SaveDir, name))
	if err != nil {
		return err
	}

	waitChan := make(chan error)

	c.execQueue <- func(core *RetroCore) {
		waitChan <- c.core.Unserialize(data, uint(len(data)))
	}

	return <-waitChan
}

func (c *RetroCore) initCore() {
	c.core.SetEnvironment(c.environment)

	c.core.SetInputPoll(c.input.poll)
	c.core.SetInputState(c.input.getState)

	c.core.SetVideoRefresh(c.refresh)

	c.core.SetAudioSample(func(left, right int16) {
		c.audioBuffer = append(c.audioBuffer, []int16{left, right}...)

		if len(c.audioBuffer) >= c.audioSamples*2 {
			c.audioCallback(c.audioBuffer)
			c.audioBuffer = nil
		}
	})
	c.core.SetAudioSampleBatch(func(data []byte, size int32) int32 {
		toAppend := make([]int16, len(data)/2)

		for i := 0; i < len(data); i += 2 {
			toAppend[i/2] = int16(data[i]) | int16(data[i+1])<<8
		}

		c.audioBuffer = append(c.audioBuffer, toAppend...)

		if len(c.audioBuffer) >= c.audioSamples*2 {
			c.audioCallback(c.audioBuffer)
			c.audioBuffer = nil
		}

		return size
	})

	c.core.Init()

	sys := c.core.GetSystemInfo()
	if sys.LibraryName != "" {
		log.Printf("juroku retro: core initialised: %+v\n", sys)
	}
}

func (c *RetroCore) Stop() {
	c.execQueue <- func(core *RetroCore) {
		core.core.UnloadGame()
		core.core.Deinit()
		core.running = false
	}
}

func (c *RetroCore) LoadGame(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	c.env.gameInfo = libretro.GameInfo{
		Path: path,
		Size: int64(len(data)),
	}

	c.env.gameInfo.SetData(data)

	if !c.core.LoadGame(c.env.gameInfo) {
		return errors.New("juroku retro: failed to load game")
	}

	c.env.avInfo = c.core.GetSystemAVInfo()

	return nil
}

func (c *RetroCore) environment(cmd uint32, data unsafe.Pointer) bool {
	switch cmd {
	case libretro.EnvironmentSetRotation:
		log.Println("juroku retro: environment: ignoring rotation")
	case libretro.EnvironmentGetUsername:
		libretro.SetString(data, c.opts.Username)
	case libretro.EnvironmentGetLogInterface:
		c.core.BindLogCallback(data, func(level uint32, msg string) {
			// log.Println("juroku retro: core log:", strings.TrimSpace(msg))
		})
	case libretro.EnvironmentGetPerfInterface:
		c.core.BindPerfCallback(data, func() int64 {
			return time.Now().UnixNano() / 1000
		})
	case libretro.EnvironmentSetFrameTimeCallback:
		c.core.SetFrameTimeCallback(data)
	case libretro.EnvironmentSetAudioCallback:
		c.core.SetAudioCallback(data)
	case libretro.EnvironmentGetCanDupe:
		libretro.SetBool(data, true)
	case libretro.EnvironmentSetPixelFormat:
		c.env.pixelFormat = libretro.GetPixelFormat(data)
	case libretro.EnvironmentGetSystemDirectory:
		libretro.SetString(data, c.opts.SystemDir)
	case libretro.EnvironmentGetSaveDirectory:
		libretro.SetString(data, c.opts.SaveDir)
	case libretro.EnvironmentShutdown:
		c.Stop()
	case libretro.EnvironmentGetVariable:
		variable := libretro.GetVariable(data)
		val, found := c.opts.Variables[variable.Key()]
		if !found {
			return false
		}
		variable.SetValue(val)
	case libretro.EnvironmentSetVariables:
		log.Println("juroku retro: available environment variables:")
		variables := libretro.GetVariables(data)
		for _, v := range variables {
			log.Printf("key: %v, desc: %v, possible values: %+v\n", v.Key(),
				v.Desc(), v.Choices())
		}
	case libretro.EnvironmentGetVariableUpdate:
		libretro.SetBool(data, c.env.variablesUpdate)
		c.env.variablesUpdate = false
	case libretro.EnvironmentSetGeometry:
		c.env.avInfo.Geometry = libretro.GetGeometry(data)
	case libretro.EnvironmentSetSystemAVInfo:
		c.env.avInfo = libretro.GetSystemAVInfo(data)
	default:
		log.Println("juroku retro: host environment command not implemented:", cmd)
		return false
	}
	return true
}
