package vector

import "math"

type Vec2 struct{ x, y float64 }

func NewVec2(x, y float64) Vec2 { return Vec2{x, y} }

func (v Vec2) X() float64 { return v.x }
func (v Vec2) Y() float64 { return v.y }

func (v *Vec2) SetX(x float64) { v.x = x }
func (v *Vec2) SetY(y float64) { v.y = y }

func (v Vec2) Add(other Vec2) Vec2   { return NewVec2(v.X()+other.X(), v.Y()+other.Y()) }
func (v Vec2) Minus(other Vec2) Vec2 { return NewVec2(v.X()-other.X(), v.Y()-other.Y()) }

// Do we actually need this?
// https://golang.org/ref/spec#Comparison_operators
func (v Vec2) Equals(other Vec2) bool { return v.X() == other.X() && v.Y() == other.Y() }

func (v Vec2) SquaredDistance(to Vec2) float64 {
	delta := v.Minus(to)
	return (delta.X() * delta.X()) + (delta.Y() * delta.Y())
}

func (v Vec2) EuclidianDistance(to Vec2) float64 { return math.Sqrt(v.SquaredDistance(to)) }

func (v Vec2) ManhattanDistance(to Vec2) float64 {
	delta := v.Minus(to)
	return math.Abs(delta.X()) + math.Abs(delta.Y())
}

func (v Vec2) Cantor() float64 {
	return ((v.X() + v.Y()) * (v.X() + v.Y() + 1) / 2) + v.Y()
}
