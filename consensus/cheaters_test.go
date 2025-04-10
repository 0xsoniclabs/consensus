package consensus

import (
	"math/rand/v2"
	"testing"
)

func TestCheaters_ConversionToSet(t *testing.T) {
	distinctCheatersNum := 100
	cheaters := make(Cheaters, 0, distinctCheatersNum)
	for cheaterID := range distinctCheatersNum {
		cheaters = append(cheaters, ValidatorID(cheaterID))
		if cheaterID%5 == 0 {
			cheaters = append(cheaters, ValidatorID(cheaterID))
		}
	}
	rand.Shuffle(len(cheaters), func(i, j int) { cheaters[i], cheaters[j] = cheaters[j], cheaters[i] })
	cheaterSet := cheaters.Set()
	if want, got := len(cheaterSet), distinctCheatersNum; want != got {
		t.Fatalf("unexpected numbers of cheaters in set, want: %d, got: %d", want, got)
	}
	for cheaterID := range distinctCheatersNum {
		if _, ok := cheaterSet[ValidatorID(cheaterID)]; !ok {
			t.Fatalf("expected cheater: %d not found in final set", cheaterID)
		}
	}
}
