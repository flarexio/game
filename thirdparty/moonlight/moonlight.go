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
	"errors"
	"unsafe"
)

// Values for the 'streamingRemotely' field below
type StreamingRemotely int

const (
	STREAM_CFG_LOCAL StreamingRemotely = iota
	STREAM_CFG_REMOTE
	STREAM_CFG_AUTO
)

// Values for the 'colorSpace' field below.
// Rec. 2020 is not supported with H.264 video streams on GFE hosts.
type ColorSpace int

const (
	COLORSPACE_REC_601 ColorSpace = iota
	COLORSPACE_REC_709
	COLORSPACE_REC_2020
)

// Values for the 'colorRange' field below
type ColorRange int

const (
	COLOR_RANGE_LIMITED ColorRange = iota
	COLOR_RANGE_FULL
)

// Passed in StreamConfiguration.supportedVideoFormats to specify supported codecs
// and to DecoderRendererSetup() to specify selected codec.
type VideoFormat int

const (
	VIDEO_FORMAT_H264            VideoFormat = 0x0001 // H.264 High Profile
	VIDEO_FORMAT_H264_HIGH8_444  VideoFormat = 0x0004 // H.264 High 4:4:4 8-bit Profile
	VIDEO_FORMAT_H265            VideoFormat = 0x0100 // HEVC Main Profile
	VIDEO_FORMAT_H265_MAIN10     VideoFormat = 0x0200 // HEVC Main10 Profile
	VIDEO_FORMAT_H265_REXT8_444  VideoFormat = 0x0400 // HEVC RExt 4:4:4 8-bit Profile
	VIDEO_FORMAT_H265_REXT10_444 VideoFormat = 0x0800 // HEVC RExt 4:4:4 10-bit Profile
	VIDEO_FORMAT_AV1_MAIN8       VideoFormat = 0x1000 // AV1 Main 8-bit profile
	VIDEO_FORMAT_AV1_MAIN10      VideoFormat = 0x2000 // AV1 Main 10-bit profile
	VIDEO_FORMAT_AV1_HIGH8_444   VideoFormat = 0x4000 // AV1 High 4:4:4 8-bit profile
	VIDEO_FORMAT_AV1_HIGH10_444  VideoFormat = 0x8000 // AV1 High 4:4:4 10-bit profile
)

// Masks for clients to use to match video codecs without profile-specific details.
type VideoFormatMask int

const (
	VIDEO_FORMAT_MASK_H264   VideoFormatMask = 0x000F
	VIDEO_FORMAT_MASK_H265   VideoFormatMask = 0x0F00
	VIDEO_FORMAT_MASK_AV1    VideoFormatMask = 0xF000
	VIDEO_FORMAT_MASK_10BIT  VideoFormatMask = 0xAA00
	VIDEO_FORMAT_MASK_YUV444 VideoFormatMask = 0xCC04
)

// Values for 'encryptionFlags' field below
type EncryptionFlags int

const (
	ENCFLG_NONE  EncryptionFlags = 0x00000000
	ENCFLG_AUDIO EncryptionFlags = 0x00000001
	ENCFLG_VIDEO EncryptionFlags = 0x00000002
	ENCFLG_ALL   EncryptionFlags = 0xFFFFFFFF
)

const (
	// This callback provides Annex B formatted elementary stream data to the
	// decoder. If the decoder is unable to process the submitted data for some reason,
	// it must return DR_NEED_IDR to generate a keyframe.
	DR_OK       int = 0
	DR_NEED_IDR int = -1
)

// These identify codec configuration data in the buffer lists
// of frames identified as IDR frames for H.264 and HEVC formats.
// For other codecs, all data is marked as BUFFER_TYPE_PICDATA.
type BufferType int

const (
	BUFFER_TYPE_PICDATA BufferType = 0x00
	BUFFER_TYPE_SPS     BufferType = 0x01
	BUFFER_TYPE_PPS     BufferType = 0x02
	BUFFER_TYPE_VPS     BufferType = 0x03
)

type FrameType int

