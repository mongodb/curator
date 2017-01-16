package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func logkeeper() {
	conf := &send.BuildloggerConfig{
		URL:            "https://logkeeper.mongodb.org",
		Number:         0,
		Phase:          "magnetic_couplers",
		Builder:        "grip",
		Test:           "poc-buffer",
		Local:          grip.GetSender(),
		BufferInterval: 20 * time.Second,
		BufferCount:    4,
	}

	conf.SetCredentials("tychoish", "password")
	err := grip.GetSender().SetLevel(send.LevelInfo{Default: level.Debug, Threshold: level.Debug})
	if err != nil {
		fmt.Println(err)
		return
	}
	grip.Info("hi there")
	globalSender, err := send.MakeBuildlogger("grip", conf)
	if err != nil {
		return
	}
	grip.CatchError(err)

	if err = grip.SetSender(globalSender); err != nil {
		fmt.Println(err)
		return
	}

	grip.Notice("hello world")

	conf.CreateTest = true
	testSender, err := send.MakeBuildlogger("test", conf)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = grip.SetSender(testSender); err != nil {
		fmt.Println(err)
		return
	}

	grip.Emergency("what is this, test")
	grip.Emergency("what is this, test")
	grip.Emergency("what is this, test")
	grip.Emergency("what is this, test")
	grip.Emergency("what is this, test")

	globalSender.Close()
	testSender.Close()
}

func main() {
	logkeeper()
	return

	grip.Info(message.NewStack(0, "hi"))
	grip.Info(message.NewStack(1, "hi"))
	grip.Info(message.NewStack(2, "hi"))

	grip.CatchError(grip.SetSender(send.MakeJSONConsoleLogger()))

	grip.Info(message.NewStack(0, "hi"))
	grip.Info(message.NewStack(1, "hi"))
	grip.Info(message.NewStack(2, "hi"))

	if os.Getenv("LOGKEEPER") != "" {
		logkeeper()
	}

}
