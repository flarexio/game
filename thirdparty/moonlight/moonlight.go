package moonlight

/*
#cgo CFLAGS:  -I${SRCDIR}/../moonlight-common-c/src
#cgo LDFLAGS: -L${SRCDIR}/../moonlight-common-c/build -lmoonlight-common-c -Wl,--allow-multiple-definition
#include <stdlib.h>
#include <Limelight.h>
#include <Windows.h>
*/
import "C"
import (
	"crypto/rand"
	"unsafe"
)

const (
	// Values for the 'streamingRemotely' field below
	STREAM_CFG_LOCAL  int = 0
	STREAM_CFG_REMOTE int = 1
	STREAM_CFG_AUTO   int = 2

	// Values for the 'colorSpace' field below.
	// Rec. 2020 is not supported with H.264 video streams on GFE hosts.
	COLORSPACE_REC_601  int = 0
	COLORSPACE_REC_709  int = 1
	COLORSPACE_REC_2020 int = 2

	// Values for the 'colorRange' field below
	COLOR_RANGE_LIMITED int = 0
	COLOR_RANGE_FULL    int = 1

	// Passed in StreamConfiguration.supportedVideoFormats to specify supported codecs
	// and to DecoderRendererSetup() to specify selected codec.
	VIDEO_FORMAT_H264            int = 0x0001 // H.264 High Profile
	VIDEO_FORMAT_H264_HIGH8_444  int = 0x0004 // H.264 High 4:4:4 8-bit Profile
	VIDEO_FORMAT_H265            int = 0x0100 // HEVC Main Profile
	VIDEO_FORMAT_H265_MAIN10     int = 0x0200 // HEVC Main10 Profile
	VIDEO_FORMAT_H265_REXT8_444  int = 0x0400 // HEVC RExt 4:4:4 8-bit Profile
	VIDEO_FORMAT_H265_REXT10_444 int = 0x0800 // HEVC RExt 4:4:4 10-bit Profile
	VIDEO_FORMAT_AV1_MAIN8       int = 0x1000 // AV1 Main 8-bit profile
	VIDEO_FORMAT_AV1_MAIN10      int = 0x2000 // AV1 Main 10-bit profile
	VIDEO_FORMAT_AV1_HIGH8_444   int = 0x4000 // AV1 High 4:4:4 8-bit profile
	VIDEO_FORMAT_AV1_HIGH10_444  int = 0x8000 // AV1 High 4:4:4 10-bit profile

	// Masks for clients to use to match video codecs without profile-specific details.
	VIDEO_FORMAT_MASK_H264   int = 0x000F
	VIDEO_FORMAT_MASK_H265   int = 0x0F00
	VIDEO_FORMAT_MASK_AV1    int = 0xF000
	VIDEO_FORMAT_MASK_10BIT  int = 0xAA00
	VIDEO_FORMAT_MASK_YUV444 int = 0xCC04

	// Values for 'encryptionFlags' field below
	ENCFLG_NONE  int = 0x00000000
	ENCFLG_AUDIO int = 0x00000001
	ENCFLG_VIDEO int = 0x00000002
	ENCFLG_ALL   int = 0xFFFFFFFF
)

var (
	// Specifies that the audio stream should be encoded
	AUDIO_CONFIGURATION_STEREO      = AudioConfiguration{2, 0x3}
	AUDIO_CONFIGURATION_51_SURROUND = AudioConfiguration{6, 0x3F}
	AUDIO_CONFIGURATION_71_SURROUND = AudioConfiguration{8, 0x63F}
)

type AudioConfiguration struct {
	ChannelCount int
	ChannelMask  int
}

func (cfg *AudioConfiguration) SurroundAudioInfo() int {
	return cfg.ChannelMask<<16 | cfg.ChannelCount
}

func (cfg *AudioConfiguration) C() C.int {
	return C.int(cfg.ChannelMask<<16 | cfg.ChannelCount<<8 | 0xCA)
}

type ServerInformation struct {
	Address                string // Server host name or IP address in text form
	AppVersion             string // Text inside 'appversion' tag in /serverinfo
	GfeVersion             string // Text inside 'GfeVersion' tag in /serverinfo (if present)
	RTSPSessionURL         string // Text inside 'sessionUrl0' tag in /resume and /launch (if present)
	ServerCodecModeSupport int    // Specifies the 'ServerCodecModeSupport' from the /serverinfo response.
}

