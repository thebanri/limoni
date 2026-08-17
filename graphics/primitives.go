package graphics

import (
	"math"
)

// NewCube creates a 3D cube model with the specified edge length, centered at the origin.
// It includes 8 vertices, 6 quad faces, and standard UV texture mapping coordinates.
func NewCube(size float64) Model3D {
	half := size / 2.0
	if size <= 0 {
		half = 1.0
	}
	return Model3D{
		Name: "Cube",
		Vertices: []Vertex3D{
			{X: -half, Y: -half, Z: -half}, // 0
			{X: half, Y: -half, Z: -half},  // 1
			{X: half, Y: half, Z: -half},   // 2
			{X: -half, Y: half, Z: -half},  // 3
			{X: -half, Y: -half, Z: half},  // 4
			{X: half, Y: -half, Z: half},   // 5
			{X: half, Y: half, Z: half},    // 6
			{X: -half, Y: half, Z: half},   // 7
		},
		Faces: [][]int{
			{0, 1, 2, 3}, // Front
			{5, 4, 7, 6}, // Back
			{1, 5, 6, 2}, // Right
			{4, 0, 3, 7}, // Left
			{3, 2, 6, 7}, // Top
			{4, 5, 1, 0}, // Bottom
		},
		UVs: []UV{
			{U: 0.0, V: 1.0}, // 0: Bottom-Left
			{U: 1.0, V: 1.0}, // 1: Bottom-Right
			{U: 1.0, V: 0.0}, // 2: Top-Right
			{U: 0.0, V: 0.0}, // 3: Top-Left
		},
		FaceUVs: [][]int{
			{0, 1, 2, 3},
			{0, 1, 2, 3},
			{0, 1, 2, 3},
			{0, 1, 2, 3},
			{0, 1, 2, 3},
			{0, 1, 2, 3},
		},
	}
}

// NewPyramid creates a 3D pyramid model with a square base and apex.
// It includes 5 vertices, 1 quad base face, and 4 triangular side faces.
func NewPyramid(baseSize, height float64) Model3D {
	half := baseSize / 2.0
	if baseSize <= 0 {
		half = 1.0
	}
	if height <= 0 {
		height = 2.0
	}
	halfH := height / 2.0
	return Model3D{
		Name: "Pyramid",
		Vertices: []Vertex3D{
			{X: -half, Y: -halfH, Z: -half}, // 0: Base Back-Left
			{X: half, Y: -halfH, Z: -half},  // 1: Base Back-Right
			{X: half, Y: -halfH, Z: half},   // 2: Base Front-Right
			{X: -half, Y: -halfH, Z: half},  // 3: Base Front-Left
			{X: 0.0, Y: halfH, Z: 0.0},      // 4: Apex Top
		},
		Faces: [][]int{
			{3, 2, 1, 0}, // Base (Quad, normal facing -Y)
			{0, 1, 4},    // Back face (Tri, edge 0->1 to apex 4)
			{1, 2, 4},    // Right face (Tri, edge 1->2 to apex 4)
			{2, 3, 4},    // Front face (Tri, edge 2->3 to apex 4)
			{3, 0, 4},    // Left face (Tri, edge 3->0 to apex 4)
		},
		UVs: []UV{
			{U: 0.0, V: 1.0}, // 0: Bottom-Left
			{U: 1.0, V: 1.0}, // 1: Bottom-Right
			{U: 1.0, V: 0.0}, // 2: Top-Right
			{U: 0.0, V: 0.0}, // 3: Top-Left
			{U: 0.5, V: 0.0}, // 4: Apex Top-Center
		},
		FaceUVs: [][]int{
			{0, 1, 2, 3},
			{0, 1, 4},
			{0, 1, 4},
			{0, 1, 4},
			{0, 1, 4},
		},
	}
}

