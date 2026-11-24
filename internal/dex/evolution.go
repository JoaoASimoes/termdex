package dex

import (
	"fmt"
	"strings"
)

// EvoNode is one species in an evolution chain.
type EvoNode struct {
	Species string         `json:"species"`
	Details []apiEvoDetail `json:"details"` // ways to evolve into this species from its parent
	Next    []EvoNode      `json:"next"`
}

// How describes the ways to evolve into this species, without duplicates.
func (n EvoNode) How() []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range n.Details {
		if s := describeEvolution(d); s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

type apiEvoDetail struct {
	Trigger               namedURL  `json:"trigger"`
	MinLevel              *int      `json:"min_level"`
	Item                  *namedURL `json:"item"`
	HeldItem              *namedURL `json:"held_item"`
	KnownMove             *namedURL `json:"known_move"`
	KnownMoveType         *namedURL `json:"known_move_type"`
	Location              *namedURL `json:"location"`
	MinHappiness          *int      `json:"min_happiness"`
	MinBeauty             *int      `json:"min_beauty"`
	MinAffection          *int      `json:"min_affection"`
	TimeOfDay             string    `json:"time_of_day"`
	NeedsOverworldRain    bool      `json:"needs_overworld_rain"`
	PartySpecies          *namedURL `json:"party_species"`
	PartyType             *namedURL `json:"party_type"`
	RelativePhysicalStats *int      `json:"relative_physical_stats"`
	TradeSpecies          *namedURL `json:"trade_species"`
	TurnUpsideDown        bool      `json:"turn_upside_down"`
	Gender                *int      `json:"gender"`
	NearSpecialRock       bool      `json:"near_special_rock"`
	NeedsMultiplayer      bool      `json:"needs_multiplayer"`
	UsedMove              *namedURL `json:"used_move"`
	MinMoveCount          *int      `json:"min_move_count"`
	MinSteps              *int      `json:"min_steps"`
	MinDamageTaken        *int      `json:"min_damage_taken"`
	Region                *namedURL `json:"region"`
}

type apiEvoChain struct {
	Species namedURL       `json:"species"`
	Details []apiEvoDetail `json:"evolution_details"`
	Next    []apiEvoChain  `json:"evolves_to"`
}

func buildEvoTree(c apiEvoChain) EvoNode {
	n := EvoNode{Species: c.Species.Name, Details: c.Details}
	for _, next := range c.Next {
		n.Next = append(n.Next, buildEvoTree(next))
	}
	return n
}

// describeEvolution turns one evolution_details entry into a short sentence,
// e.g. "Level 36", "Use Water Stone", "Trade holding Metal Coat".
func describeEvolution(d apiEvoDetail) string {
	var base string
	switch d.Trigger.Name {
	case "level-up":
		switch {
		case d.MinLevel != nil:
			base = fmt.Sprintf("Level %d", *d.MinLevel)
		case d.Location != nil || d.NearSpecialRock:
			// Location-based evolutions differ per game; keep it generic.
			return "Level up at a special location"
		default:
			base = "Level up"
		}
	case "use-item":
		if d.Item != nil {
			base = "Use " + DisplayName(d.Item.Name)
		} else {
			base = "Use an item"
		}
	case "trade":
		base = "Trade"
	case "use-move":
		base = "Level up"
	case "shed":
		return "Level 20 with an empty party slot and a Poké Ball (Shedinja)"
	default:
		base = DisplayName(d.Trigger.Name)
	}

	var cond []string
	if d.MinHappiness != nil {
		cond = append(cond, "with high friendship")
	}
	if d.MinAffection != nil {
		cond = append(cond, "with high affection")
	}
	if d.MinBeauty != nil {
		cond = append(cond, "with high beauty")
	}
	if d.HeldItem != nil {
		cond = append(cond, "holding "+DisplayName(d.HeldItem.Name))
	}
	if d.KnownMove != nil {
		cond = append(cond, "knowing "+DisplayName(d.KnownMove.Name))
	}
	if d.KnownMoveType != nil {
		cond = append(cond, "knowing a "+Capitalize(d.KnownMoveType.Name)+"-type move")
	}
	if d.UsedMove != nil {
		s := "after using " + DisplayName(d.UsedMove.Name)
		if d.MinMoveCount != nil {
			s += fmt.Sprintf(" %d times", *d.MinMoveCount)
		}
		cond = append(cond, s)
	}
	if d.MinSteps != nil {
		cond = append(cond, fmt.Sprintf("after walking %d steps", *d.MinSteps))
	}
	if d.MinDamageTaken != nil {
		cond = append(cond, fmt.Sprintf("after taking %d damage", *d.MinDamageTaken))
	}
	if d.PartySpecies != nil {
		cond = append(cond, "with "+DisplayName(d.PartySpecies.Name)+" in the party")
	}
	if d.PartyType != nil {
		cond = append(cond, "with a "+Capitalize(d.PartyType.Name)+"-type in the party")
	}
	if d.TradeSpecies != nil {
		cond = append(cond, "for "+DisplayName(d.TradeSpecies.Name))
	}
	if d.RelativePhysicalStats != nil {
		switch *d.RelativePhysicalStats {
		case 1:
			cond = append(cond, "with Attack > Defense")
		case -1:
			cond = append(cond, "with Attack < Defense")
		case 0:
			cond = append(cond, "with Attack = Defense")
		}
	}
	if d.Gender != nil {
		switch *d.Gender {
		case 1:
			cond = append(cond, "(female)")
		case 2:
			cond = append(cond, "(male)")
		}
	}
	switch d.TimeOfDay {
	case "":
	case "day", "night":
		cond = append(cond, "during the "+d.TimeOfDay)
	default:
		cond = append(cond, "at "+d.TimeOfDay)
	}
	if d.NeedsOverworldRain {
		cond = append(cond, "while it rains")
	}
	if d.TurnUpsideDown {
		cond = append(cond, "with the console upside down")
	}
	if d.NeedsMultiplayer {
		cond = append(cond, "in multiplayer")
	}
	if d.Region != nil {
		cond = append(cond, "in "+DisplayName(d.Region.Name))
	}
	return strings.TrimSpace(base + " " + strings.Join(cond, " "))
}

// FindEvo returns the node for species and its parent (nil for the first stage).
func FindEvo(root *EvoNode, species string) (node, parent *EvoNode) {
	if root == nil {
		return nil, nil
	}
	if root.Species == species {
		return root, nil
	}
	for i := range root.Next {
		child := &root.Next[i]
		if n, p := FindEvo(child, species); n != nil {
			if p == nil {
				p = root
			}
			return n, p
		}
	}
	return nil, nil
}

func HowText(how []string) string {
	if len(how) == 0 {
		return "special condition"
	}
	out := how[0]
	for _, h := range how[1:] {
		out += " or " + strings.ToLower(h[:1]) + h[1:]
	}
	return out
}
