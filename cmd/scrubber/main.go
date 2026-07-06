package main

import (
	"flag"
	"log"
)

func main() {
	source := flag.String("source", "", "path to local 5etools repository root")
	edition := flag.String("edition", "", "target edition: \"5e\" or \"5.5e\" (monsters mode only)")
	mode := flag.String("mode", "monsters", "what to scrub: \"monsters\" or \"items\"")
	flag.Parse()

	if *source == "" {
		log.Fatal("--source is required")
	}

	switch *mode {
	case "items":
		// No --edition: items derive their edition per entry from their
		// 5etools source code, not from which checkout is scrubbed — see
		// design.md Decision 7. Point --source at SCRUBBER_SOURCE_5_5E, the
		// only checkout items are scrubbed from (it's a superset of the 5e
		// checkout's item data).
		runItemScrub(*source)
	case "monsters":
		runMonsterScrub(*source, *edition)
	default:
		log.Fatalf("--mode must be \"monsters\" or \"items\", got %q", *mode)
	}
}
