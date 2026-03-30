package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	_ "image/png"
	"math/rand"
	"os"
	"time"

	xdraw "golang.org/x/image/draw"
)

type Config struct {
	InputPath   string
	OutputPath  string
	JPEGQuality int     // 1..100
	Scale       float64 // 0.1..1.0 (уменьшение и обратно)
	Noise       int     // 0..100 интенсивность шума
	Blur        bool    // простой blur
}

func main() {
	cfg := parseFlags()

	img, err := loadImage(cfg.InputPath)
	if err != nil {
		exitErr("ошибка загрузки изображения: %v", err)
	}

	// 1) Пикселизация через downscale/upscale
	if cfg.Scale < 1.0 {
		img = pixelate(img, cfg.Scale)
	}

	// 2) Добавление шума
	if cfg.Noise > 0 {
		img = addNoise(img, cfg.Noise)
	}

	// 3) Простейший box blur
	if cfg.Blur {
		img = boxBlur(img)
	}

	// 4) Сжатие JPEG с quality
	if err := saveJPEG(cfg.OutputPath, img, cfg.JPEGQuality); err != nil {
		exitErr("ошибка сохранения изображения: %v", err)
	}

	fmt.Printf("Готово: %s\n", cfg.OutputPath)
}

func parseFlags() Config {
	var cfg Config

	flag.StringVar(&cfg.InputPath, "in", "", "путь к входному изображению (jpg/png)")
	flag.StringVar(&cfg.OutputPath, "out", "out.jpg", "путь к выходному jpeg")
	flag.IntVar(&cfg.JPEGQuality, "quality", 40, "качество JPEG (1..100, меньше = хуже)")
	flag.Float64Var(&cfg.Scale, "scale", 0.5, "коэффициент пикселизации (0.1..1.0)")
	flag.IntVar(&cfg.Noise, "noise", 0, "интенсивность шума (0..100)")
	flag.BoolVar(&cfg.Blur, "blur", false, "включить простой blur")

	flag.Parse()

	if cfg.InputPath == "" {
		exitErr("обязательный параметр -in")
	}
	if cfg.JPEGQuality < 1 || cfg.JPEGQuality > 100 {
		exitErr("-quality должен быть в диапазоне 1..100")
	}
	if cfg.Scale < 0.1 || cfg.Scale > 1.0 {
		exitErr("-scale должен быть в диапазоне 0.1..1.0")
	}
	if cfg.Noise < 0 || cfg.Noise > 100 {
		exitErr("-noise должен быть в диапазон�� 0..100")
	}

	return cfg
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

func saveJPEG(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
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

	amp := intensity * 2 // амплитуда шума

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
