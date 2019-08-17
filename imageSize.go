package izero

import "image/color"

type ImageSize struct {
	Name            string      `yaml:"name", json:"name"`
	Dimensions      []uint      `yaml:"dimensions", json:"dimensions"`
	Mode            string      `yaml:"mode", json:"mode"`
	Quality         int         `yaml:"quality", json:"quality"`
	BackgroundColor *color.RGBA `yaml:"background_color", json:"background_color"`
}

func isValidSize(size *ImageSize) error {
	if len(size.Dimensions) < 2 || size.Dimensions[0] == 0 || size.Dimensions[1] == 0 {
		return InvalidDimensions
	}
	return nil
}
