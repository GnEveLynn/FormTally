package storage

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestProcessImageAcceptsJPEGAndPNGAndStripsMetadata(t *testing.T) {
	for _, format := range []string{"jpeg", "png"} {
		t.Run(format, func(t *testing.T) {
			input := encodedImage(t, format, 4, 3)
			processed, err := ProcessImage(bytes.NewReader(input), int64(len(input)+1), 100)
			if err != nil {
				t.Fatal(err)
			}
			if processed.ContentType != "image/jpeg" || processed.Width != 4 || processed.Height != 3 {
				t.Fatalf("processed = %#v", processed)
			}
			if bytes.Contains(processed.Bytes, []byte("EXIF")) {
				t.Fatal("processed image retained metadata marker")
			}
		})
	}
}

func TestProcessImageDecodesWebP(t *testing.T) {
	encoded := "UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA=="
	input, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	processed, err := ProcessImage(bytes.NewReader(input), int64(len(input)+1), 100_000)
	if err != nil {
		t.Fatal(err)
	}
	if processed.ContentType != "image/jpeg" || processed.Width < 1 || processed.Height < 1 {
		t.Fatalf("processed = %#v", processed)
	}
}

func TestProcessImageRejectsUnsupportedOversizeAndTooManyPixels(t *testing.T) {
	if _, err := ProcessImage(bytes.NewReader([]byte("not an image")), 100, 100); !IsImageError(err, "IMAGE_UNSUPPORTED") {
		t.Fatalf("unsupported error = %v", err)
	}
	input := encodedImage(t, "png", 4, 3)
	if _, err := ProcessImage(bytes.NewReader(input), int64(len(input)-1), 100); !IsImageError(err, "IMAGE_TOO_LARGE") {
		t.Fatalf("oversize error = %v", err)
	}
	if _, err := ProcessImage(bytes.NewReader(input), int64(len(input)+1), 11); !IsImageError(err, "IMAGE_DIMENSIONS_TOO_LARGE") {
		t.Fatalf("pixel error = %v", err)
	}
}

func encodedImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 220, G: 80, B: 30, A: 255})
	var out bytes.Buffer
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&out, img, nil)
	} else {
		err = png.Encode(&out, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
