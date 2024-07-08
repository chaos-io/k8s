package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEncodeString(t *testing.T) {
	encodeString := GetEncodeString("2ixLkVepVChtU01SYBTJO0s6wNe")
	assert.Equal(t, "hvozkguvmwqoyizmf", encodeString)
}
