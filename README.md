# Image Degrader (Go CLI)

A simple Go command-line app that takes an input image and creates a ZIP archive with 3 degraded versions:

- `easy`
- `normal`
- `hard`

## What the app does

1. Reads an input image (`jpg/png`).
2. Generates 3 degraded variants.
3. Packs them into a ZIP archive.
4. Saves the archive to the `output` folder.

## Output naming

If the input file is `cat.png`, the app creates:

- archive: `output/cat.zip`
- files inside the archive:
    - `cat_easy.jpg`
    - `cat_normal.jpg`
    - `cat_hard.jpg`

## Requirements

- Go 1.20+ (recommended)

## Install dependencies

```bash
go mod init img-degrader
go get golang.org/x/image/draw
```

## Run

```bash
go run . -in ./input/cat.png // you may use any other path to image
```

Arguments:

- `-in` — path to the input image (required)

Result:

- `./output/photo.zip`

## Project structure

```text
.
├── main.go
├── go.mod
├── go.sum
└── output/
```

