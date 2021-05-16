package main

import (
	"log"
	"math"

	"github.com/tmhmitchell/ebitoolbox/datastructures/vector"
)

type AABB interface {
	X() float64
	Y() float64
	Width() float64
	Height() float64
}

type CollisionData struct {
	// The point at which the ray contacted the rectangle
	Contact vector.Vec2

	// Normal is the normal vector of the surface in collision
	Normal vector.Vec2

	// Time is the distance down the ray at which contact is made
	Time float64
}

func PointInAABB(point vector.Vec2, box AABB) bool {
	return box.X() < point.X() && box.X()+box.Width() >= point.X() &&
		box.Y() < point.Y() && box.Y()+box.Height() >= point.Y()
}

// determineNearAndFar is a helper function for determining if a ray
// intersects with an AABB (see RayVsRect). It calculates the near and far "ray
// times" (ie, the distance to collision down a ray) in a single axis
func determineNearAndFar(origin, direction, targetOrigin, targetSize float64) (float64, float64) {
	// Protect against division by zero
	// If the direction vector for a given axis is 0, we know there will never
	// be an intersection - as such, the "ray time" is infinite
	// The -/+ signing is because of the direction to the appropriate intersection
	// points relative to the ray. Draw a square with an arrow crossing through
	// straight through it, you'll see what I mean!
	if direction == 0 {
		return math.Inf(-1), math.Inf(1)
	}

	// Determine some candidate values for near and far times
	cxn := (targetOrigin - origin) / direction
	cxf := ((targetOrigin + targetSize) - origin) / direction

	// If the candidate far time is less than the candidate near time, we'll
	// need to switch them around - or the far time isn't actually far away!
	if cxn <= cxf {
		return cxn, cxf
	}

	return cxf, cxn
}

// RayVsRect determines if a ray intersects an AABB. If it does, the first
// value returned will be true, and the second will be details of the collision.
// If not, false and an empty struct will be returned. If there was no collision,
// you should not consult the details, as they will not be meaningful.
func RayVsRect(origin, direction vector.Vec2, target AABB) (bool, CollisionData) {
	xn, xf := determineNearAndFar(origin.X(), direction.X(), target.X(), target.Width())
	yn, yf := determineNearAndFar(origin.Y(), direction.Y(), target.Y(), target.Height())

	if xn > yf || yn > xf {
		return false, CollisionData{}
	}

	n := math.Max(xn, yn)
	f := math.Min(xf, yf)

	// Some corner cases we need to consider:
	// If the nearest collision is beyond the end of the ray, we're not colliding
	// If the furthest collision is behind us, we're not colliding
	// Letting either of these pass would cause false positives
	if n > 1 || f < 0 {
		return false, CollisionData{}
	}

	// Determine the normal vector of the collision
	// BUG: There's an edge case here - if your ray intersects the AABB
	// diagonally, you'll get a (0, 0) normal vector because it's never re-assigned.
	// This probably has bad implications for corner collisions!
	var normal vector.Vec2
	{
		if xn > yn {
			if direction.X() < 0 {
				normal = vector.NewVec2(1, 0)
			} else {
				normal = vector.NewVec2(-1, 0)
			}
		} else if xn < yn {
			if direction.Y() < 0 {
				normal = vector.NewVec2(0, 1)
			} else {
				normal = vector.NewVec2(0, -1)
			}
		} else {
			log.Println("ebitoolbox: RayVsRect: returning a (0, 0 normal vector!")
		}

	}

	return true, CollisionData{
		Contact: origin.Add(vector.NewVec2(direction.X()*n, direction.Y()*n)),
		Normal:  normal,
		Time:    n,
	}
}

// MovingRectVsRect determines collision details for
func MovingRectVsRect(moving AABB, movement vector.Vec2, static AABB) (bool, CollisionData) {
	// We presume these two AABBs aren't __already__ colliding.
	// As a result, if there's no movement, there can be no collision.
	if movement.X() == 0 && movement.Y() == 0 {
		return false, CollisionData{}
	}

	expanded := Rect{
		static.X() - (moving.Width() / 2),
		static.Y() - (moving.Height() / 2),
		static.Width() + (moving.Width()),
		static.Height() + (moving.Height()),
	}

	colliding, details := RayVsRect(
		vector.NewVec2(moving.X()+(moving.Width()/2), moving.Y()+(moving.Height()/2)),
		movement,
		expanded,
	)

	if colliding && details.Time >= 0 && details.Time < 1.0 {
		return true, details
	}

	return false, CollisionData{}
}
