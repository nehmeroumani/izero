package izero

import "image/color"

type ImageSize struct {
	Name            string      `yaml:"name", json:"name"`
	Dimensions      []uint      `yaml:"dimensions", json:"dimensions"`
	Quality         int         `yaml:"quality", json:"quality"`         // 0 - 100
	ResizeType      string      `yaml:"resize_type", json:"resize_type"` // fit - fit_with_crop - closest
	BackgroundColor *color.RGBA `yaml:"background_color", json:"background_color"`
}

func isValidSize(size *ImageSize) error {
	if len(size.Dimensions) < 2 || size.Dimensions[0] == 0 || size.Dimensions[1] == 0 {
		return InvalidDimensions
	}
	return nil
}
