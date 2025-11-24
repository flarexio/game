package moonlight

/*
#cgo CFLAGS:  -I../moonlight-common-c/src -I. -Wno-dll-attribute-on-redeclaration
#cgo LDFLAGS: -L../moonlight-common-c/build -lmoonlight-common-c -Wl,--allow-multiple-definition
#include <stdlib.h>
#include <Limelight.h>
#include <Windows.h>
#include "callback.h"
*/
import "C"
import (
	"unsafe"
)

var (
	connectionListener ConnectionListener
	videoRenderer      VideoDecoderRenderer
	audioRenderer      AudioRenderer
)

var (
	clCallbacks *C.CONNECTION_LISTENER_CALLBACKS
	drCallbacks *C.DECODER_RENDERER_CALLBACKS
	arCallbacks *C.AUDIO_RENDERER_CALLBACKS
)

func SetupCallbacks(cl ConnectionListener, vr VideoDecoderRenderer, ar AudioRenderer) {
	connectionListener = cl
	videoRenderer = vr
	audioRenderer = ar

	clCallbacks = (*C.CONNECTION_LISTENER_CALLBACKS)(
		C.malloc(C.size_t(unsafe.Sizeof(C.CONNECTION_LISTENER_CALLBACKS{}))),
	)
	C.LiInitializeConnectionCallbacks(clCallbacks)

	drCallbacks = (*C.DECODER_RENDERER_CALLBACKS)(
		C.malloc(C.size_t(unsafe.Sizeof(C.DECODER_RENDERER_CALLBACKS{}))),
	)
	C.LiInitializeVideoCallbacks(drCallbacks)

	arCallbacks = (*C.AUDIO_RENDERER_CALLBACKS)(
		C.malloc(C.size_t(unsafe.Sizeof(C.AUDIO_RENDERER_CALLBACKS{}))),
	)
	C.LiInitializeAudioCallbacks(arCallbacks)

	C.setupCallbacks(clCallbacks, drCallbacks, arCallbacks)
}

type ConnectionListener interface {
	StageStarting(stage int)
	StageComplete(stage int)
	StageFailed(stage int, errorCode int)
	ConnectionStarted()
	ConnectionTerminated(errorCode int)
	LogMessage(format string, args ...any)
	Rumble(controllerNumber, lowFreqMotor, highFreqMotor uint16)
	ConnectionStatusUpdate(connectionStatus int)
	SetHDRMode(hdrEnabled bool)
	RumbleTriggers(controllerNumber, leftTriggerMotor, rightTriggerMotor uint16)
	SetMotionEventState(controllerNumber uint16, motionType uint8, reportRateHz uint16)
	SetControllerLED(controllerNumber uint16, r, g, b uint8)
}

//export goClStageStarting
func goClStageStarting(stage C.int) {
	connectionListener.StageStarting(int(stage))
}

//export goClStageComplete
func goClStageComplete(stage C.int) {
	connectionListener.StageComplete(int(stage))
}

//export goClStageFailed
func goClStageFailed(stage C.int, errorCode C.int) {
	connectionListener.StageFailed(int(stage), int(errorCode))
}

//export goClConnectionStarted
func goClConnectionStarted() {
	connectionListener.ConnectionStarted()
}

//export goClConnectionTerminated
func goClConnectionTerminated(errorCode C.int) {
	connectionListener.ConnectionTerminated(int(errorCode))
}

//export goClLogMessageImpl
func goClLogMessageImpl(message *C.char) {
	goMessage := C.GoString(message)
	connectionListener.LogMessage("%s", goMessage)
}

//export goClRumble
func goClRumble(controllerNumber C.uint16_t, lowFreqMotor C.uint16_t, highFreqMotor C.uint16_t) {
	connectionListener.Rumble(uint16(controllerNumber), uint16(lowFreqMotor), uint16(highFreqMotor))
}

//export goClConnectionStatusUpdate
func goClConnectionStatusUpdate(connectionStatus C.int) {
	connectionListener.ConnectionStatusUpdate(int(connectionStatus))
}

//export goClSetHDRMode
func goClSetHDRMode(hdrEnabled C.bool) {
	// TODO: if enabled && C.LiGetHdrMetadata()
	connectionListener.SetHDRMode(bool(hdrEnabled))
}

//export goClRumbleTriggers
func goClRumbleTriggers(controllerNumber C.uint16_t, leftTriggerMotor C.uint16_t, rightTriggerMotor C.uint16_t) {
	connectionListener.RumbleTriggers(uint16(controllerNumber), uint16(leftTriggerMotor), uint16(rightTriggerMotor))
}

//export goClSetMotionEventState
func goClSetMotionEventState(controllerNumber C.uint16_t, motionType C.uint8_t, reportRateHz C.uint16_t) {
	connectionListener.SetMotionEventState(uint16(controllerNumber), uint8(motionType), uint16(reportRateHz))
}

//export goClSetControllerLED
func goClSetControllerLED(controllerNumber C.uint16_t, r C.uint8_t, g C.uint8_t, b C.uint8_t) {
	connectionListener.SetControllerLED(uint16(controllerNumber), uint8(r), uint8(g), uint8(b))
}

type VideoDecoderRenderer interface {
	Setup(format, width, height, redrawRate int) int
	Start()
	Stop()
	Cleanup()
	SubmitDecodeUnit(decodeUnit *DecodeUnit) int
	Capabilities() int
}

