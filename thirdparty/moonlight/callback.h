#ifndef CALLBACK_H
#define CALLBACK_H

#include <Limelight.h>

#ifdef __cplusplus
extern "C" {
#endif

// Connection Listener Callbacks
extern void goClStageStarting(int stage);
extern void goClStageComplete(int stage);
extern void goClStageFailed(int stage, int errorCode);
extern void goClConnectionStarted(void);
extern void goClConnectionTerminated(int errorCode);
extern void goClLogMessage(const char* format, ...);
extern void goClRumble(unsigned short controllerNumber, unsigned short lowFreqMotor, unsigned short highFreqMotor);
extern void goClConnectionStatusUpdate(int connectionStatus);
extern void goClSetHDRMode(bool hdrEnabled);
extern void goClRumbleTriggers(uint16_t controllerNumber, uint16_t leftTriggerMotor, uint16_t rightTriggerMotor);
extern void goClSetMotionEventState(uint16_t controllerNumber, uint8_t motionType, uint16_t reportRateHz);
extern void goClSetControllerLED(uint16_t controllerNumber, uint8_t r, uint8_t g, uint8_t b);

// Video Decoder Callbacks
extern int goDrSetup(int videoFormat, int width, int height, int redrawRate, void* context, int drFlags);
extern void goDrStart(void);
extern void goDrStop(void);
extern void goDrCleanup(void);
extern int goDrSubmitDecodeUnit(PDECODE_UNIT decodeUnit);

// Audio Renderer Callbacks
extern int goArInit(int audioConfiguration, POPUS_MULTISTREAM_CONFIGURATION opusConfig, void* context, int arFlags);
extern void goArStart(void);
extern void goArStop(void);
extern void goArCleanup(void);
extern void goArDecodeAndPlaySample(char* sampleData, int sampleLength);

// Helper function to setup callbacks
void setupCallbacks(
    PCONNECTION_LISTENER_CALLBACKS clCallbacks,
    PDECODER_RENDERER_CALLBACKS drCallbacks,
    PAUDIO_RENDERER_CALLBACKS arCallbacks
);

#ifdef __cplusplus
}
#endif

#endif // CALLBACK_H
