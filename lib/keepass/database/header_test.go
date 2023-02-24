package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomizeHeader(t *testing.T) {
	assert := assert.New(t)

	h1 := newHeader(3, 1, false, 10, IRSID(0), 8)
	h2 := newHeader(3, 1, false, 10, IRSID(0), 8)
	h1.randomize()
	h2.randomize()
	assert.NotEqual(h1.masterSeed, h2.masterSeed)
	assert.NotEqual(h1.transformSeed, h2.transformSeed)
	assert.NotEqual(h1.encryptionIV, h2.encryptionIV)
	assert.NotEqual(h1.streamStartBytes, h2.streamStartBytes)
	assert.NotEqual(h1.innerRandomStreamKey, h2.innerRandomStreamKey)
}
