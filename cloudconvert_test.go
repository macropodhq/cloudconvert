package cloudconvert

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var TEST_KEY = os.Getenv("CLOUDCONVERT_KEY")

func TestNew(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	assert.Equal(t, TEST_KEY, c.APIKey)
}

func TestCreateProcess(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	p, err := c.CreateProcess("pdf", "png")
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestInvalidCreateProcessInputType(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	_, err = c.CreateProcess("invalid-type", "png")
	assert.IsType(t, ErrCloudConvert{}, err)
}

func TestInvalidCreateProcessOutputType(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	_, err = c.CreateProcess("pdf", "invalid-type")
	assert.IsType(t, ErrCloudConvert{}, err)
}

func TestConvertStream(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	p, err := c.CreateProcess("pdf", "png")
	assert.NoError(t, err)
	assert.NotNil(t, p)

	f, err := os.Open("testdata/Creativecommons-informational-flyer_eng.pdf")
	assert.NoError(t, err)

	s, err := p.Wait(true).ConvertStream(f, "Creativecommons-informational-flyer_eng.pdf", "png", nil)
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

func TestDownload(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	p, err := c.CreateProcess("pdf", "png")
	assert.NoError(t, err)
	assert.NotNil(t, p)

	f, err := os.Open("testdata/Creativecommons-informational-flyer_eng.pdf")
	assert.NoError(t, err)

	s, err := p.Wait(true).ConvertStream(f, "Creativecommons-informational-flyer_eng.pdf", "png", nil)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	d, err := p.Download()
	assert.NoError(t, err)
	assert.NotNil(t, d)

	if d != nil {
		b := bytes.NewBuffer(nil)
		_, err := io.Copy(b, d)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, b.Len())
	}
}

func TestDownloadOne(t *testing.T) {
	c, err := New(TEST_KEY)
	assert.NoError(t, err)

	p, err := c.CreateProcess("pdf", "png")
	assert.NoError(t, err)
	assert.NotNil(t, p)

	f, err := os.Open("testdata/Creativecommons-informational-flyer_eng.pdf")
	assert.NoError(t, err)

	s, err := p.Wait(true).ConvertStream(f, "Creativecommons-informational-flyer_eng.pdf", "png", nil)
	assert.NoError(t, err)
	assert.NotNil(t, s)

	assert.Equal(t, "zip", s.Output.Ext)
	assert.NotEmpty(t, s.Output.Files)

	if len(s.Output.Files) == 0 {
		return
	}

	d, err := p.DownloadOne(s.Output.Files[0])
	assert.NoError(t, err)
	assert.NotNil(t, d)

	if d != nil {
		b := bytes.NewBuffer(nil)
		_, err := io.Copy(b, d)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, b.Len())
	}
}
