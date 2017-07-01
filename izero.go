package izer

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/RobCherry/vibrant"
	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
	"github.com/parnurzeal/gorequest"
)

var (
	imageExtensions   = []string{".jpeg", ".jpg", ".gif", ".png"}
	imageContentTypes = []string{"image/jpeg", "image/jpg", "image/gif", "image/png"}
)

type Img struct {
	Image       image.Image
	SizeName    string
	Name        string
	ContentType string
	GIF         *gif.GIF
}

func resizeImg(f os.FileInfo, sourceDir string, targetSizes map[string][]uint, , targetDir string, withCrop bool, backgroundColor *color.RGBA) (map[string]*Img, error) {
	var resizedImages map[string]*Img
	export := false
	targetDir = strings.TrimSpace(targetDir)
	if targetDir != "" {
		export = true
	}
	file, err := os.Open(filepath.Join(sourceDir, f.Name()))
	defer file.Close()
	if err != nil {
		return resizedImages, err
	}
	fileExtension := filepath.Ext(f.Name())
	fileExtension = strings.ToLower(fileExtension)
	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		return resizedImages, err
	}
	isImageFile, imgType := isImageFile(buff, fileExtension)

	if !isImageFile {
		err = errors.New("invalid_image_type")
		return resizedImages, err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return resizedImages, err
	}

	fileName := f.Name()

	if imgType == "image/gif" {
		var ok bool
		if export {
			if ok, err = createFolderPath(targetDir); !ok {
				return resizedImages, err
			}
		}
		if targetSizes != nil {
			var img *gif.GIF
			img, err = gif.DecodeAll(file)
			if err != nil {
				return resizedImages, err
			}
			for sizeName, size := range targetSizes {
				if export {
					if ok, err = createFolderPath(filepath.Join(targetDir, sizeName)); !ok {
						return resizedImages, err
					}
				}
				outGif := &gif.GIF{}
				outGif.Delay = img.Delay

				// Create a new RGBA image to hold the incremental frames.
				firstFrame := img.Image[0]
				r := image.Rect(0, 0, firstFrame.Bounds().Dx(), firstFrame.Bounds().Dy())
				rgbaImg := image.NewRGBA(r)

				// Resize each frame.
				for _, frame := range img.Image {
					bounds := frame.Bounds()
					draw.Draw(rgbaImg, bounds, frame, bounds.Min, draw.Over)
					m := ResizeImgToClosestSizeOfTargetSize(rgbaImg, size[0], size[1], resize.Lanczos3, withCrop)
					if size[0] > 0 && size[1] > 0 {
						if withCrop {
							m, _ = FitAspectRatioWithCroping(m, int(size[0]), int(size[1]), backgroundColor)
						} else {
							m = FitAspectRatioWithoutCroping(m, int(size[0]), int(size[1]), backgroundColor)
						}
					}
					outGif.Image = append(outGif.Image, ImageToPaletted(m, size, backgroundColor))
				}
				if export {
					// Write resized gif.
					var out *os.File
					out, err = os.Create(filepath.Join(targetDir, sizeName, fileName))
					if err != nil {
						return resizedImages, err
					}
					defer out.Close()
					gif.EncodeAll(out, outGif)
				} else {
					resizedImg := &Img{
						Name:        fileName,
						GIF:         outGif,
						SizeName:    sizeName,
						ContentType: imgType,
					}
					if resizedImages == nil {
						resizedImages = map[string]*Img{}
					}
					resizedImages[sizeName] = resizedImg
				}
			}
			return resizedImages, nil
		}
		return resizedImages, err
	} else {
		var ok bool
		if export {
			if ok, err = createFolderPath(targetDir); !ok {
				return resizedImages, err
			}
		}
		if targetSizes != nil {
			var img image.Image
			img, _, err = image.Decode(file)
			if err != nil {
				return resizedImages, err
			}
			for sizeName, size := range targetSizes {
				if export {
					if ok, err = createFolderPath(filepath.Join(targetDir, sizeName)); !ok {
						return resizedImages, err
					}
				}
				m := ResizeImgToClosestSizeOfTargetSize(img, size[0], size[1], resize.Lanczos3, withCrop)
				if size[0] > 0 && size[1] > 0 {
					if withCrop {
						m, _ = FitAspectRatioWithCroping(m, int(size[0]), int(size[1]), backgroundColor)
					} else {
						m = FitAspectRatioWithoutCroping(m, int(size[0]), int(size[1]), backgroundColor)
					}
				}
				if export {
					var out *os.File
					out, err = os.Create(filepath.Join(targetDir, sizeName, fileName))
					if err != nil {
						return resizedImages, err
					}
					defer out.Close()
					switch imgType {
					case "image/jpeg", "image/jpg":
						err = jpeg.Encode(out, m, nil)
					case "image/png":
						err = png.Encode(out, m)
					case "image/gif":
						err = gif.Encode(out, m, nil)
					}
				} else {
					resizedImg := &Img{
						Name:        fileName,
						Image:       m,
						SizeName:    sizeName,
						ContentType: imgType,
					}
					if resizedImages == nil {
						resizedImages = map[string]*Img{}
					}
					resizedImages[sizeName] = resizedImg
				}
			}
		}
		return resizedImages, nil
	}
	return resizedImages, err
}

