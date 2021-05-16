package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"math"

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
	GameModeBox   GameMode = "box"
)

type Rect struct {
	x, y, w, h float64
}

func (r Rect) X() float64      { return r.x }
func (r Rect) Y() float64      { return r.y }
func (r Rect) Width() float64  { return r.w }
func (r Rect) Height() float64 { return r.h }

type Game struct {
	// Indicates which collision mode we're in
	mode GameMode

	// Used in all modes
	target       Rect
	cursorVector vector.Vec2
	isColliding  bool

	// Used in ray mode
	rayOrigin        vector.Vec2
	collisionDetails RayRectCollision

	// Used in box mode
	movingRect Rect
}

func NewGame() *Game {
	rs := 300.0

	return &Game{
		mode:       GameModePoint,
		target:     Rect{(screenWidth / 2) - (rs / 2), (screenHeight / 2) - (rs / 2), rs, rs},
		movingRect: Rect{100, 100, 100, 100},
	}
}

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
			g.mode = GameModeBox
		case GameModeBox:
			g.mode = GameModePoint
		}
	}

	// Re-set the game's cursor vector
	{
		cx, cy := ebiten.CursorPosition()
		g.cursorVector = vector.NewVec2(float64(cx), float64(cy))
	}

	// Do whatever setup is required for the current mode
	switch g.mode {
	case GameModePoint:
		g.isColliding = PointInAABB(g.cursorVector, g.target)
	case GameModeRay:
		g.rayOrigin = vector.NewVec2(g.target.X()-200, g.target.Y()-200)
	case GameModeRay2:
		g.rayOrigin = vector.NewVec2(
			g.target.X()+g.target.Width()+200,
			g.target.Y()+g.target.Height()+200,
		)
	}

	// Perform the current mode's collision test
	switch g.mode {
	case GameModePoint:
		g.isColliding = PointInAABB(g.cursorVector, g.target)
	case GameModeRay, GameModeRay2:
		direction := vector.Vec2(g.cursorVector.Minus(g.rayOrigin))
		g.isColliding, g.collisionDetails = RayVsRect(g.rayOrigin, direction, g.target)
	case GameModeBox:
		// Determine how the box should move this tick
		mv := vector.NewVec2(0, 0)
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			lerpSpeed := 0.05
			mv.SetX((g.cursorVector.X() - (g.movingRect.Width() / 2) - g.movingRect.X()) * lerpSpeed)
			mv.SetY((g.cursorVector.Y() - (g.movingRect.Height() / 2) - g.movingRect.Y()) * lerpSpeed)
		}

		g.isColliding, g.collisionDetails = MovingRectVsRect(g.movingRect, mv, g.target)

		if g.isColliding {
			log.Println(g.collisionDetails)

		}

		overlap := 1 - g.collisionDetails.Time

		// mv.SetX(mv.X() * g.collisionDetails.Normal.X() )

		cmv := mv.Add(
			vector.NewVec2(
				math.Abs(mv.X())*g.collisionDetails.Normal.X()*overlap,
				math.Abs(mv.Y())*g.collisionDetails.Normal.Y()*overlap,
			),
		)

		g.movingRect.x += cmv.X()
		g.movingRect.y += cmv.Y()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// If we're in box mode, visualise the expanded rectangle we use for collision
	if g.mode == GameModeBox {
		ebitenutil.DrawRect(
			screen,
			g.target.X()-(g.movingRect.Width()/2),
			g.target.Y()-(g.movingRect.Height()/2),
			g.target.Width()+g.movingRect.Width(),
			g.target.Height()+g.movingRect.Height(),
			colorPurple,
		)
	}

	// Draw the target rect - in all modes
	{
		var chosenColor color.RGBA
		if g.isColliding {
			chosenColor = colorRed
		} else {
			chosenColor = colorGreen
		}

		ebitenutil.DrawRect(
			screen,
			g.target.X(),
			g.target.Y(),
			g.target.Width(),
			g.target.Height(),
			chosenColor,
		)
	}

	// If we're in ray mode, draw the ray and collision details
	if g.mode == GameModeRay || g.mode == GameModeRay2 {
		ebitenutil.DrawLine(
			screen,
			g.rayOrigin.X(), g.rayOrigin.Y(), g.cursorVector.X(), g.cursorVector.Y(),
			colorBlue,
		)

		if g.isColliding {
			ebitenutil.DrawLine(
				screen,
				g.collisionDetails.Contact.X(),
				g.collisionDetails.Contact.Y(),
				g.collisionDetails.Contact.X()+g.collisionDetails.Normal.X()*50,
				g.collisionDetails.Contact.Y()+g.collisionDetails.Normal.Y()*50,
				colorYellow,
			)

			ebitenutil.DebugPrintAt(
				screen,
				fmt.Sprintf("%f", g.collisionDetails.Time),
				int(g.collisionDetails.Contact.X()),
				int(g.collisionDetails.Contact.Y()),
			)
		}
	}

	// If we're in box mode, visualise the user-controlled moving rect
	if g.mode == GameModeBox {
		ebitenutil.DrawRect(
			screen,
			g.movingRect.x, g.movingRect.y,
			g.movingRect.w, g.movingRect.h,
			colorGreen,
		)
	}

	// Draw debug text
	{
		ebitenutil.DebugPrint(
			screen,
			fmt.Sprintf("mode: %s, colliding: %t", g.mode, g.isColliding),
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
