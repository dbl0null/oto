// Copyright 2019 The Oto Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !js

package oto

// #cgo LDFLAGS: -framework AudioToolbox
//
// #import <AudioToolbox/AudioToolbox.h>
//
// void oto_render(void* inUserData, AudioQueueRef inAQ, AudioQueueBufferRef inBuffer);
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

const queueBufferSize = 1024

type driver struct {
	audioQueue      C.AudioQueueRef
	buf             []byte
	bufSize         int
	channelNum      int
	bitDepthInBytes int
	buffers         []C.AudioQueueBufferRef
	m               sync.Mutex
}

var (
	theDriver *driver
	driverM   sync.Mutex
)

func setDriver(d *driver) {
	driverM.Lock()
	defer driverM.Unlock()

	if theDriver != nil && d != nil {
		panic("oto: at most one driver object can exist")
	}
	theDriver = d

	setNotificationHandler(d)
}

func getDriver() *driver {
	driverM.Lock()
	defer driverM.Unlock()

	return theDriver
}

// TOOD: Convert the error code correctly.
// See https://stackoverflow.com/questions/2196869/how-do-you-convert-an-iphone-osstatus-code-to-something-useful

func newDriver(sampleRate, channelNum, bitDepthInBytes, bufferSizeInBytes int) (tryWriteCloser, error) {
	flags := C.kAudioFormatFlagIsPacked
	if bitDepthInBytes != 1 {
		flags |= C.kAudioFormatFlagIsSignedInteger
	}
	desc := C.AudioStreamBasicDescription{
		mSampleRate:       C.double(sampleRate),
		mFormatID:         C.kAudioFormatLinearPCM,
		mFormatFlags:      C.UInt32(flags),
		mBytesPerPacket:   C.UInt32(channelNum * bitDepthInBytes),
		mFramesPerPacket:  1,
		mBytesPerFrame:    C.UInt32(channelNum * bitDepthInBytes),
		mChannelsPerFrame: C.UInt32(channelNum),
		mBitsPerChannel:   C.UInt32(8 * bitDepthInBytes),
	}
	var audioQueue C.AudioQueueRef
	if osstatus := C.AudioQueueNewOutput(
		&desc,
		(C.AudioQueueOutputCallback)(C.oto_render),
		nil,
		(C.CFRunLoopRef)(0),
		(C.CFStringRef)(0),
		0,
		&audioQueue); osstatus != C.noErr {
		return nil, fmt.Errorf("oto: AudioQueueNewFormat with StreamFormat failed: %d", osstatus)
	}

	d := &driver{
		audioQueue:      audioQueue,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
		bufSize:         bufferSizeInBytes / queueBufferSize * queueBufferSize,
		buffers:         make([]C.AudioQueueBufferRef, bufferSizeInBytes / queueBufferSize),
	}
	runtime.SetFinalizer(d, (*driver).Close)
	// Set the driver before setting the rendering callback.
	setDriver(d)

	for i := 0; i < len(d.buffers); i++ {
		C.AudioQueueAllocateBuffer(audioQueue, C.UInt32(queueBufferSize), &d.buffers[i])
		d.buffers[i].mAudioDataByteSize = C.UInt32(queueBufferSize)
		for j := 0; j < queueBufferSize; j++ {
			*(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(d.buffers[i].mAudioData)) + uintptr(j))) = 0
		}
		C.AudioQueueEnqueueBuffer(audioQueue, d.buffers[i], 0, nil)
	}

	C.AudioQueueStart(audioQueue, nil)

	return d, nil
}

//export oto_render
func oto_render(inUserData unsafe.Pointer, inAQ C.AudioQueueRef, inBuffer C.AudioQueueBufferRef) {
	d := getDriver()
	d.m.Lock()
	defer d.m.Unlock()

	n := queueBufferSize
	if n > len(d.buf) {
		n = len(d.buf)
	}

	for i := 0; i < n; i++ {
		*(*byte)(unsafe.Pointer(uintptr(inBuffer.mAudioData) + uintptr(i))) = d.buf[i]
	}
	for i := n; i < queueBufferSize; i++ {
		// TODO: Is there a better way to surppress choppy sound?
		*(*byte)(unsafe.Pointer(uintptr(inBuffer.mAudioData) + uintptr(i))) = 0
	}
	d.buf = d.buf[n:]
	// Do not update mAudioDataByteSize, or the buffer is not used any more.

	C.AudioQueueEnqueueBuffer(inAQ, inBuffer, 0, nil)
}

func (d *driver) TryWrite(data []byte) (int, error) {
	d.m.Lock()
	defer d.m.Unlock()

	n := d.bufSize - len(d.buf)
	if n > len(data) {
		n = len(data)
	}
	d.buf = append(d.buf, data[:n]...)
	return n, nil
}

func (d *driver) Close() error {
	d.m.Lock()
	defer d.m.Unlock()

	runtime.SetFinalizer(d, nil)

	for _, b := range d.buffers {
		if osstatus := C.AudioQueueFreeBuffer(d.audioQueue, b); osstatus != C.noErr {
			return fmt.Errorf("oto: AudioQueueFreeBuffer failed: %d", osstatus)
		}
	}

	if osstatus := C.AudioQueueStop(d.audioQueue, C.false); osstatus != C.noErr {
		return fmt.Errorf("oto: AudioQueueStop failed: %d", osstatus)
	}
	if osstatus := C.AudioQueueDispose(d.audioQueue, C.false); osstatus != C.noErr {
		return fmt.Errorf("oto: AudioQueueDispose failed: %d", osstatus)
	}
	d.audioQueue = nil
	setDriver(nil)
	return nil
}
