package izero

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ResizedImage struct {
	StaticImage  image.Image
	DynamicImage *gif.GIF
	ContentType  string
	Name         string
	Size         *ImageSize
}

func (rImg *ResizedImage) ToReader() (io.Reader, error) {
	// create buffer
	buff := new(bytes.Buffer)
	var err error
	switch rImg.ContentType {
	case "image/jpeg", "image/jpg":
		err = jpeg.Encode(buff, rImg.StaticImage, &jpeg.Options{Quality: rImg.Size.Quality})
	case "image/png":
		err = png.Encode(buff, rImg.StaticImage)
	case "image/gif":
		err = gif.EncodeAll(buff, rImg.DynamicImage)
	}
	if err != nil {
		return nil, err
	}

	// convert buffer to reader
	return bytes.NewReader(buff.Bytes()), nil
}

func (rImg *ResizedImage) saveTo(dest string) error {
	dest = strings.TrimSpace(dest)
	if ok, err := createFolderPath(filepath.Join(dest, rImg.Size.Name)); !ok {
		return err
	}
	imgFile, err := os.Create(filepath.Join(dest, rImg.Size.Name, rImg.Name))
	if err != nil {
		return err
	}
	defer imgFile.Close()
	switch rImg.ContentType {
	case "image/jpeg", "image/jpg":
		err = jpeg.Encode(imgFile, rImg.StaticImage, &jpeg.Options{Quality: rImg.Size.Quality})
	case "image/png":
		err = png.Encode(imgFile, rImg.StaticImage)
	case "image/gif":
		err = gif.EncodeAll(imgFile, rImg.DynamicImage)
	}
	return err
}