func ResizeImgWithCroping(f os.FileInfo, sourceDir string, targetSizes map[string][]uint, opts ...string) (map[string]*Img, error) {
	var targetDir
	if opts != nil && len(opts) > 0{
		targetDir = opts[0]
	}
	return resizeImg(f, sourceDir, targetSizes, targetDir, true, nil)
}

func ResizeImgWithoutCroping(f os.FileInfo, sourceDir string, targetSizes map[string][]uint, opts ...interface) (map[string]*Img, error) {
	var targetDir string
	var backgroundColor *color.RGBA = &color.RGBA{R: 0, G: 0, B: 0, A: 255}
	if opts != nil && len(opts) > 0{
		targetDir = opts[0].(string)
		if len(opts) > 1{
			backgroundColor = opts[1].(*color.RGBA)
		}
	}
	return resizeImg(f, sourceDir, targetSizes, targetDir, false, nil)
}

func ResizeImgFromURL(u string, targetSize []uint, withCrop bool, backgroundColor *color.RGBA) (*Img, error) {
	var resizedImage *Img
	imgUrl, err := url.Parse(u)
	if err != nil {
		return resizedImage, err
	}
	_, imgData, errs := gorequest.New().Get(u).End()
	if errs != nil && len(errs) > 0 {
		return resizedImage, errs[0]
	}
	fileExtension := filepath.Ext(imgUrl.Path)
	fileExtension = strings.ToLower(fileExtension)

	isImageFile, imgType := isImageFile([]byte(imgData), fileExtension)

	if !isImageFile {
		err = errors.New("invalid_image_type")
		return resizedImage, err
	}

	if imgType == "image/gif" {
		if targetSize != nil && len(targetSize) > 1 {
			var img *gif.GIF
			b := bytes.NewBufferString(imgData)
			img, err = gif.DecodeAll(b)
			if err != nil {
				return resizedImage, err
			}
			outGif := &gif.GIF{}
			outGif.Delay = img.Delay

			// Create a new RGBA image to hold the incremental frames.
			firstFrame := img.Image[0]
			r := image.Rect(0, 0, firstFrame.Bounds().Dx(), firstFrame.Bounds().Dy())
			rgbaImg := image.NewRGBA(r)

			// Resize each frame.
			for _, frame := range img.Image {
				bounds := frame.Bounds()
				draw.Draw(rgbaImg, bounds, frame, bounds.Min, draw.Over)
				m := ResizeImgToClosestSizeOfTargetSize(rgbaImg, targetSize[0], targetSize[1], resize.Lanczos3, withCrop)
				if targetSize[0] > 0 && targetSize[1] > 0 {
					if withCrop {
						m, _ = FitAspectRatioWithCroping(m, int(targetSize[0]), int(targetSize[1]), backgroundColor)
					} else {
						m = FitAspectRatioWithoutCroping(m, int(targetSize[0]), int(targetSize[1]), backgroundColor)
					}
				}
				outGif.Image = append(outGif.Image, ImageToPaletted(m, targetSize, backgroundColor))
			}
			resizedImage = &Img{
				GIF:         outGif,
				ContentType: imgType,
			}
			return resizedImage, nil
		}
		return resizedImage, err
	} else {
		if targetSize != nil && len(targetSize) > 1 {
			var img image.Image
			b := bytes.NewBufferString(imgData)
			img, _, err = image.Decode(b)
			if err != nil {
				return resizedImage, err
			}
			m := ResizeImgToClosestSizeOfTargetSize(img, targetSize[0], targetSize[1], resize.Lanczos3, withCrop)
			if targetSize[0] > 0 && targetSize[1] > 0 {
				if withCrop {
					m, _ = FitAspectRatioWithCroping(m, int(targetSize[0]), int(targetSize[1]), backgroundColor)
				} else {
					m = FitAspectRatioWithoutCroping(m, int(targetSize[0]), int(targetSize[1]), backgroundColor)
				}
			}
			resizedImage = &Img{
				Image:       m,
				ContentType: imgType,
			}
		}
		return resizedImage, nil
	}
	return resizedImage, err
}

