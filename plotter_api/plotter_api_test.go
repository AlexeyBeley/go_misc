package plotterapi

import (
	"testing"

	"gonum.org/v1/plot/plotter"
)

func TestPrintPoints(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		pts := plotter.XYs{
			{X: 1, Y: 1},
			{X: 2, Y: 2},
			{X: 3, Y: 4},
			{X: 4, Y: 8},
			{X: 5, Y: 16},
		}
		err := PrintPoints(pts)
		if err != nil {
			panic(err)
		}

	})
}
