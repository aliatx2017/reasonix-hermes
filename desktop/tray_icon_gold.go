package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"
)

var (
	goldIconOnce sync.Once
	goldIconPNG  []byte
)

// goldTrayIcon returns the default tray icon tinted with Hermes gold (#d4a853).
func goldTrayIcon() []byte {
	goldIconOnce.Do(func() {
		img, err := png.Decode(bytes.NewReader(trayIconBytes))
		if err != nil {
			goldIconPNG = trayIconBytes
			return
		}
		bounds := img.Bounds()
		tinted := image.NewRGBA(bounds)
		draw.Draw(tinted, bounds, img, bounds.Min, draw.Src)

		// Overlay with Hermes gold at 30% opacity.
		gold := color.RGBA{R: 0xd4, G: 0xa8, B: 0x53, A: 77} // 0.3 * 255 ≈ 77
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				c := tinted.RGBAAt(x, y)
				// Alpha-blend gold overlay.
				alpha := uint32(gold.A)
				invAlpha := 255 - alpha
				c.R = uint8((uint32(c.R)*invAlpha + uint32(gold.R)*alpha) / 255)
				c.G = uint8((uint32(c.G)*invAlpha + uint32(gold.G)*alpha) / 255)
				c.B = uint8((uint32(c.B)*invAlpha + uint32(gold.B)*alpha) / 255)
				tinted.SetRGBA(x, y, c)
			}
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, tinted); err != nil {
			goldIconPNG = trayIconBytes
			return
		}
		goldIconPNG = buf.Bytes()
	})
	return goldIconPNG
}
