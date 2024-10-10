package game

/*
#cgo CFLAGS: -Wno-pragma-pack
#cgo CFLAGS: -IViGEm
#cgo LDFLAGS: -LViGEm -lViGEmClient
#include <stdlib.h>
#include <Windows.h>
#include <ViGEm/Client.h>
*/
import "C"
import (
	"errors"
)

type Gamepad interface {
	Connect() error
	Update(report GamepadReport) error
	Close()
}

func NewXBoxGamepad() Gamepad {
	return &xboxGamepad{}
}

type xboxGamepad struct {
	client C.PVIGEM_CLIENT
	target C.PVIGEM_TARGET
}

func (gamepad *xboxGamepad) Connect() error {
	// Initialize ViGEm client
	client := C.vigem_alloc()
	if client == nil {
		return errors.New("failed to allocate ViGEm client")
	}
	gamepad.client = client

	// Connect to ViGEmBus
	if C.vigem_connect(client) != C.VIGEM_ERROR_NONE {
		return errors.New("failed to connect to ViGEmBus")
	}

	// Create a virtual Xbox 360 controller
	target := C.vigem_target_x360_alloc()
	if target == nil {
		return errors.New("failed to allocate Xbox 360 target")
	}
	gamepad.target = target

	// Add the virtual controller to the system
	if C.vigem_target_add(client, target) != C.VIGEM_ERROR_NONE {
		return errors.New("failed to add virtual controller")
	}

	return nil
}

func (gamepad *xboxGamepad) Update(r GamepadReport) error {
	var report C.XUSB_REPORT
	report.wButtons = C.USHORT(r.Buttons())
	report.bLeftTrigger = C.BYTE(r.LeftTrigger())
	report.bRightTrigger = C.BYTE(r.RightTrigger())

	leftThumbStick := r.LeftThumbStick()
	report.sThumbLX = C.SHORT(leftThumbStick.X)
	report.sThumbLY = C.SHORT(leftThumbStick.Y)

	rightThumbStick := r.RightThumbStick()
	report.sThumbRX = C.SHORT(rightThumbStick.X)
	report.sThumbRY = C.SHORT(rightThumbStick.Y)

	if C.vigem_target_x360_update(gamepad.client, gamepad.target, report) != C.VIGEM_ERROR_NONE {
		return errors.New("failed to update virtual controller")
	}

	return nil
}

func (gamepad *xboxGamepad) Close() {
	client := gamepad.client
	target := gamepad.target

	C.vigem_target_remove(client, target)
	C.vigem_target_free(target)

	C.vigem_disconnect(client)
	C.vigem_free(client)
}

type ThumbStick struct {
	X int16
	Y int16
}

type GamepadReport interface {
	Buttons() uint16
	LeftTrigger() uint8
	RightTrigger() uint8
	LeftThumbStick() ThumbStick
	RightThumbStick() ThumbStick
}

func NewXBoxGamepadReport(
	buttons uint16,
	leftTrigger uint8,
	rightTrigger uint8,
	leftThumbStickX int16,
	leftThumbStickY int16,
	rightThumbStickX int16,
	rightThumbStickY int16,
) GamepadReport {
	return &xboxGamepadReport{
		buttons,
		leftTrigger, rightTrigger,
		leftThumbStickX, leftThumbStickY,
		rightThumbStickX, rightThumbStickY,
	}
}

type xboxGamepadReport struct {
	buttons          uint16
	leftTrigger      uint8
	rightTrigger     uint8
	leftThumbStickX  int16
	leftThumbStickY  int16
	rightThumbStickX int16
	rightThumbStickY int16
}

func (report *xboxGamepadReport) Buttons() uint16 {
	return report.buttons
}

func (report *xboxGamepadReport) LeftTrigger() uint8 {
	return report.leftTrigger
}

func (report *xboxGamepadReport) RightTrigger() uint8 {
	return report.rightTrigger
}

func (report *xboxGamepadReport) LeftThumbStick() ThumbStick {
	return ThumbStick{
		X: report.leftThumbStickX,
		Y: report.leftThumbStickY,
	}
}

func (report *xboxGamepadReport) RightThumbStick() ThumbStick {
	return ThumbStick{
		X: report.rightThumbStickX,
		Y: report.rightThumbStickY,
	}
}
