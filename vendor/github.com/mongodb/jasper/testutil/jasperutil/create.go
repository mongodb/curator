package jasperutil

import (
	"fmt"
	"time"

	"github.com/mongodb/jasper"
)

func YesCreateOpts(timeout time.Duration) jasper.CreateOptions {
	return jasper.CreateOptions{Args: []string{"yes"}, Timeout: timeout}
}

func TrueCreateOpts() *jasper.CreateOptions {
	return &jasper.CreateOptions{
		Args: []string{"true"},
	}
}

func FalseCreateOpts() *jasper.CreateOptions {
	return &jasper.CreateOptions{
		Args: []string{"false"},
	}
}

func SleepCreateOpts(num int) *jasper.CreateOptions {
	return &jasper.CreateOptions{
		Args: []string{"sleep", fmt.Sprint(num)},
	}
}
