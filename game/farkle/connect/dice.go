package connect

import "github.com/google/uuid"

const (
	DiceRegular = "regular" // Same chance for all values
	DiceEven    = "even"    // Higher chance for even values
	DiceOdd     = "odd"     // Higher chance for odd values
	DiceLucky   = "lucky"   // Higher chance for 1, 5 and 6
)

var DiceDistributions = map[string][6]int{
	DiceRegular: {1, 1, 1, 1, 1, 1},
	DiceEven:    {1, 3, 1, 3, 1, 3},
	DiceOdd:     {3, 1, 3, 1, 3, 1},
	DiceLucky:   {5, 1, 1, 1, 4, 2},
}

type LobbyDice struct {
	ID      uuid.UUID `json:"id"`
	Variant string    `json:"variant"`
}
