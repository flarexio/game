#include <stdarg.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <opus_multistream.h>
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

static OpusMSDecoder* decoder = NULL;
static OPUS_MULTISTREAM_CONFIGURATION opusConfig;

static OpusEncoder* encoder = NULL;
static short* decodedPcm = NULL;
static short* encodePcm = NULL;
static unsigned char* opusData = NULL;

extern int goArInit(int audioConfiguration, POPUS_MULTISTREAM_CONFIGURATION opusConfig, void* context, int arFlags);
extern void goArCleanup(void);
extern void goArPlayEncodedSample(unsigned char* opusData, int opusLength);

int arInit(int audioConfiguration, POPUS_MULTISTREAM_CONFIGURATION config, void* context, int arFlags) {
    int err;

    if (decoder != NULL) {
        opus_multistream_decoder_destroy(decoder);
        decoder = NULL;
    }

    if (encoder != NULL) {
        opus_encoder_destroy(encoder);
        encoder = NULL;
    }

    if (config != NULL) {
        opusConfig = *config;
    }

    decoder = opus_multistream_decoder_create(
        config->sampleRate,
        config->channelCount,
        config->streams,
        config->coupledStreams,
        config->mapping,
        &err
    );

    if (err != OPUS_OK || decoder == NULL) {
        return -1;
    }

    int outputChannels = (config->channelCount > 2) ? 2 : config->channelCount;
    int bitrate = 64000 * outputChannels;

    encoder = opus_encoder_create(
        config->sampleRate,
        outputChannels,
        OPUS_APPLICATION_RESTRICTED_LOWDELAY,
        &err
    );

    if (err != OPUS_OK || encoder == NULL) {
        opus_multistream_decoder_destroy(decoder);
        decoder = NULL;
        return -1;
    }

    opus_encoder_ctl(encoder, OPUS_SET_BITRATE(bitrate));
    opus_encoder_ctl(encoder, OPUS_SET_COMPLEXITY(5));

    int pcmBufferSize = config->samplesPerFrame * config->channelCount;
    if (decodedPcm != NULL) {
        free(decodedPcm);
        decodedPcm = NULL;
    }

    decodedPcm = (short*)malloc(sizeof(short) * pcmBufferSize);
    if (decodedPcm == NULL) {
        opus_multistream_decoder_destroy(decoder);
        opus_encoder_destroy(encoder);
        decoder = NULL;
        encoder = NULL;
        return -1;
    }

    if (config->channelCount > 2) {
        if (encodePcm != NULL) {
            free(encodePcm);
            encodePcm = NULL;
        }

        encodePcm = (short*)malloc(sizeof(short) * config->samplesPerFrame * 2);
        if (encodePcm == NULL) {
            free(decodedPcm);
            opus_multistream_decoder_destroy(decoder);
            opus_encoder_destroy(encoder);
            decodedPcm = NULL;
            decoder = NULL;
            encoder = NULL;
            return -1;
        }
    }

    if (opusData != NULL) {
        free(opusData);
        opusData = NULL;
    }

    opusData = (unsigned char*)malloc(4000);
    if (opusData == NULL) {
        if (encodePcm != NULL) {
            free(encodePcm);
        }

        free(decodedPcm);
        opus_multistream_decoder_destroy(decoder);
        opus_encoder_destroy(encoder);
        decodedPcm = NULL;
        encodePcm = NULL;
        decoder = NULL;
        encoder = NULL;
        return -1;
    }

    return goArInit(audioConfiguration, config, context, arFlags);
}

void arCleanup(void) {
    if (decoder != NULL) {
        opus_multistream_decoder_destroy(decoder);
        decoder = NULL;
    }

    if (encoder != NULL) {
        opus_encoder_destroy(encoder);
        encoder = NULL;
    }

    if (decodedPcm != NULL) {
        free(decodedPcm);
        decodedPcm = NULL;
    }

    if (encodePcm != NULL) {
        free(encodePcm);
        encodePcm = NULL;
    }

    if (opusData != NULL) {
        free(opusData);
        opusData = NULL;
    }

    goArCleanup();
}

void arDecodeAndPlaySample(char* sampleData, int sampleLength) {
    if (decoder == NULL || encoder == NULL || sampleData == NULL || sampleLength <= 0) {
        return;
    }

    int decodedSamples = opus_multistream_decode(
        decoder,
        (const unsigned char*)sampleData,
        sampleLength,
        decodedPcm,
        opusConfig.samplesPerFrame,
        0
    );

    if (decodedSamples < 0) {
        return;
    }

    short* inputPcm = decodedPcm;
    if (opusConfig.channelCount > 2 && encodePcm != NULL) {
        for (int i = 0; i < decodedSamples; i++) {
            encodePcm[i * 2 + 0] = decodedPcm[i * opusConfig.channelCount + 0]; // Left
            encodePcm[i * 2 + 1] = decodedPcm[i * opusConfig.channelCount + 1]; // Right
        }

        inputPcm = encodePcm;
    }

    int encodedBytes = opus_encode(
        encoder,
        inputPcm,
        decodedSamples,
        opusData,
        4000
    );

    if (encodedBytes < 0) {
        return;
    }

    goArPlayEncodedSample(opusData, encodedBytes);
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

    arCallbacks->init = arInit;
    arCallbacks->start = goArStart;
    arCallbacks->stop = goArStop;
    arCallbacks->cleanup = arCleanup;
    arCallbacks->decodeAndPlaySample = arDecodeAndPlaySample;
    arCallbacks->capabilities = 0;
}
