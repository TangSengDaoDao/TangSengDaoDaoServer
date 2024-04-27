package file

import (
	"crypto/sha512"
	"encoding/base64"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeCompose(t *testing.T) {
	s := &Service{}

	f1, err := os.OpenFile("../../../assets/assets/fileHelper.jpeg", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f2, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f3, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f4, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f5, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f6, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f7, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f8, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	f9, err := os.OpenFile("../../../assets/assets/u_10000.png", os.O_RDONLY, 0777)
	assert.NoError(t, err)

	img, err := s.MakeCompose([]io.ReadCloser{f1, f2, f3, f4, f5, f6, f7, f8, f9})
	assert.NoError(t, err)

	result, err := os.OpenFile("test.png", os.O_CREATE|os.O_WRONLY, 0777)
	assert.NoError(t, err)
	err = png.Encode(result, img)
	assert.NoError(t, err)

}

func TestUploadPCFile(t *testing.T) {
	file, err := os.Open("../../assets/assets/TangSengDaoDao-mac-1.0.5-arm64.zip")
	assert.NoError(t, err)
	bytes, err := ioutil.ReadAll(file)
	assert.NoError(t, err)
	hash := sha512.Sum512(bytes)
	// hash := sha512.New().Sum(bytes[:1024*1024])
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	println("编码结果")
	println(encoded)
}