func ResizeImgToClosestSizeOfTargetSize(img image.Image, targetW uint, targetH uint, interp resize.InterpolationFunction, withCrop bool) image.Image {
	origBounds := img.Bounds()
	origWidth := float64(origBounds.Dx())
	origHeight := float64(origBounds.Dy())
	newWidth, newHeight := origWidth, origHeight

	targetHeight := float64(targetH)
	targetWidth := float64(targetW)

	if targetW > 0 && targetH > 0 {
		// Return original image if it have same size as constraints
		if targetWidth == origWidth && targetHeight == origHeight {
			return img
		}

		//if original height and width grater than target width and height scale img down
		if origWidth > targetWidth && origHeight > targetWidth {
			scale := origWidth / targetWidth
			origWidth /= scale
			origHeight /= scale
		}

		//if original height and width less than target width and height scale img up
		if origWidth < targetWidth && origHeight < targetWidth {
			scale := targetWidth / origWidth
			origWidth *= scale
			origHeight *= scale
		}

		if withCrop {
			if origWidth < targetWidth {
				//origWidth -> origHeight
				//targetWidth -> targetHeight
				newHeight = (origHeight * targetWidth) / origWidth
				newWidth = targetWidth

				if newHeight < targetHeight {
					//origWidth -> origHeight
					//targetWidth -> targetHeight
					newWidth = targetHeight * origWidth / origHeight
					newHeight = targetHeight
				}
			} else if origHeight < targetHeight { //375 < 400
				//origWidth -> origHeight
				//targetWidth -> targetHeight
				newWidth = (origWidth * targetHeight) / origHeight //500 * 400 / 375 = 533
				newHeight = targetHeight                           //400

				if newWidth < targetWidth { //533 > 500
					//origWidth -> origHeight
					//targetWidth -> targetHeight
					newHeight = targetWidth * origHeight / origWidth //500 * 375 / 500 = 375
					newWidth = targetWidth                           //500
				}
			} else {
				newHeight = origHeight
				newWidth = origWidth
			}
		} else {
			if origWidth > targetWidth {
				//origWidth -> origHeight
				//targetWidth -> targetHeight
				newHeight = (origHeight * targetWidth) / origWidth
				newWidth = targetWidth

				if newHeight > targetHeight {
					//origWidth -> origHeight
					//targetWidth -> targetHeight
					newWidth = targetHeight * origWidth / origHeight
					newHeight = targetHeight
				}
			} else if origHeight > targetHeight {
				//origWidth -> origHeight
				//targetWidth -> targetHeight
				newWidth = (origWidth * targetHeight) / origHeight
				newHeight = targetHeight

				if newWidth > targetWidth {
					//origWidth -> origHeight
					//targetWidth -> targetHeight
					newHeight = targetWidth * origHeight / origWidth
					newWidth = targetWidth
				}
			} else {
				newHeight = origHeight
				newWidth = origWidth
			}
		}
	} else {
		newHeight = float64(targetH)
		newWidth = float64(targetW)
	}
	return resize.Resize(uint(newWidth), uint(newHeight), img, interp)
}

