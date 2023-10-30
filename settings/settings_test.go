package settings

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInit(t *testing.T) {
	err := Init("../hades.yaml")
	assert.Nil(t, err)
	assert.Equal(t, "hades", Conf.Name)
	assert.Equal(t, "log", Conf.LogConfig.Ext)
	assert.Equal(t, int8(1), Conf.DBConfig.IndexType)
}
