package game

type Gamepad interface {
	Connect() error
	Update(report GamepadReport) error
	Close()
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
