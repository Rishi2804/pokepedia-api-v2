package service

import "testing"

func TestBattleOnly(t *testing.T) {
	base, item, itemName := int32(6), "charizarditex", "Charizardite X"

	if got := battleOnly(nil, false, nil, nil); got != nil {
		t.Errorf("an ordinary pokemon has no battle-only info, got %+v", got)
	}

	mega := battleOnly(&base, false, &item, &itemName)
	if mega == nil || mega.BaseID == nil || *mega.BaseID != 6 || mega.RequiredItem == nil ||
		mega.RequiredItem.ShowdownID != "charizarditex" || mega.RequiredItem.Name != "Charizardite X" || mega.Gigantamax {
		t.Errorf("mega = %+v", mega)
	}

	gmax := battleOnly(&base, true, nil, nil)
	if gmax == nil || !gmax.Gigantamax || gmax.RequiredItem != nil {
		t.Errorf("gmax = %+v, want a base, no item and the Gmax flag", gmax)
	}

	// Ogerpon's masks need an item but have no base species.
	mask, maskName := "wellspringmask", "Wellspring Mask"
	ogerpon := battleOnly(nil, false, &mask, &maskName)
	if ogerpon == nil || ogerpon.BaseID != nil || ogerpon.RequiredItem == nil {
		t.Errorf("ogerpon = %+v, want an item and no base", ogerpon)
	}
}
