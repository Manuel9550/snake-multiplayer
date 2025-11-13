package internal

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

var PreMadeLayouts []PreMadeLayout = nil

type PreMadeLayout struct {
	Name  string
	Walls []Location
}

// Only should be called once.
// If the premade layouts weren't initialized, the slice will be nil
// If it has (or errored out while being initialized) it won't be nil
func InitializePremadelayouts() ([]PreMadeLayout, error) {
	premadeLayouts := make([]PreMadeLayout, 0)
	dirPath := "./images"
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading directory: %v\n", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext == ".png" {
				// Try to open and parse this file
				fullFileLocation := fmt.Sprintf("%s/%s", dirPath, file.Name())
				foundFile, err := os.Open(fullFileLocation)
				if err != nil {
					fmt.Printf("Error opening file %s: %v\n", file.Name(), err)
					return nil, err
				}
				defer foundFile.Close()

				newLocations, err := ParseFile(foundFile)
				if err != nil {
					fmt.Printf("Error parsing file %s: %v\n", file.Name(), err)
					return nil, err
				}

				premadeLayouts = append(premadeLayouts, PreMadeLayout{
					Name:  file.Name(),
					Walls: newLocations,
				})

			}
		}
	}

	return premadeLayouts, nil
}

func ParseFile(file *os.File) ([]Location, error) {
	returnLocations := []Location{}
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Printf("Error decoding image: %v\n", err)
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get the color of the pixel at (x, y)
			color := img.At(x, y)

			// Extract RGBA values (uint32, need to be converted to uint8 for typical 0-255 range)
			r, g, b, _ := color.RGBA()

			// We know there's only two values in the image: White or black
			// Black RGBA value is 0,0,0,1
			if r == 0 && g == 0 && b == 0 {
				returnLocations = append(returnLocations, Location{X: x, Y: y})
			}
		}
	}

	return returnLocations, nil
}
