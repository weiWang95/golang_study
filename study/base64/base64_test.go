package base64

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase64(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(Base64("test"), "abc", "should be equal")
}
