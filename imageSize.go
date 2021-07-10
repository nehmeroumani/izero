package izero

import "image/color"

type ImageSize struct {
	Name       string      `yaml:"name" json:"name"`
	Dimensions []uint      `yaml:"dimensions" json:"dimensions"`
	Mode       string      `yaml:"mode" json:"mode"`
	Quality    int         `yaml:"quality" json:"quality"`
	Background *color.RGBA `yaml:"background" json:"background"`
}

func isValidSize(size *ImageSize) error {
	if size.Mode == "fit_width" || size.Mode == "fit_height" {
		if len(size.Dimensions) < 2 {
			return InvalidDimensions
		}
		if size.Mode == "fit_width" && (size.Dimensions[0] <= 0 || size.Dimensions[1] > 0) {
			return InvalidDimensions
		}
		if size.Mode == "fit_height" && (size.Dimensions[0] > 0 || size.Dimensions[1] <= 0) {
			return InvalidDimensions
		}
	} else {
		if len(size.Dimensions) < 2 || size.Dimensions[0] <= 0 || size.Dimensions[1] <= 0 {
			return InvalidDimensions
		}
	}
	return nil
}