const (
	// This is a standard frame which references the IDR frame and
	// previous P-frames.
	FRAME_TYPE_PFRAME FrameType = 0x00

	// This is a key frame.
	//
	// For H.264 and HEVC, this means the frame contains SPS, PPS, and VPS (HEVC only) NALUs
	// as the first buffers in the list. The I-frame data follows immediately
	// after the codec configuration NALUs.
	//
	// For other codecs, any configuration data is not split into separate buffers.
	FRAME_TYPE_IDR FrameType = 0x01
)

func ParseStreamingRemotely(remote string) (StreamingRemotely, error) {
	switch remote {
	case "local":
		return STREAM_CFG_LOCAL, nil
	case "remote":
		return STREAM_CFG_REMOTE, nil
	case "auto":
		return STREAM_CFG_AUTO, nil
	default:
		return -1, errors.New("invalid remote value")
	}
}

func ParseVideoFormat(s string) (VideoFormat, error) {
	switch s {
	case "h264", "avc":
		return VIDEO_FORMAT_H264, nil
	case "h264_high8_444", "avc_high8_444":
		return VIDEO_FORMAT_H264_HIGH8_444, nil
	case "h265", "hevc":
		return VIDEO_FORMAT_H265, nil
	case "h265_main10", "hevc_main10":
		return VIDEO_FORMAT_H265_MAIN10, nil
	case "h265_rext8_444", "hevc_rext8_444":
		return VIDEO_FORMAT_H265_REXT8_444, nil
	case "h265_rext10_444", "hevc_rext10_444":
		return VIDEO_FORMAT_H265_REXT10_444, nil
	case "av1", "av1_main8":
		return VIDEO_FORMAT_AV1_MAIN8, nil
	case "av1_main10":
		return VIDEO_FORMAT_AV1_MAIN10, nil
	case "av1_high8_444":
		return VIDEO_FORMAT_AV1_HIGH8_444, nil
	case "av1_high10_444":
		return VIDEO_FORMAT_AV1_HIGH10_444, nil
	default:
		return 0, errors.New("invalid supportedVideoFormats value")
	}
}

func ParseEncryptionFlags(s string) (EncryptionFlags, error) {
	switch s {
	case "none":
		return ENCFLG_NONE, nil
	case "audio":
		return ENCFLG_AUDIO, nil
	case "video":
		return ENCFLG_VIDEO, nil
	case "all":
		return ENCFLG_ALL, nil
	default:
		return -1, errors.New("invalid encryptionFlags value")
	}
}

func ParseColorRange(s string) (ColorRange, error) {
	switch s {
	case "limited":
		return COLOR_RANGE_LIMITED, nil
	case "full":
		return COLOR_RANGE_FULL, nil
	default:
		return -1, errors.New("invalid colorRange value")
	}
}

func ParseColorSpace(s string) (ColorSpace, error) {
	switch s {
	case "rec601":
		return COLORSPACE_REC_601, nil
	case "rec709":
		return COLORSPACE_REC_709, nil
	default:
		return -1, errors.New("invalid colorSpace value")
	}
}

var (
	// Specifies that the audio stream should be encoded
	AUDIO_CONFIGURATION_STEREO      = AudioConfiguration{2, 0x3}
	AUDIO_CONFIGURATION_51_SURROUND = AudioConfiguration{6, 0x3F}
	AUDIO_CONFIGURATION_71_SURROUND = AudioConfiguration{8, 0x63F}
)

func ParseAudioConfiguration(s string) (AudioConfiguration, error) {
	switch s {
	case "stereo":
		return AUDIO_CONFIGURATION_STEREO, nil
	case "5.1":
		return AUDIO_CONFIGURATION_51_SURROUND, nil
	case "7.1":
		return AUDIO_CONFIGURATION_71_SURROUND, nil
	default:
		return AudioConfiguration{}, errors.New("invalid audioConfiguration value")
	}
}

func NewAudioConfiguration(i int) (AudioConfiguration, error) {
	if i&0xFF != 0xCA {
		return AudioConfiguration{}, errors.New("invalid audio configuration")
	}

	return AudioConfiguration{
		ChannelCount: (i >> 8) & 0xFF,
		ChannelMask:  (i >> 16) & 0xFFFF,
	}, nil
}

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
