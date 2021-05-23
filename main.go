package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/tmhmitchell/ebitoolbox/datastructures/vector"
)

var ErrGameExit = errors.New("Game exited correctly")

const (
	screenWidth  = 1000
	screenHeight = 1000
)

var (
	colorRed    = color.RGBA{255, 0, 0, 255}
	colorGreen  = color.RGBA{0, 255, 0, 255}
	colorBlue   = color.RGBA{0, 0, 255, 255}
	colorYellow = color.RGBA{255, 255, 0, 255}
	colorPurple = color.RGBA{255, 0, 255, 255}
)

type GameMode string

const (
	GameModePoint GameMode = "point"
	GameModeRay   GameMode = "ray"
	GameModeRay2  GameMode = "ray2"
	GameModeRect  GameMode = "rect"
)

type Rect struct {
	x, y, w, h float64
	// inCollision bool
}

func (r Rect) X() float64      { return r.x }
func (r Rect) Y() float64      { return r.y }
func (r Rect) Width() float64  { return r.w }
func (r Rect) Height() float64 { return r.h }

// func (r Rect) InCollision() bool      { return r.inCollision }
// func (r *Rect) SetInCollision(c bool) { r.inCollision = c }

// Returns a Rect instance with a fixec size of 50x50
func NewFixedSizeRect(x, y float64) *Rect {
	return &Rect{x, y, 50, 50}
}

type Game struct {
	// Indicates which collision mode we're in
	mode GameMode

	// User in all modes
	cursorVector  vector.Vec2
	terrain       []*Rect
	collisionData map[*Rect]CollisionData

	// Used in ray/ray2 mode
	rayOrigin  vector.Vec2
	ray2Origin vector.Vec2

	// Used in box mode
	movingRect Rect
}

func NewGame() *Game {
	// Define a load of static rectangles formed into some sort of "complex terrain"
	statics := make([]*Rect, 0)
	{
		offset := 50

		// Top row
		for x := 200; x < 700; x += offset {
			statics = append(statics, NewFixedSizeRect(float64(x), 200))
		}

		// Left column
		for y := 250; y < 700; y += offset {
			statics = append(statics, NewFixedSizeRect(200, float64(y)))
		}

		// Bottom row
		for x := 250; x < 700; x += offset {
			statics = append(statics, NewFixedSizeRect(float64(x), 650))
		}

		// Top part of the right column
		for y := 250; y < 400; y += offset {
			statics = append(statics, NewFixedSizeRect(650, float64(y)))
		}

		// Bottom part of the right column
		for y := 500; y < 650; y += offset {
			statics = append(statics, NewFixedSizeRect(650, float64(y)))
		}

		// Central column
		for y := 300; y < 500; y += offset {
			statics = append(statics, NewFixedSizeRect(400, float64(y)))
		}
	}

	return &Game{
		mode:       GameModePoint,
		terrain:    statics,
		rayOrigin:  vector.NewVec2(50, 100),
		ray2Origin: vector.NewVec2(900, 800),
		movingRect: Rect{50, 50, 75, 75},
	}
}

// type rectPtrCollisionTimePair struct {
// 	time float64
// }

