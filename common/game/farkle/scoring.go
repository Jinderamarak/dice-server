package farkle

import (
	"dice-server/common/game/farkle/data"
	"math"
)

const (
	ScoreFullStraight  = 1500
	ScoreUpperStraight = 750
	ScoreLowerStraight = 500
)

func countValues(dice []data.Dice) map[int]int {
	counts := make(map[int]int)
	for _, d := range dice {
		counts[d.Value]++
	}
	return counts
}

func hasSingle(counts map[int]int) bool {
	return counts[1] > 0 || counts[5] > 0
}

func hasThreeOfKind(counts map[int]int) bool {
	for _, count := range counts {
		if count >= 3 {
			return true
		}
	}
	return false
}

func hasLowerStraight(counts map[int]int) bool {
	return counts[1] > 0 && counts[2] > 0 && counts[3] > 0 && counts[4] > 0 && counts[5] > 0
}

func hasUpperStraight(counts map[int]int) bool {
	return counts[2] > 0 && counts[3] > 0 && counts[4] > 0 && counts[5] > 0 && counts[6] > 0
}

func hasFullStraight(counts map[int]int) bool {
	return counts[1] == 1 && counts[2] == 1 && counts[3] == 1 && counts[4] == 1 && counts[5] == 1 && counts[6] == 1
}

func hasBusted(counts map[int]int) bool {
	return !hasSingle(counts) &&
		!hasThreeOfKind(counts) &&
		!hasLowerStraight(counts) &&
		!hasUpperStraight(counts)
}

func scoreCounts(counts map[int]int) (int, bool) {
	total := 0
	for _, count := range counts {
		total += count
	}

	if hasFullStraight(counts) {
		return ScoreFullStraight, false
	}
	if hasUpperStraight(counts) {
		if counts[5] == 2 {
			return ScoreUpperStraight + 50, false
		}
		return ScoreUpperStraight, total > 5
	}
	if hasLowerStraight(counts) {
		if counts[1] == 2 {
			return ScoreLowerStraight + 100, false
		}
		if counts[5] == 2 {
			return ScoreLowerStraight + 50, false
		}
		return ScoreLowerStraight, total > 5
	}

	score := 0
	extra := false
	for value, count := range counts {
		switch value {
		case 1:
			if count >= 3 {
				score += diceCountMultiplier(1000, count)
			} else {
				score += count * 100
			}
		case 5:
			if count >= 3 {
				score += diceCountMultiplier(500, count)
			} else {
				score += count * 50
			}
		case 2, 3, 4, 6:
			if count >= 3 {
				score += diceCountMultiplier(value*100, count)
			} else if count > 0 {
				extra = true
			}
		}
	}

	return score, extra
}

func diceCountMultiplier(base, count int) int {
	return base * powInt(2, count-3)
}

func powInt(x, y int) int {
	return int(math.Pow(float64(x), float64(y)))
}