// NewSphere creates a parametric 3D UV sphere model with the specified radius and segment resolution.
func NewSphere(radius float64, latitudes, longitudes int) Model3D {
	if radius <= 0 {
		radius = 1.0
	}
	if latitudes < 4 {
		latitudes = 14
	}
	if longitudes < 4 {
		longitudes = 14
	}

	var vertices []Vertex3D
	var faces [][]int
	var uvs []UV
	var faceUVs [][]int

	for lat := 0; lat <= latitudes; lat++ {
		theta := float64(lat) * math.Pi / float64(latitudes)
		sinTheta, cosTheta := math.Sin(theta), math.Cos(theta)
		v := float64(lat) / float64(latitudes)

		for lon := 0; lon <= longitudes; lon++ {
			phi := float64(lon) * 2 * math.Pi / float64(longitudes)
			sinPhi, cosPhi := math.Sin(phi), math.Cos(phi)
			u := float64(lon) / float64(longitudes)

			x := radius * sinTheta * cosPhi
			y := radius * cosTheta
			z := radius * sinTheta * sinPhi

			vertices = append(vertices, Vertex3D{X: x, Y: y, Z: z})
			uvs = append(uvs, UV{U: u, V: v})
		}
	}

	stride := longitudes + 1
	for lat := 0; lat < latitudes; lat++ {
		for lon := 0; lon < longitudes; lon++ {
			p0 := lat*stride + lon
			p1 := lat*stride + (lon + 1)
			p2 := (lat+1)*stride + (lon + 1)
			p3 := (lat+1)*stride + lon

			faces = append(faces, []int{p3, p2, p1, p0})
			faceUVs = append(faceUVs, []int{p3, p2, p1, p0})
		}
	}

	return Model3D{
		Name:     "Sphere",
		Vertices: vertices,
		Faces:    faces,
		UVs:      uvs,
		FaceUVs:  faceUVs,
	}
}

// NewTorus creates a parametric 3D torus (donut) model.
// r1 is the major radius (distance from center of tube to center of torus).
// r2 is the minor radius (radius of the tube itself).
func NewTorus(r1, r2 float64, radialSegments, tubularSegments int) Model3D {
	if r1 <= 0 {
		r1 = 0.8
	}
	if r2 <= 0 {
		r2 = 0.35
	}
	if radialSegments < 4 {
		radialSegments = 14
	}
	if tubularSegments < 4 {
		tubularSegments = 14
	}

	var vertices []Vertex3D
	var faces [][]int
	var uvs []UV
	var faceUVs [][]int

	for r := 0; r < radialSegments; r++ {
		u := float64(r) / float64(radialSegments)
		theta := u * 2 * math.Pi
		sinTheta, cosTheta := math.Sin(theta), math.Cos(theta)

		for t := 0; t < tubularSegments; t++ {
			v := float64(t) / float64(tubularSegments)
			phi := v * 2 * math.Pi
			sinPhi, cosPhi := math.Sin(phi), math.Cos(phi)

			x := (r1 + r2*cosPhi) * cosTheta
			y := r2 * sinPhi
			z := (r1 + r2*cosPhi) * sinTheta

			vertices = append(vertices, Vertex3D{X: x, Y: y, Z: z})
			uvs = append(uvs, UV{U: u, V: v})
		}
	}

	for r := 0; r < radialSegments; r++ {
		for t := 0; t < tubularSegments; t++ {
			p0 := r*tubularSegments + t
			p1 := r*tubularSegments + (t + 1)%tubularSegments
			p2 := ((r+1)%radialSegments)*tubularSegments + (t + 1)%tubularSegments
			p3 := ((r+1)%radialSegments)*tubularSegments + t

			faces = append(faces, []int{p3, p2, p1, p0})
			faceUVs = append(faceUVs, []int{p3, p2, p1, p0})
		}
	}

	return Model3D{
		Name:     "Torus",
		Vertices: vertices,
		Faces:    faces,
		UVs:      uvs,
	}
}