func (g *Game) Update() error {
	// Close the game when escape is pressed
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ErrGameExit
	}

	// Cycle to the next game mode when space is pressed
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch g.mode {
		case GameModePoint:
			g.mode = GameModeRay
		case GameModeRay:
			g.mode = GameModeRay2
		case GameModeRay2:
			g.mode = GameModeRect
		case GameModeRect:
			g.mode = GameModePoint
		}
	}

	// Reset the game's cursor vector
	{
		cx, cy := ebiten.CursorPosition()
		g.cursorVector = vector.NewVec2(float64(cx), float64(cy))
	}

	// Reset our knowledge of what's colliding
	g.collisionData = make(map[*Rect]CollisionData)

	// Perform the current mode's collision test
	switch g.mode {
	case GameModePoint:
		for _, r := range g.terrain {
			// We have to mis-use the g.collisionData map a little bit here
			// Because there's no data other than "yes/no" for point/AABB
			// collision, we'll just use the presence of a key as an indicator
			// of collision - the values will just be "default" structs
			if PointInAABB(g.cursorVector, r) {
				g.collisionData[r] = CollisionData{}
			}
		}

	case GameModeRay, GameModeRay2:
		// Pick an origin based on which of the two ray modes we're in
		var ro vector.Vec2
		if g.mode == GameModeRay {
			ro = g.rayOrigin
		} else {
			ro = g.ray2Origin
		}

		// Determine the ray's direction
		rd := vector.Vec2(g.cursorVector.Minus(ro))

		// Determine collisions
		for _, r := range g.terrain {
			if colliding, data := RayVsRect(ro, rd, r); colliding {
				g.collisionData[r] = data
			}
		}

	case GameModeRect:
		// We only move when the LMB is pressed
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			break
		}

		// Determine how the game's movingRect would __like__ to move this tick
		// We consider this a candidate because it will be refined as we resolve
		// any potential collisions the origin movement would cause
		var cmv vector.Vec2
		{
			lerpSpeed := 0.05
			cmv.SetX((g.cursorVector.X() - (g.movingRect.Width() / 2) - g.movingRect.X()) * lerpSpeed)
			cmv.SetY((g.cursorVector.Y() - (g.movingRect.Height() / 2) - g.movingRect.Y()) * lerpSpeed)
		}

		// Determine all of the tiles moving by mv would bring us into collision with

		collisions := make([]CollisionData, 0)
		for _, r := range g.terrain {
			colliding, data := MovingRectVsRect(g.movingRect, cmv, r)
			if !colliding {
				continue
			}

			g.collisionData[r] = data
			collisions = append(collisions, data)
		}

		sort.Slice(collisions, func(i, j int) bool {
			return collisions[i].Time < collisions[j].Time
		})

		if len(collisions) == 0 {
			g.movingRect.x += cmv.X()
			g.movingRect.y += cmv.Y()
			break
		}

		c := collisions[0]

		mv := vector.NewVec2(
			cmv.X()*c.Time,
			cmv.Y()*c.Time,
		)

		r := 1.0 - c.Time

		sdp := (cmv.X() * c.Normal.Y()) + (cmv.Y()*c.Normal.X())*r

		mv = vector.NewVec2(
			(cmv.X()*c.Time)+(sdp*c.Normal.Y()),
			(cmv.Y()*c.Time)+(sdp*c.Normal.X()),
		)

		g.movingRect.x += mv.X()
		g.movingRect.y += mv.Y()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// In all modes, draw all of the static rects
	{
		for _, r := range g.terrain {
			var chosenColor color.RGBA

			if _, ok := g.collisionData[r]; ok {
				chosenColor = colorGreen
			} else {
				chosenColor = colorRed
			}

			ebitenutil.DrawRect(
				screen,
				r.X(),
				r.Y(),
				r.Width(),
				r.Height(),
				chosenColor,
			)
		}
	}

	// If we're in either ray mode, visualise the ray as well as the normal
	// vectors of any faces in collision with the ray
	if g.mode == GameModeRay || g.mode == GameModeRay2 {
		var ro vector.Vec2
		if g.mode == GameModeRay {
			ro = g.rayOrigin
		} else {
			ro = g.ray2Origin
		}

		ebitenutil.DrawLine(
			screen,
			ro.X(), ro.Y(),
			g.cursorVector.X(), g.cursorVector.Y(),
			colorBlue,
		)

		normalLength := 30.0
		for _, data := range g.collisionData {
			ebitenutil.DrawLine(
				screen,
				data.Contact.X(),
				data.Contact.Y(),
				data.Contact.X()+data.Normal.X()*normalLength,
				data.Contact.Y()+data.Normal.Y()*normalLength,
				colorPurple,
			)
		}
	}

	// If we're in rect mode, visualise the user-controlled moving rect
	if g.mode == GameModeRect {
		ebitenutil.DrawRect(
			screen,
			g.movingRect.X(), g.movingRect.Y(),
			g.movingRect.Width(), g.movingRect.Height(),
			colorGreen,
		)

		ebitenutil.DrawLine(
			screen,
			g.movingRect.X()+g.movingRect.Width()/2,
			g.movingRect.Y()+g.movingRect.Height()/2,
			g.cursorVector.X(), g.cursorVector.Y(),
			colorPurple,
		)
	}

	// Draw debug text
	{
		ebitenutil.DebugPrint(
			screen,
			fmt.Sprintf("mode: %s, colliding: %t", g.mode, len(g.collisionData) != 0),
		)
	}
}

func (g *Game) Layout(_, _ int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	game := NewGame()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Ebitoolbox Collision Demo")

	if err := ebiten.RunGame(game); err != nil && err != ErrGameExit {
		log.Fatal(err)
	}
}
