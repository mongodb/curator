package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockInterfaces(t *testing.T) {
	manager := &Manager{}
	_, ok := interface{}(manager).(Manager)
	assert.True(t, ok)

	process := &Process{}
	_, ok = interface{}(process).(Process)
	assert.True(t, ok)

	remoteClient := &RemoteClient{}
	_, ok = interface{}(remoteClient).(RemoteClient)
	assert.True(t, ok)
}
