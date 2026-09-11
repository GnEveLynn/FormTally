package storage

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	_ "golang.org/x/image/webp"
)

type ImageError struct{ Code string }

func (e *ImageError) Error() string { return e.Code }
func IsImageError(err error, code string) bool {
	var target *ImageError
	return errors.As(err, &target) && target.Code == code
}

type ProcessedImage struct {
	Bytes       []byte
	ContentType string
	Width       int
	Height      int
}

func ProcessImage(reader io.Reader, maxBytes int64, maxPixels int64) (ProcessedImage, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return ProcessedImage{}, err
	}
	if int64(len(data)) > maxBytes {
		return ProcessedImage{}, &ImageError{Code: "IMAGE_TOO_LARGE"}
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return ProcessedImage{}, &ImageError{Code: "IMAGE_UNSUPPORTED"}
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, &ImageError{Code: "IMAGE_UNSUPPORTED"}
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxPixels {
		return ProcessedImage{}, &ImageError{Code: "IMAGE_DIMENSIONS_TOO_LARGE"}
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, &ImageError{Code: "IMAGE_UNSUPPORTED"}
	}
	var output bytes.Buffer
	if err := jpeg.Encode(&output, decoded, &jpeg.Options{Quality: 88}); err != nil {
		return ProcessedImage{}, fmt.Errorf("encode image: %w", err)
	}
	return ProcessedImage{Bytes: output.Bytes(), ContentType: "image/jpeg", Width: config.Width, Height: config.Height}, nil
}
