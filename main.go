package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		fmt.Printf("LOG: %v\n", message)
	})
	if err != nil {
		fmt.Println("Failed to initialize context:", err)
		return
	}
	defer ctx.Uninit()

	// List all capture devices
	fmt.Println("Available capture devices:")
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		fmt.Println("Failed to retrieve device infos:", err)
		return
	}
	for i, info := range infos {
		fmt.Printf("%d: %v\n", i, info.Name())
	}

	// Select a capture device - this example selects the first capture device
	if len(infos) == 0 {
		fmt.Println("No capture devices found.")
		return
	}
	selectedDeviceID := infos[3].ID

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 44100
	deviceConfig.Capture.DeviceID = selectedDeviceID.Pointer()
	deviceConfig.Alsa.NoMMap = 1

	// file, err := os.Create("recording.raw")
	// if err != nil {
	// 	fmt.Println("Failed to create file:", err)
	// 	return
	// }
	// defer file.Close()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s, type: %s", message, websocket.FormatMessageType(mt))
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	onRecvFrames := func(pSample2, pSample []byte, framecount uint32) {
		// Here you can process the raw audio data
		// fmt.Println("Received frames:", len(pSample))
		c.WriteMessage(websocket.BinaryMessage, pSample)
		// file.Write(pSample)
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		fmt.Println("Failed to initialize device:", err)
		return
	}
	defer device.Uninit()

	fmt.Println("Recording... Press Ctrl+C to stop.")
	err = device.Start()
	if err != nil {
		fmt.Println("Failed to start device:", err)
		return
	}

	for range interrupt {
		log.Println("interrupt")
		device.Stop()
		break
	}

	fmt.Println("stopped.")
}
