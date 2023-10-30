package logger

import (
	"github.com/stretchr/testify/assert"
	"hades/settings"
	"testing"
	"time"
)

func TestInfo(t *testing.T) {
	err := settings.Init("../../hades.yaml")
	assert.Nil(t, err)
	Setup(settings.Conf.LogConfig)
	Info("hello")
	Infof("hello %s", "world")
	time.Sleep(1 * time.Second)
}
