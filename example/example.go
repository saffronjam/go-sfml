package main

import (
	"github.com/saffronjam/go-sfml/public/sfml"
	"runtime"
	"time"
)

func init() { runtime.LockOSThread() }

func main() {
	wnd := sfml.NewRenderWindow(sfml.VideoMode{
		Width:        800,
		Height:       600,
		BitsPerPixel: 32,
	}, "my window", uint32(sfml.Titlebar)|uint32(sfml.Close), &sfml.ContextSettings{
		DepthBits:         0,
		StencilBits:       0,
		AntialiasingLevel: 0,
		MajorVersion:      0,
		MinorVersion:      0,
		AttributeFlags:    0,
		SRgbCapable:       false,
	})
	asNormalWindow := sfml.NewWindowFromHandle(wnd.SystemHandle(), &sfml.ContextSettings{
		DepthBits:         0,
		StencilBits:       0,
		AntialiasingLevel: 0,
		MajorVersion:      0,
		MinorVersion:      0,
		AttributeFlags:    0,
		SRgbCapable:       false,
	})

	rect := sfml.NewRectangleShape()
	rect.SetFillColor(sfml.Color{R: 255, G: 255, B: 255, A: 255})
	rect.SetSize(sfml.Vector2f{X: 100, Y: 100})

	circle := sfml.NewCircleShape()
	circle.SetRadius(10)
	circle.SetFillColor(sfml.Color{R: 255, G: 0, B: 0})
	circle.SetPosition(sfml.Vector2f{X: 400, Y: 300})
	circle.SetOrigin(sfml.Vector2f{X: 10, Y: 10})

	for wnd.IsOpen() {
		position := wnd.Position()

		event, hasMore := wnd.PollEvent()
		if hasMore {
			switch event.EventType() {
			case sfml.EvtKeyPressed:
				keyEvent := event.(*sfml.KeyEvent)
				if keyEvent.Code == sfml.KeyA {
					circle.SetFillColor(sfml.Color{R: 0, G: 100, B: 0, A: 255})
				}
			case sfml.EvtKeyReleased:
				keyEvent := event.(*sfml.KeyEvent)
				if keyEvent.Code == sfml.KeyA {
					circle.SetFillColor(sfml.Color{R: 255, G: 0, B: 0, A: 255})
				}
			case sfml.EvtClosed:
				wnd.Close()
				continue
			case sfml.EvtMouseEntered:
				circle.SetFillColor(sfml.Color{R: 0, G: 255, B: 0, A: 255})
			case sfml.EvtMouseLeft:
				circle.SetFillColor(sfml.Color{R: 255, G: 0, B: 0, A: 255})
			}
		}

		pos := sfml.MouseGetPosition(asNormalWindow)
		rect.SetPosition(sfml.Vector2f{X: float32(pos.X), Y: float32(pos.Y)})
		wnd.Clear(sfml.Color{R: uint8(position.X % 255), G: uint8(position.Y % 255), B: 0, A: 255})

		wnd.DrawRectangleShape(rect, sfml.RenderStatesDefault())
		wnd.DrawCircleShape(circle, sfml.RenderStatesDefault())

		wnd.Display()
	}

	time.Sleep(5 * time.Second)
}
