package farkle

import "testing"

func prepareCounts(values []int) map[int]int {
	counts := make(map[int]int)
	for val, count := range values {
		counts[val+1] = count
	}
	return counts
}

func testVariant(t *testing.T, values []int, expectedScore int, expectedExtra bool) {
	score, extra := scoreCounts(prepareCounts(values))
	if score != expectedScore {
		t.Errorf("Expected score %d, got %d", expectedScore, score)
	}
	if extra != expectedExtra {
		t.Errorf("Expected extra %t, got %t", expectedExtra, extra)
	}
}

func TestScoreCountsFullStraight(t *testing.T) {
	testVariant(t, []int{1, 1, 1, 1, 1, 1}, ScoreFullStraight, false)
}

func TestScoreCountsUpperStraight(t *testing.T) {
	testVariant(t, []int{0, 1, 1, 1, 1, 1}, ScoreUpperStraight, false)
	testVariant(t, []int{0, 2, 1, 1, 1, 1}, ScoreUpperStraight, true)
	testVariant(t, []int{0, 1, 2, 1, 1, 1}, ScoreUpperStraight, true)
	testVariant(t, []int{0, 1, 1, 2, 1, 1}, ScoreUpperStraight, true)
	testVariant(t, []int{0, 1, 1, 1, 2, 1}, ScoreUpperStraight+50, false)
	testVariant(t, []int{0, 1, 1, 1, 1, 2}, ScoreUpperStraight, true)
}

func TestScoreCountsLowerStraight(t *testing.T) {
	testVariant(t, []int{1, 1, 1, 1, 1, 0}, ScoreLowerStraight, false)
	testVariant(t, []int{2, 1, 1, 1, 1, 0}, ScoreLowerStraight+100, false)
	testVariant(t, []int{1, 2, 1, 1, 1, 0}, ScoreLowerStraight, true)
	testVariant(t, []int{1, 1, 2, 1, 1, 0}, ScoreLowerStraight, true)
	testVariant(t, []int{1, 1, 1, 2, 1, 0}, ScoreLowerStraight, true)
	testVariant(t, []int{1, 1, 1, 1, 2, 0}, ScoreLowerStraight+50, false)
}

func TestScoreCountsOnes(t *testing.T) {
	testVariant(t, []int{2, 0, 1, 0, 0, 0}, 200, true)
}

func TestScoreCountsFives(t *testing.T) {
	testVariant(t, []int{0, 0, 1, 0, 2, 0}, 100, true)
}

func TestScoreCountsThreeOfKind(t *testing.T) {
	testVariant(t, []int{3, 0, 0, 0, 0, 0}, 1000, false)
	testVariant(t, []int{0, 3, 0, 0, 0, 0}, 200, false)
	testVariant(t, []int{0, 0, 3, 0, 0, 0}, 300, false)
	testVariant(t, []int{0, 0, 0, 3, 0, 0}, 400, false)
	testVariant(t, []int{0, 0, 0, 0, 3, 0}, 500, false)
	testVariant(t, []int{0, 0, 0, 0, 0, 3}, 600, false)
	testVariant(t, []int{0, 0, 1, 0, 0, 3}, 600, true)
}

func TestScoreCountsFourOfKind(t *testing.T) {
	testVariant(t, []int{4, 0, 0, 0, 0, 0}, 2000, false)
	testVariant(t, []int{0, 4, 0, 0, 0, 0}, 400, false)
	testVariant(t, []int{0, 0, 4, 0, 0, 0}, 600, false)
	testVariant(t, []int{0, 0, 0, 4, 0, 0}, 800, false)
	testVariant(t, []int{0, 0, 0, 0, 4, 0}, 1000, false)
	testVariant(t, []int{0, 0, 0, 0, 0, 4}, 1200, false)
	testVariant(t, []int{0, 0, 1, 0, 0, 4}, 1200, true)
}

func TestScoreCountsFiveOfKind(t *testing.T) {
	testVariant(t, []int{5, 0, 0, 0, 0, 0}, 4000, false)
	testVariant(t, []int{0, 5, 0, 0, 0, 0}, 800, false)
	testVariant(t, []int{0, 0, 5, 0, 0, 0}, 1200, false)
	testVariant(t, []int{0, 0, 0, 5, 0, 0}, 1600, false)
	testVariant(t, []int{0, 0, 0, 0, 5, 0}, 2000, false)
	testVariant(t, []int{0, 0, 0, 0, 0, 5}, 2400, false)
	testVariant(t, []int{0, 0, 1, 0, 0, 5}, 2400, true)
}

func TestScoreCountsSixOfKind(t *testing.T) {
	testVariant(t, []int{6, 0, 0, 0, 0, 0}, 8000, false)
	testVariant(t, []int{0, 6, 0, 0, 0, 0}, 1600, false)
	testVariant(t, []int{0, 0, 6, 0, 0, 0}, 2400, false)
	testVariant(t, []int{0, 0, 0, 6, 0, 0}, 3200, false)
	testVariant(t, []int{0, 0, 0, 0, 6, 0}, 4000, false)
	testVariant(t, []int{0, 0, 0, 0, 0, 6}, 4800, false)
}

func TestDiceCountMultiplier(t *testing.T) {
	const expected = 8000
	actual := diceCountMultiplier(1000, 6)
	if actual != expected {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}
