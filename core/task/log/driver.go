package log

import "fmt"

func GetLogDriver(logDriverType string) Driver {
	switch logDriverType {
	case DriverTypeFile:
		return GetFileLogDriver()
	case DriverTypeMongo:
		panic("mongo driver not implemented")
	case DriverTypeEs:
		panic("es driver not implemented")
	default:
		panic(fmt.Sprintf("invalid log driver type: %s", logDriverType))
	}
}
