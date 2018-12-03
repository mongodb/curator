package poplar

import "runtime"

func getPathOfFile() string {
	_, file, _, _ := runtime.Caller(1)
	return file
}
