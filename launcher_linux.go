//go:build linux

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

func watchButtons() error {
	const device = "/dev/input/by-path/platform-30370000.snvs:snvs-powerkey-event"
	var busy atomic.Bool
	for {
		input, err := os.Open(device)
		if err != nil {
			log.Print("Power shortcut unavailable: ", err)
			time.Sleep(5 * time.Second)
			continue
		}
		var presses pressSequence
		log.Print("Chess shortcut ready: triple-press power")
		event := make([]byte, 16)
		for {
			if _, err = io.ReadFull(input, event); err != nil {
				break
			}
			typ, code, value := binary.LittleEndian.Uint16(event[8:10]), binary.LittleEndian.Uint16(event[10:12]), int32(binary.LittleEndian.Uint32(event[12:16]))
			if typ == 1 && code == 116 && value == 1 && presses.add(time.Now()) && !busy.Swap(true) {
				go func() {
					defer busy.Store(false)
					log.Print("Chess open requested")
					output, e := exec.Command("/home/root/paper-chess/session", "start").CombinedOutput()
					if e != nil {
						log.Printf("Chess launch failed: %v %s", e, output)
					} else {
						log.Print("Chess shortcut completed")
					}
				}()
			}
		}
		input.Close()
		log.Print("Reopening power button stream: ", err)
		time.Sleep(time.Second)
	}
}
func monotonicMicros() (int64, error) {
	var value syscall.Timespec
	_, _, err := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, 1, uintptr(unsafe.Pointer(&value)), 0)
	if err != 0 {
		return 0, err
	}
	return int64(value.Sec)*1000000 + int64(value.Nsec)/1000, nil
}
func diagnosticTap(x, y int) error {
	if x < 0 || x > 1403 || y < 0 || y > 1871 {
		return fmt.Errorf("tap outside portrait screen")
	}
	// Native touchscreen input is injected only by explicit root diagnostics.
	f, e := os.OpenFile("/dev/input/event2", os.O_WRONLY, 0)
	if e != nil {
		return e
	}
	defer f.Close()
	for _, code := range []uint{53, 54} {
		var info [6]int32
		_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(0x80184540+code), uintptr(unsafe.Pointer(&info[0])))
		if e != 0 {
			return e
		}
		fmt.Printf("Touch axis %d range %d..%d\n", code, info[1], info[2])
	}
	send := func(typ, code uint16, value int32) error {
		event := make([]byte, 16)
		binary.LittleEndian.PutUint16(event[8:10], typ)
		binary.LittleEndian.PutUint16(event[10:12], code)
		binary.LittleEndian.PutUint32(event[12:16], uint32(value))
		_, e := f.Write(event)
		return e
	}
	for _, v := range [][3]int{{3, 47, 0}, {3, 57, 240}, {3, 53, x}, {3, 54, y}, {3, 58, 50}, {1, 330, 1}, {0, 0, 0}} {
		if e = send(uint16(v[0]), uint16(v[1]), int32(v[2])); e != nil {
			return e
		}
	}
	time.Sleep(80 * time.Millisecond)
	for _, v := range [][3]int{{3, 47, 0}, {3, 57, -1}, {1, 330, 0}, {0, 0, 0}} {
		if e = send(uint16(v[0]), uint16(v[1]), int32(v[2])); e != nil {
			return e
		}
	}
	return nil
}