func FitAspectRatioWithoutCroping(img image.Image, targetW int, targetH int, backgroundColor *color.RGBA) image.Image {
	imgBounds := img.Bounds()
	imgWidth := imgBounds.Dx()
	imgHeight := imgBounds.Dy()
	if imgWidth != targetW || imgHeight != targetH {
		r1 := image.Rectangle{image.Point{0, 0}, image.Point{targetW, targetH}}
		rgba := image.NewRGBA(r1)
		sp := image.Point{0, 0}
		if imgWidth != targetW {
			sp.X = (targetW - imgWidth) / 2
		}
		if imgHeight != targetH {
			sp.Y = (targetH - imgHeight) / 2
		}
		ep := image.Point{imgWidth, imgHeight}
		r2 := image.Rectangle{sp, sp.Add(ep)}
		if backgroundColor != nil {
			draw.Draw(rgba, rgba.Bounds(), &image.Uniform{*backgroundColor}, image.ZP, draw.Src)
			draw.Draw(rgba, r2, img, image.ZP, draw.Over)
		} else {
			draw.Draw(rgba, r2, img, image.ZP, draw.Src)
		}
		return rgba
	} else if backgroundColor != nil {
		r1 := image.Rectangle{image.Point{0, 0}, image.Point{targetW, targetH}}
		rgba := image.NewRGBA(r1)
		draw.Draw(rgba, rgba.Bounds(), &image.Uniform{*backgroundColor}, image.ZP, draw.Src)
		draw.Draw(rgba, r1, img, imgBounds.Min, draw.Over)
		return rgba
	}
	return img
}

func FitAspectRatioWithCroping(img image.Image, targetW int, targetH int, backgroundColor *color.RGBA) (image.Image, error) {
	m, err := cutter.Crop(img, cutter.Config{
		Width:  targetW,
		Height: targetH,
		Mode:   cutter.Centered,
	})
	if err != nil {
		return nil, err
	}
	if backgroundColor != nil {
		mBounds := m.Bounds()
		r1 := image.Rectangle{image.Point{0, 0}, image.Point{targetW, targetH}}
		rgba := image.NewRGBA(r1)
		draw.Draw(rgba, rgba.Bounds(), &image.Uniform{*backgroundColor}, image.ZP, draw.Src)
		draw.Draw(rgba, r1, m, mBounds.Min, draw.Over)
		return rgba, nil
	}
	return m, nil
}

func createFolderPath(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0777); err != nil {
			return false, err
		}
	}
	return true, err
}

func detectContentType(data []byte) string {
	if data != nil {
		if len(data) > 512 {
			data = data[:512]
		}
		filetype := http.DetectContentType(data)
		return filetype
	}
	return ""
}

func isImageFile(data []byte, fileExtension string) (bool, string) {
	isValidExtension := false
	isValidContentType := false
	imgType := ""
	for _, imageExtension := range imageExtensions {
		if imageExtension == fileExtension {
			isValidExtension = true
			break
		}
	}
	if isValidExtension {
		imgType = detectContentType(data)
		for _, imageContentType := range imageContentTypes {
			if imgType == imageContentType {
				isValidContentType = true
				break
			}
		}
	}
	return isValidContentType && isValidExtension, imgType
}

func ImageToPaletted(img image.Image, size []uint, backgroundColor *color.RGBA) *image.Paletted {
	if m, ok := img.(*image.Paletted); !ok {
		opts := gif.Options{
			NumColors: 256,
			Drawer:    draw.FloydSteinberg,
			Quantizer: vibrant.NewColorCutQuantizer(),
		}
		bounds := img.Bounds()
		pal := make(color.Palette, 0, 2)
		pal = append(pal, color.Transparent)
		if backgroundColor != nil {
			pal = append(pal, *backgroundColor)
		}
		r := image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: int(size[0]), Y: int(size[1])}}
		palettedImage := image.NewPaletted(r, pal)
		if opts.Quantizer != nil {
			qPal := opts.Quantizer.Quantize(make(color.Palette, 0, opts.NumColors), img)
			if len(qPal) == opts.NumColors {
				qPal[opts.NumColors-1] = color.Transparent
			} else {
				qPal = append(qPal, color.Transparent)
			}
			palettedImage.Palette = qPal
		} else {
			var k bool
			if pal, k = img.ColorModel().(color.Palette); !k {
				pal = palette.Plan9[:opts.NumColors]
				pal[opts.NumColors-1] = color.Transparent
				if backgroundColor != nil {
					pal[opts.NumColors-2] = *backgroundColor
				}
			}
			palettedImage.Palette = pal
		}
		opts.Drawer.Draw(palettedImage, palettedImage.Bounds(), img, bounds.Min)
		return palettedImage
	} else {
		return m
	}
}

func ImageDimensionsToPairNumbers(dimensions []uint) []uint {
	if dimensions != nil {
		for i, d := range dimensions {
			if d%2 != 0 {
				dimensions[i] = d - (d % 2)
			}
		}
		return dimensions
	}
	return nil
}