func (info *ServerInformation) C() (*C.SERVER_INFORMATION, func()) {
	cAddress := C.CString(info.Address)
	cAppVersion := C.CString(info.AppVersion)
	cGfeVersion := C.CString(info.GfeVersion)
	cRTSPSessionURL := C.CString(info.RTSPSessionURL)

	cServerInfo := &C.SERVER_INFORMATION{
		address:                cAddress,
		serverInfoAppVersion:   cAppVersion,
		serverInfoGfeVersion:   cGfeVersion,
		rtspSessionUrl:         cRTSPSessionURL,
		serverCodecModeSupport: C.int(info.ServerCodecModeSupport),
	}

	cleanup := func() {
		C.free(unsafe.Pointer(cAddress))
		C.free(unsafe.Pointer(cAppVersion))
		C.free(unsafe.Pointer(cGfeVersion))
		C.free(unsafe.Pointer(cRTSPSessionURL))
	}

	return cServerInfo, cleanup
}

func NewRemoteInputAES() (*RemoteInputAES, error) {
	ri := &RemoteInputAES{
		Key: [16]byte{},
		IV:  [16]byte{},
	}

	if _, err := rand.Read(ri.Key[:]); err != nil {
		return nil, err
	}

	if _, err := rand.Read(ri.IV[:]); err != nil {
		return nil, err
	}

	return ri, nil
}

type RemoteInputAES struct {
	Key [16]byte
	IV  [16]byte
}

type StreamConfiguration struct {
	// Dimensions in pixels of the desired video stream
	Width  int
	Height int

	// FPS of the desired video stream
	FPS int

	// Bitrate of the desired video stream (audio adds another ~1 Mbps). This
	// includes error correction data, so the actual encoder bitrate will be
	// about 20% lower when using the standard 20% FEC configuration.
	Bitrate int

	// Max video packet size in bytes (use 1024 if unsure). If STREAM_CFG_AUTO
	// determines the stream is remote (see below), it will cap this value at
	// 1024 to avoid MTU-related issues like packet loss and fragmentation.
	PacketSize int

	// Determines whether to enable remote (over the Internet)
	// streaming optimizations. If unsure, set to STREAM_CFG_AUTO.
	// STREAM_CFG_AUTO uses a heuristic (whether the target address is
	// in the RFC 1918 address blocks) to decide whether the stream
	// is remote or not.
	StreamingRemotely int

	// Specifies the channel configuration of the audio stream.
	// See AUDIO_CONFIGURATION constants and MAKE_AUDIO_CONFIGURATION() below.
	AudioConfiguration AudioConfiguration

	// Specifies the mask of supported video formats.
	// See VIDEO_FORMAT constants below.
	SupportedVideoFormats int

	// If specified, the client's display refresh rate x 100. For example,
	// 59.94 Hz would be specified as 5994. This is used by recent versions
	// of GFE for enhanced frame pacing.
	ClientRefreshRateX100 int

	// If specified, sets the encoder colorspace to the provided COLORSPACE_*
	// option (listed above). If not set, the encoder will default to Rec 601.
	ColorSpace int

	// If specified, sets the encoder color range to the provided COLOR_RANGE_*
	// option (listed above). If not set, the encoder will default to Limited.
	ColorRange int

	// Specifies the data streams where encryption may be enabled if supported
	// by the host PC. Ideally, you would pass ENCFLG_ALL to encrypt everything
	// that we support encrypting. However, lower performance hardware may not
	// be able to support encrypting heavy stuff like video or audio data, so
	// that encryption may be disabled here. Remote input encryption is always
	// enabled.
	EncryptionFlags int

	// AES encryption data for the remote input stream. This must be
	// the same as what was passed as rikey and rikeyid
	// in /launch and /resume requests.
	RemoteInputAES *RemoteInputAES
}

func (cfg *StreamConfiguration) C() (*C.STREAM_CONFIGURATION, func()) {
	cStreamConfig := &C.STREAM_CONFIGURATION{
		width:                 C.int(cfg.Width),
		height:                C.int(cfg.Height),
		fps:                   C.int(cfg.FPS),
		bitrate:               C.int(cfg.Bitrate),
		packetSize:            C.int(cfg.PacketSize),
		streamingRemotely:     C.int(cfg.StreamingRemotely),
		audioConfiguration:    cfg.AudioConfiguration.C(),
		supportedVideoFormats: C.int(cfg.SupportedVideoFormats),
		clientRefreshRateX100: C.int(cfg.ClientRefreshRateX100),
		colorSpace:            C.int(cfg.ColorSpace),
		colorRange:            C.int(cfg.ColorRange),
		encryptionFlags:       C.int(cfg.EncryptionFlags),
	}

	C.memcpy(
		unsafe.Pointer(&cStreamConfig.remoteInputAesKey[0]),
		unsafe.Pointer(&cfg.RemoteInputAES.Key[0]),
		C.size_t(16),
	)

	C.memcpy(
		unsafe.Pointer(&cStreamConfig.remoteInputAesIv[0]),
		unsafe.Pointer(&cfg.RemoteInputAES.IV[0]),
		C.size_t(16),
	)

	cleanup := func() {
		// No-op
	}

	return cStreamConfig, cleanup
}
