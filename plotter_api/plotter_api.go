package plotterapi

import (
	"fmt"
	"log"
	"image/color"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func PrintPoints(pts plotter.XYs) error {

	// 2. Create a new plot
	p := plot.New()

	// 3. Set the plot's title and axis labels
	p.Title.Text = "Simple Line Plot"
	p.X.Label.Text = "X-axis"
	p.Y.Label.Text = "Y-axis"

	// 4. Create a line and a scatter plot from the data points
	line, err := plotter.NewLine(pts)
	line.Color= color.RGBA{R: 255, A: 255}
	
	if err != nil {
		log.Fatal(err)
	}

	scatter, err := plotter.NewScatter(pts)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Add the line and scatter plot to the plot
	p.Add(line, scatter)

	// 6. Save the plot to a file
	filename := "line-plot.png"
	if err := p.Save(4*vg.Inch, 4*vg.Inch, filename); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Graph saved to %s\n", filename)

	// You can also add more elements, like a legend
	p.Legend.Add("Data Points", line, scatter)
	if err := p.Save(4*vg.Inch, 4*vg.Inch, "line-plot-with-legend.png"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Graph with legend saved to line-plot-with-legend.png")
	return nil
}
