package gimlet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteResolutionHelpers(t *testing.T) {
	assert := assert.New(t)

	res := getDefaultRoute(false, "/foo", "/bar")
	assert.Equal("/bar", res)
	res = getDefaultRoute(true, "/foo", "/bar")
	assert.Equal("/foo/bar", res)
	res = getDefaultRoute(true, "/foo", "/foo/bar")
	assert.Equal("/foo/bar", res)

	res = getVersionedRoute(false, "/foo", 1, "/bar")
	assert.Equal("/v1/bar", res)
	res = getVersionedRoute(true, "/foo", 1, "/bar")
	assert.Equal("/foo/v1/bar", res)
	res = getVersionedRoute(true, "/foo", 1, "/foo/bar")
	assert.Equal("/foo/v1/bar", res)

	handler, err := NewApp().Handler()
	assert.NoError(err)
	h := getRouteHandlerWithMiddlware(nil, handler)
	assert.Equal(handler, h)
	h = getRouteHandlerWithMiddlware([]Middleware{NewRecoveryLogger()}, handler)
	assert.NotEqual(handler, h)

}
