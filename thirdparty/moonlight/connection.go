package moonlight

/*
#cgo CFLAGS:  -I${SRCDIR}/../moonlight-common-c/src
#cgo LDFLAGS: -L${SRCDIR}/../moonlight-common-c/build -lmoonlight-common-c -Wl,--allow-multiple-definition
#include <stdlib.h>
#include <Limelight.h>
#include <Windows.h>
*/
import "C"
import "fmt"

func StartConnection(serverInfo ServerInformation, streamConfig StreamConfiguration) error {
	cServerInfo, cleanupSI := serverInfo.C()
	defer cleanupSI()

	cStreamConfig, cleanupSC := streamConfig.C()
	defer cleanupSC()

	rc := C.LiStartConnection(
		cServerInfo, cStreamConfig,
		clCallbacks, drCallbacks, arCallbacks,
		nil, 0,
		nil, 0,
	)

	if rc < 0 {
		return fmt.Errorf("LiStartConnection failed with code %d", int(rc))
	}

	return nil
}

func StopConnection() {
	C.LiInterruptConnection()
	C.LiStopConnection()
}
