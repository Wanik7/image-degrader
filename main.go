package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	_ "image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

type Preset struct {
	Name        string
	JPEGQuality int
	Scale       float64
	Noise       int
	Blur        bool
}

func main() {
	inputPath := parseFlags()

	src, err := loadImage(inputPath)
	if err != nil {
		exitErr("ошибка загрузки изображения: %v", err)
	}

	baseName := baseFileName(inputPath) // без расширения
	if err := os.MkdirAll("output", 0o755); err != nil {
		exitErr("не удалось создать папку output: %v", err)
	}

	zipPath := filepath.Join("output", baseName+".zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		exitErr("не удалось создать zip архив: %v", err)
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	presets := []Preset{
		{
			Name:        "easy",
			JPEGQuality: 65,
			Scale:       0.85,
			Noise:       5,
			Blur:        false,
		},
		{
			Name:        "normal",
			JPEGQuality: 40,
			Scale:       0.60,
			Noise:       18,
			Blur:        true,
		},
		{
			Name:        "hard",
			JPEGQuality: 18,
			Scale:       0.35,
			Noise:       35,
			Blur:        true,
		},
	}

	for _, p := range presets {
		out := applyPreset(src, p)

		jpgBytes, err := encodeJPEGToBytes(out, p.JPEGQuality)
		if err != nil {
			exitErr("ошибка кодирования пресета %s: %v", p.Name, err)
		}

		fileNameInZip := fmt.Sprintf("%s_%s.jpg", baseName, p.Name)
		w, err := zw.Create(fileNameInZip)
		if err != nil {
			exitErr("ошибка добавления файла %s в zip: %v", fileNameInZip, err)
		}

		if _, err := w.Write(jpgBytes); err != nil {
			exitErr("ошибка записи файла %s в zip: %v", fileNameInZip, err)
		}
	}

	if err := zw.Close(); err != nil {
		exitErr("ошибка закрытия zip: %v", err)
	}

	fmt.Printf("Готово! Архив создан: %s\n", zipPath)
}

func parseFlags() string {
	var inputPath string
	flag.StringVar(&inputPath, "in", "", "путь к входному изображению (jpg/png)")
	flag.Parse()

	if strings.TrimSpace(inputPath) == "" {
		exitErr("обязательный параметр -in")
	}
	return inputPath
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func baseFileName(path string) string {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext)
}

func applyPreset(src image.Image, p Preset) image.Image {
	img := src
	if p.Scale < 1.0 {
		img = pixelate(img, p.Scale)
	}
	if p.Noise > 0 {
		img = addNoise(img, p.Noise)
	}
	if p.Blur {
		img = boxBlur(img)
	}
	return img
}

func encodeJPEGToBytes(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pixelate(src image.Image, scale float64) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	dw := int(float64(w) * scale)
	dh := int(float64(h) * scale)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}

	down := image.NewRGBA(image.Rect(0, 0, dw, dh))
	xdraw.NearestNeighbor.Scale(down, down.Bounds(), src, b, stddraw.Src, nil)

	up := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.NearestNeighbor.Scale(up, up.Bounds(), down, down.Bounds(), stddraw.Src, nil)

	return up
}

func addNoise(src image.Image, intensity int) image.Image {
	rand.Seed(time.Now().UnixNano())

	b := src.Bounds()
	dst := image.NewRGBA(b)
	stddraw.Draw(dst, b, src, b.Min, stddraw.Src)

	amp := intensity * 2
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := dst.At(x, y).RGBA()
			rr := clamp8(int(r>>8) + rand.Intn(amp+1) - amp/2)
			gg := clamp8(int(g>>8) + rand.Intn(amp+1) - amp/2)
			bb := clamp8(int(bl>>8) + rand.Intn(amp+1) - amp/2)
			dst.Set(x, y, color.RGBA{uint8(rr), uint8(gg), uint8(bb), uint8(a >> 8)})
		}
	}
	return dst
}

func boxBlur(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var rSum, gSum, bSum, aSum, count int
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					nx, ny := x+kx, y+ky
					if nx < b.Min.X || nx >= b.Max.X || ny < b.Min.Y || ny >= b.Max.Y {
						continue
					}
					r, g, bl, a := src.At(nx, ny).RGBA()
					rSum += int(r >> 8)
					gSum += int(g >> 8)
					bSum += int(bl >> 8)
					aSum += int(a >> 8)
					count++
				}
			}
			dst.Set(x, y, color.RGBA{
				uint8(rSum / count),
				uint8(gSum / count),
				uint8(bSum / count),
				uint8(aSum / count),
			})
		}
	}
	return dst
}

func clamp8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