type DecodeUnit struct {
	FrameNumber                int
	FrameType                  int
	FrameHostProcessingLatency uint16
	ReceiveTimeMs              uint64
	EnqueueTimeMs              uint64
	PresentationTimeMs         uint
	FullLength                 int
	BufferList                 *Lentry
	HDRActive                  bool
	ColorSpace                 uint8
}

type Lentry struct {
	Next       *Lentry
	Data       []byte
	Length     int
	BufferType int
}

//export goDrSetup
func goDrSetup(videoFormat, width, height, redrawRate C.int, context unsafe.Pointer, drFlags C.int) C.int {
	if videoRenderer == nil {
		return 0
	}

	rc := videoRenderer.Setup(
		int(videoFormat),
		int(width), int(height), int(redrawRate),
	)

	if rc != 0 {
		return C.int(rc)
	}

	return 0
}

//export goDrStart
func goDrStart() {
	if videoRenderer != nil {
		videoRenderer.Start()
	}
}

//export goDrStop
func goDrStop() {
	if videoRenderer != nil {
		videoRenderer.Stop()
	}
}

//export goDrCleanup
func goDrCleanup() {
	if videoRenderer != nil {
		videoRenderer.Cleanup()
		videoRenderer = nil
	}
}

//export goDrSubmitDecodeUnit
func goDrSubmitDecodeUnit(unit *C.DECODE_UNIT) C.int {
	if videoRenderer == nil || unit == nil {
		return C.DR_OK
	}

	var bufferList *Lentry
	if unit.bufferList != nil {
		bufferList = convertCLentryToGo(unit.bufferList)
	}

	decodeUnit := &DecodeUnit{
		FrameNumber:                int(unit.frameNumber),
		FrameType:                  int(unit.frameType),
		FrameHostProcessingLatency: uint16(unit.frameHostProcessingLatency),
		ReceiveTimeMs:              uint64(unit.receiveTimeMs),
		EnqueueTimeMs:              uint64(unit.enqueueTimeMs),
		PresentationTimeMs:         uint(unit.presentationTimeMs),
		FullLength:                 int(unit.fullLength),
		BufferList:                 bufferList,
		HDRActive:                  bool(unit.hdrActive),
		ColorSpace:                 uint8(unit.colorspace),
	}

	result := videoRenderer.SubmitDecodeUnit(decodeUnit)

	return C.int(result)
}

func convertCLentryToGo(cEntry *C.LENTRY) *Lentry {
	if cEntry == nil {
		return nil

	}

	goEntry := &Lentry{
		Length:     int(cEntry.length),
		BufferType: int(cEntry.bufferType),
	}

	if cEntry.data != nil && cEntry.length > 0 {
		goEntry.Data = C.GoBytes(unsafe.Pointer(cEntry.data), cEntry.length)
	}

	if cEntry.next != nil {
		goEntry.Next = convertCLentryToGo(cEntry.next)
	}

	return goEntry
}

type AudioRenderer interface {
	Init(audioConfiguration AudioConfiguration, opusConfig *OpusMultiStreamConfiguration) int
	Start()
	Stop()
	Cleanup()
	AudioRendererDecodeAndPlaySample(sampleData []byte, sampleLength int)
	Capabilities() int
}

const AUDIO_CONFIGURATION_MAX_CHANNEL_COUNT int = 8

type OpusMultiStreamConfiguration struct {
	SampleRate      int
	ChannelCount    int
	Streams         int
	CoupledStreams  int
	SamplesPerFrame int
	Mapping         [AUDIO_CONFIGURATION_MAX_CHANNEL_COUNT]byte
}

//export goArInit
func goArInit(audioConfiguration C.int, cfg *C.OPUS_MULTISTREAM_CONFIGURATION, context unsafe.Pointer, arFlags C.int) C.int {
	if audioRenderer == nil {
		return 0
	}

	audioConfig, err := NewAudioConfiguration(int(audioConfiguration))
	if err != nil {
		return C.int(-1)
	}

	var mapping [AUDIO_CONFIGURATION_MAX_CHANNEL_COUNT]byte
	for i := 0; i < AUDIO_CONFIGURATION_MAX_CHANNEL_COUNT; i++ {
		mapping[i] = byte(cfg.mapping[i])
	}

	opusConfig := &OpusMultiStreamConfiguration{
		SampleRate:      int(cfg.sampleRate),
		ChannelCount:    int(cfg.channelCount),
		Streams:         int(cfg.streams),
		CoupledStreams:  int(cfg.coupledStreams),
		SamplesPerFrame: int(cfg.samplesPerFrame),
		Mapping:         mapping,
	}

	rc := audioRenderer.Init(audioConfig, opusConfig)

	return C.int(rc)
}

//export goArStart
func goArStart() {
	if audioRenderer != nil {
		audioRenderer.Start()
	}
}

//export goArStop
func goArStop() {
	if audioRenderer != nil {
		audioRenderer.Stop()
	}
}

//export goArCleanup
func goArCleanup() {
	if audioRenderer != nil {
		audioRenderer.Cleanup()
		audioRenderer = nil
	}
}

//export goArDecodeAndPlaySample
func goArDecodeAndPlaySample(sampleData *C.char, sampleLength C.int) {
	if audioRenderer == nil {
		return
	}

	data := C.GoBytes(unsafe.Pointer(sampleData), sampleLength)
	audioRenderer.AudioRendererDecodeAndPlaySample(data, int(sampleLength))
}
