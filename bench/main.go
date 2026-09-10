package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"fractal-gen/internal/fractal"
	"fractal-gen/internal/renderer"
)

func renderToPNG(key string, v renderer.View, size int) error {
	reg := fractal.NewRegistry()
	f := reg.Create(key)
	pal := fractal.DefaultPalettes()["Bloom"]
	r := renderer.NewRenderer(pal, f)
	r.MaxIter = 700

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	v.Aspect = 1

	start := time.Now()
	r.RenderTo(img, v)
	elapsed := time.Since(start)

	path := "bench_" + key + ".png"
	fo, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fo.Close()
	png.Encode(fo, img)
	fmt.Printf("%-12s %5d px  %8s  %6.1f fps-equiv\n", key, size, elapsed.Round(time.Millisecond), 1e9/float64(elapsed.Nanoseconds()))
	return nil
}

func main() {
	views := map[string]renderer.View{
		"mandelbrot":   {CenterX: -0.55, CenterY: 0, HalfWidth: 1.7, Aspect: 1},
		"julia":        {CenterX: 0, CenterY: 0, HalfWidth: 1.5, Aspect: 1},
		"burning_ship": {CenterX: -0.4, CenterY: -0.5, HalfWidth: 1.6, Aspect: 1},
		"newton":       {CenterX: 0, CenterY: 0, HalfWidth: 1.6, Aspect: 1},
		"tricorn":      {CenterX: 0, CenterY: 0, HalfWidth: 2.2, Aspect: 1},
		"phoenix":      {CenterX: 0, CenterY: 0, HalfWidth: 1.4, Aspect: 1},
	}
	for key, v := range views {
		if err := renderToPNG(key, v, 800); err != nil {
			fmt.Println("ERROR", key, err)
		}
	}
}
