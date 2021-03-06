package models

import (
	"math/rand"
	"strings"
)

// batchNameLists: these lists are each randomly shuffled and then one item is
// pulled from each list to generate a batch name.  In a perfect world, all
// lists would be the same size, just to make for absolute uniqueness per
// iteration, but that isn't worth worrying over.
var batchNameLists = [][]string{
	// Materials (as if this were a work of art chiseled / carved from these)
	{
		// Earthy stuff: stone / metal / gemstone
		"Basalt",
		"Bronze",
		"Clay",
		"Diamond",
		"Emerald",
		"Granite",
		"Jade",
		"Marble",
		"Obsidian",
		"Quartzite",
		"Serpentine",

		// Wood
		"Cedar",
		"Mahogany",
		"Maple",
		"Oak",
		"Pine",
		"Teak",

		// Slightly more unique / random / weird
		"Holographic",
		"Polyester",
		"Origami",
	},

	// Creatures
	{
		"Dracaenae",
		"Gargouille",
		"Gendenwitha",
		"HuayChivo",
		"Kubikajiri",
		"Laestrygonian",
		"Manananggal",
		"Namahage",
		"OrcoMamman",
		"Penanggalan",
		"Qingniao",
		"Shinigami",
		"Soucouyant",
		"Unicorn",
		"Xiangliu",
		"Zombie",
		"Vampire",
		"Werewolf",
		"Cthulu",
		"Yeti",
	},

	// Connecting words
	{
		"Behind",
		"Harvesting",
		"Planting",
		"SurroundedBy",
		"Trampling",
		"Eating",
	},

	// Vegetation
	{
		"Crabgrass",
		"PoisonOak",
		"Roses",
		"Shrubs",
		"Strawberries",
		"Sunflowers",
		"Trees",
		"Wheat",
		"Bamboo",
		"Amaranth",
		"Kelp",
		"PricklyPear",
	},
}

// copyLists does what you think: takes the main batch name lists and copies
// them to allow for list manipulation without losing the original data
func copyLists() [][]string {
	var localLists = make([][]string, len(batchNameLists))
	for i, mainList := range batchNameLists {
		localLists[i] = make([]string, len(mainList))
		copy(localLists[i], mainList)
	}

	return localLists
}

// RandomBatchName generates a unique name for the given sequence.  The names
// are guaranteed not to duplicate any of the component pieces until one
// component's values have all been used.  The full names are guaranteed not to
// duplicate until the longest list has been used up, at which point all lists
// will be reshuffled.
func RandomBatchName(seq uint32) string {
	var lists = copyLists()

	// Find the longest list so we know when it's time to reshuffle - this is our
	// shortcut for the sequence number.  By dividing seq by iterations, we only
	// run through a single list instead of potentially reshuffling hundreds of
	// times to get to the right sequence.
	var iterations uint32
	for _, list := range lists {
		var l = uint32(len(list))
		if l > iterations {
			iterations = l
		}
	}

	var seedPlus = seq / iterations
	seq %= iterations

	// Set a constant randomization seed
	rand.Seed(int64(0xF00D1E5 + seedPlus))

	// Shuffle the lists
	for _, list := range lists {
		rand.Shuffle(len(list), func(i, j int) {
			list[i], list[j] = list[j], list[i]
		})
	}

	// Now grab the items at seq
	var nameParts = make([]string, len(lists))
	for i, list := range lists {
		var l = uint32(len(list))
		nameParts[i] = list[seq%l]
	}

	return strings.Join(nameParts, "")
}
