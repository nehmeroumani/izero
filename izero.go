package izero

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"
	"sync"

	"github.com/RobCherry/vibrant"
	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
)

func ResizeImage(imageFile io.Reader, imageName string, imageType string, targetSizes []*ImageSize, destPath string) (map[string]*ResizedImage, map[string]error, error) {
	if imageFile != nil && imageName != "" && targetSizes != nil {
		resizedImages := map[string]*ResizedImage{}
		var wg sync.WaitGroup
		errs := map[string]error{}
		if imageType == "image/gif" {
			gifImg, err := gif.DecodeAll(imageFile)
			if err != nil {
				return nil, nil, err
			}
			wg.Add(len(targetSizes))
			for _, imgSize := range targetSizes {
				go func(imgSize *ImageSize) {
					defer wg.Done()
					if resizedImg, err := resizeDynamicImg(gifImg, imageName, imageType, imgSize, destPath); err == nil {
						resizedImages[imgSize.Name] = resizedImg
					} else {
						errs[imgSize.Name] = err
					}
				}(imgSize)
			}
		} else {
			img, _, err := image.Decode(imageFile)
			if err != nil {
				return nil, nil, err
			}
			wg.Add(len(targetSizes))
			for _, imgSize := range targetSizes {
				go func(imgSize *ImageSize) {
					defer wg.Done()
					if resizedImg, err := resizeStaticImg(img, imageName, imageType, imgSize, destPath); err == nil {
						resizedImages[imgSize.Name] = resizedImg
					} else {
						errs[imgSize.Name] = err
					}

				}(imgSize)
			}
		}
		wg.Wait()
		var err error
		if len(errs) > 0 {
			err = ResizeFailed
		} else {
			errs = nil
		}
		return resizedImages, errs, err
	}
	return nil, nil, InvalidData
}

func resizeStaticImg(img image.Image, imageName string, imageType string, imgSize *ImageSize, destPath string) (*ResizedImage, error) {
	if err := isValidSize(imgSize); err != nil {
		return nil, err
	}
	m := ResizeImgToClosestSizeOfTargetSize(img, imgSize, resize.Lanczos3)
	if imgSize.Mode == "fit_with_crop" {
		m, _ = FitAspectRatioWithCroping(m, imgSize)
	} else if imgSize.Mode == "fit" {
		m = FitAspectRatioWithoutCroping(m, imgSize)
	}

	resizedImg := &ResizedImage{
		Name:        imageName,
		StaticImage: m,
		ContentType: imageType,
		Size:        imgSize,
	}
	if destPath != "" {
		if err := resizedImg.saveTo(destPath); err != nil {
			return nil, err
		}
	}
	return resizedImg, nil
}

func resizeDynamicImg(gifImg *gif.GIF, imageName string, imageType string, imgSize *ImageSize, destPath string) (*ResizedImage, error) {
	if err := isValidSize(imgSize); err != nil {
		return nil, err
	}

	outGif := &gif.GIF{}
	outGif.Delay = gifImg.Delay

	// Create a new RGBA image to hold the incremental frames.
	firstFrame := gifImg.Image[0]
	r := image.Rect(0, 0, firstFrame.Bounds().Dx(), firstFrame.Bounds().Dy())
	rgbaImg := image.NewRGBA(r)

	// Resize each frame.
	for _, frame := range gifImg.Image {
		bounds := frame.Bounds()
		draw.Draw(rgbaImg, bounds, frame, bounds.Min, draw.Over)
		m := ResizeImgToClosestSizeOfTargetSize(rgbaImg, imgSize, resize.Lanczos3)
		if imgSize.Mode == "fit_with_crop" {
			m, _ = FitAspectRatioWithCroping(m, imgSize)
		} else if imgSize.Mode == "fit" {
			m = FitAspectRatioWithoutCroping(m, imgSize)
		}
		outGif.Image = append(outGif.Image, ImageToPaletted(m, imgSize))
	}
	resizedImg := &ResizedImage{
		Name:         imageName,
		DynamicImage: outGif,
		ContentType:  imageType,
		Size:         imgSize,
	}
	if destPath != "" {
		if err := resizedImg.saveTo(destPath); err != nil {
			return nil, err
		}
	}
	return resizedImg, nil

}

