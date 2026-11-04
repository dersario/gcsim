package characters

import (
	"fmt"
	"testing"

	_ "github.com/genshinsim/gcsim/internal/characters/nefer"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/keys"
	"github.com/genshinsim/gcsim/pkg/testhelper"
)

func TestNeferPhantasmCancel(t *testing.T) {
	for _, cons := range []int{0, 6} {
		for _, route := range []struct {
			name       string
			hold       int
			delayedDew bool
		}{
			{name: "tap"},
			{name: "hold-with-dew", hold: 150},
			{name: "slither-to-phantasm", hold: 150, delayedDew: true},
		} {
			for _, cancel := range []action.Action{action.InvalidAction, action.ActionDash, action.ActionBurst} {
				t.Run(fmt.Sprintf("C%d/%s/cancel=%s", cons, route.name, cancel), func(t *testing.T) {
					c, targets := makeCore(1)
					c.Combat.DefaultTarget = targets[0].Key()
					profile := testhelper.DefaultProfile(keys.Nefer, testhelper.TestWeaponKey)
					profile.Base.Cons = cons
					idx, err := c.AddChar(profile)
					if err != nil {
						t.Fatal(err)
					}
					c.Player.SetActive(idx)
					if err := c.Init(); err != nil {
						t.Fatal(err)
					}
					if !route.delayedDew {
						c.Player.SetVerdantDew(3)
					}
					c.Player.ActiveChar().AddEnergy("test", 60)

					hits := make(map[string]int)
					c.Events.Subscribe(event.OnEnemyDamage, func(args ...any) {
						attack := args[1].(*info.AttackEvent)
						hits[attack.Info.Abil]++
					}, "nefer-phantasm-hits")
					execute := func(next action.Action, params map[string]int) {
						t.Helper()
						for range 300 {
							if c.Player.ReadyCheck(next, keys.Nefer, params) == nil {
								if err := c.Player.Exec(next, keys.Nefer, params); err != nil {
									t.Fatal(err)
								}
								return
							}
							advanceCoreFrame(c)
						}
						t.Fatalf("%s did not become ready", next)
					}
					execute(action.ActionSkill, nil)
					execute(action.ActionCharge, map[string]int{"hold": route.hold})
					if route.delayedDew {
						// Supply Dew after Slither starts, so this case cannot take
						// the direct Phantasm route used when Dew is already available.
						for range 30 {
							advanceCoreFrame(c)
						}
						if hits["Phantasm Performance (Nefer 1)"] != 0 {
							t.Fatal("Phantasm started before Dew became available")
						}
						c.Player.SetVerdantDew(3)
					}
					if cancel != action.InvalidAction {
						execute(cancel, nil)
					}
					for range 200 {
						advanceCoreFrame(c)
					}

					wantSecond := 1
					if cancel != action.InvalidAction {
						wantSecond = 0
					}
					wantEnd := 0
					if cons == 6 {
						wantEnd = 1
					}
					for ability, want := range map[string]int{
						"Phantasm Performance (Nefer 1)":    1,
						"Phantasm Performance (Nefer 2)":    wantSecond,
						"Phantasm Performance (Shade 1)":    1,
						"Phantasm Performance (Shade 2)":    1,
						"Phantasm Performance (Shade 3)":    1,
						"Phantasm Performance (C6 End Hit)": wantEnd,
					} {
						if hits[ability] != want {
							t.Errorf("%s: got %d hits, want %d", ability, hits[ability], want)
						}
					}
				})
			}
		}
	}
}

func TestNeferConsecutivePhantasmEndHits(t *testing.T) {
	c, targets := makeCore(1)
	c.Combat.DefaultTarget = targets[0].Key()
	profile := testhelper.DefaultProfile(keys.Nefer, testhelper.TestWeaponKey)
	profile.Base.Cons = 6
	idx, err := c.AddChar(profile)
	if err != nil {
		t.Fatal(err)
	}
	c.Player.SetActive(idx)
	if err := c.Init(); err != nil {
		t.Fatal(err)
	}
	c.Player.SetVerdantDew(3)

	hits := make(map[string]int)
	c.Events.Subscribe(event.OnEnemyDamage, func(args ...any) {
		attack := args[1].(*info.AttackEvent)
		hits[attack.Info.Abil]++
	}, "consecutive-phantasm-hits")
	execute := func(next action.Action) {
		t.Helper()
		for range 300 {
			if c.Player.ReadyCheck(next, keys.Nefer, nil) == nil {
				if err := c.Player.Exec(next, keys.Nefer, nil); err != nil {
					t.Fatal(err)
				}
				return
			}
			advanceCoreFrame(c)
		}
		t.Fatalf("%s did not become ready", next)
	}
	execute(action.ActionSkill)
	execute(action.ActionCharge)
	execute(action.ActionDash)
	execute(action.ActionCharge)
	for range 200 {
		advanceCoreFrame(c)
	}

	for ability, want := range map[string]int{
		"Phantasm Performance (Nefer 1)":    2,
		"Phantasm Performance (Nefer 2)":    1,
		"Phantasm Performance (Shade 1)":    2,
		"Phantasm Performance (Shade 2)":    2,
		"Phantasm Performance (Shade 3)":    2,
		"Phantasm Performance (C6 End Hit)": 2,
	} {
		if hits[ability] != want {
			t.Errorf("%s: got %d hits, want %d", ability, hits[ability], want)
		}
	}
}
