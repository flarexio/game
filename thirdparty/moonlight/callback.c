#include <stdarg.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include "callback.h"

extern void goClLogMessage(const char* message);

void clLogMessage(const char* format, ...) {
    char buffer[2048];
    va_list args;

    va_start(args, format);
    vsnprintf(buffer, sizeof(buffer), format, args);
    va_end(args);

    goClLogMessage(buffer);
}

void setupCallbacks(
    PCONNECTION_LISTENER_CALLBACKS clCallbacks,
    PDECODER_RENDERER_CALLBACKS drCallbacks,
    PAUDIO_RENDERER_CALLBACKS arCallbacks
) {
    if (clCallbacks == NULL) {
        return;
    }

    clCallbacks->stageStarting = goClStageStarting;
    clCallbacks->stageComplete = goClStageComplete;
    clCallbacks->stageFailed = goClStageFailed;
    clCallbacks->connectionStarted = goClConnectionStarted;
    clCallbacks->connectionTerminated = goClConnectionTerminated;
    clCallbacks->logMessage = clLogMessage;
    clCallbacks->rumble = goClRumble;
    clCallbacks->connectionStatusUpdate = goClConnectionStatusUpdate;
    clCallbacks->setHdrMode = goClSetHDRMode;
    clCallbacks->rumbleTriggers = goClRumbleTriggers;
    clCallbacks->setMotionEventState = goClSetMotionEventState;
    clCallbacks->setControllerLED = goClSetControllerLED;

    if (drCallbacks == NULL) {
        return;
    }

    drCallbacks->setup = goDrSetup;
    drCallbacks->start = goDrStart;
    drCallbacks->stop = goDrStop;
    drCallbacks->cleanup = goDrCleanup;
    drCallbacks->submitDecodeUnit = goDrSubmitDecodeUnit;
    drCallbacks->capabilities = 0;

    if (arCallbacks == NULL) {
        return;
    }

    arCallbacks->init = goArInit;
    arCallbacks->start = goArStart;
    arCallbacks->stop = goArStop;
    arCallbacks->cleanup = goArCleanup;
    arCallbacks->decodeAndPlaySample = goArDecodeAndPlaySample;
    arCallbacks->capabilities = 0;
}