func ResizeImgToClosestSizeOfTargetSize(img image.Image, imgSize *ImageSize, interp resize.InterpolationFunction) image.Image {
	origBounds := img.Bounds()
	origWidth := float64(origBounds.Dx())
	origHeight := float64(origBounds.Dy())
	newWidth, newHeight := origWidth, origHeight

	targetWidth := float64(imgSize.Dimensions[0])
	targetHeight := float64(imgSize.Dimensions[1])

	if targetWidth > 0 && targetHeight > 0 {
		// Return th3e original image if it have the same size as constraints
		if targetWidth == origWidth && targetHeight == origHeight {
			return img
		}

		//if the original height and width are grater than the target width and height then scale the image down
		if origWidth > targetWidth && origHeight > targetWidth {
			scale := origWidth / targetWidth
			origWidth /= scale
			origHeight /= scale
		}

		//if the original height and width are less than the target width and height then scale image up
		if origWidth < targetWidth && origHeight < targetWidth {
			scale := targetWidth / origWidth
			origWidth *= scale
			origHeight *= scale
		}

		if imgSize.Mode == "fit_with_crop" {
			if origWidth < targetWidth {
				//origWidth -> targetWidth
				//origHeight -> (newHeight)
				newHeight = (origHeight * targetWidth) / origWidth
				newWidth = targetWidth

				if newHeight < targetHeight {
					//origWidth -> (newWidth)
					//origHeight -> targetHeight
					newWidth = targetHeight * origWidth / origHeight
					newHeight = targetHeight
				}
			} else if origHeight < targetHeight {
				//origWidth -> (newWidth)
				//origHeight -> targetHeight
				newWidth = (origWidth * targetHeight) / origHeight
				newHeight = targetHeight

				if newWidth < targetWidth {
					//origWidth -> targetWidth
					//origHeight -> (newHeight)
					newHeight = targetWidth * origHeight / origWidth
					newWidth = targetWidth
				}
			} else {
				newHeight = origHeight
				newWidth = origWidth
			}
		} else {
			if origWidth > targetWidth {
				//origWidth -> targetWidth
				//origHeight -> (newHeight)
				newHeight = (origHeight * targetWidth) / origWidth
				newWidth = targetWidth

				if newHeight > targetHeight {
					//origWidth -> (newWidth)
					//origHeight -> targetHeight
					newWidth = targetHeight * origWidth / origHeight
					newHeight = targetHeight
				}
			} else if origHeight > targetHeight {
				//origWidth -> (newWidth)
				//origHeight -> targetHeight
				newWidth = (origWidth * targetHeight) / origHeight
				newHeight = targetHeight

				if newWidth > targetWidth {
					//origWidth -> targetWidth
					//origHeight -> (newHeight)
					newHeight = targetWidth * origHeight / origWidth
					newWidth = targetWidth
				}
			} else {
				newHeight = origHeight
				newWidth = origWidth
			}
		}
	}
	return resize.Resize(uint(newWidth), uint(newHeight), img, interp)
}

func FitAspectRatioWithoutCroping(img image.Image, imgSize *ImageSize) image.Image {
	imgBounds := img.Bounds()
	imgWidth := imgBounds.Dx()
	imgHeight := imgBounds.Dy()
	targetW := int(imgSize.Dimensions[0])
	targetH := int(imgSize.Dimensions[1])
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
		if imgSize.Background != nil {
			draw.Draw(rgba, rgba.Bounds(), &image.Uniform{*imgSize.Background}, image.ZP, draw.Src)
			draw.Draw(rgba, r2, img, image.ZP, draw.Over)
		} else {
			draw.Draw(rgba, r2, img, image.ZP, draw.Src)
		}
		return rgba
	} else if imgSize.Background != nil {
		r1 := image.Rectangle{image.Point{0, 0}, image.Point{targetW, targetH}}
		rgba := image.NewRGBA(r1)
		draw.Draw(rgba, rgba.Bounds(), &image.Uniform{*imgSize.Background}, image.ZP, draw.Src)
		draw.Draw(rgba, r1, img, imgBounds.Min, draw.Over)
		return rgba
	}
	return img
}

func FitAspectRatioWithCroping(img image.Image, imgSize *ImageSize) (image.Image, error) {
	targetW := int(imgSize.Dimensions[0])
	targetH := int(imgSize.Dimensions[1])
	m, err := cutter.Crop(img, cutter.Config{
		Width:  targetW,
		Height: targetH,
		Mode:   cutter.Centered,
	})
	if err != nil {
		return nil, err
	}
	if imgSize.Background != nil {
		mBounds := m.Bounds()
		r1 := image.Rectangle{image.Point{0, 0}, image.Point{targetW, targetH}}
		rgba := image.NewRGBA(r1)
		draw.Draw(rgba, rgba.Bounds(), &image.Uniform{*imgSize.Background}, image.ZP, draw.Src)
		draw.Draw(rgba, r1, m, mBounds.Min, draw.Over)
		return rgba, nil
	}
	return m, nil
}

func ImageToPaletted(img image.Image, imgSize *ImageSize) *image.Paletted {
	if m, ok := img.(*image.Paletted); !ok {
		opts := gif.Options{
			NumColors: 256,
			Drawer:    draw.FloydSteinberg,
			Quantizer: vibrant.NewColorCutQuantizer(),
		}
		bounds := img.Bounds()
		pal := make(color.Palette, 0, 2)
		pal = append(pal, color.Transparent)
		if imgSize.Background != nil {
			pal = append(pal, *imgSize.Background)
		}
		r := image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: int(imgSize.Dimensions[0]), Y: int(imgSize.Dimensions[1])}}
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
				if imgSize.Background != nil {
					pal[opts.NumColors-2] = *imgSize.Background
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
