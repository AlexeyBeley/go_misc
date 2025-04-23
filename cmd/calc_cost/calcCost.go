package main

import "log"

func main() {
	mliters := 0.71 * 6
	cost := 3.77

	costperLiter := float64((cost / mliters))
	log.Printf("Cost per Liters: %f", costperLiter)
}
