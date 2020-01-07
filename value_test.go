package flagx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValue(t *testing.T) {
	assert.Equal(t, "", new(stringValue).String())
	assert.Equal(t, "false", new(boolValue).String())
	assert.Equal(t, "0", new(float64Value).String())
	assert.Equal(t, "0", new(intValue).String())
	assert.Equal(t, "0s", new(durationValue).String())
	assert.Equal(t, "0", new(uintValue).String())
}
