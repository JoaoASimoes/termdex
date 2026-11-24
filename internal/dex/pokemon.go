package dex

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ----------------------------------------------------------------------------- models

type Stat struct {
	Name string `json:"name"`
	Base int    `json:"base"`
}

type AbilityRef struct {
	Name   string `json:"name"`
	Hidden bool   `json:"hidden"`
}

// MoveRef is one way a Pokémon learns a move in a given game.
type MoveRef struct {
	Name         string `json:"name"`
	Method       string `json:"method"` // level-up, machine, egg, tutor, …
	Level        int    `json:"level"`
	VersionGroup string `json:"version_group"`
}

// Learnset is the move pool of a Pokémon in one game (version Group).
type Learnset struct {
	VersionGroup string    `json:"version_group"`
	Moves        []MoveRef `json:"moves"`
}

type Pokemon struct {
	Name      string       `json:"name"`
	ID        int          `json:"id"`
	Types     []string     `json:"types"`
	Abilities []AbilityRef `json:"abilities"`
	Stats     []Stat       `json:"stats"`
	Height    int          `json:"height"` // decimetres
	Weight    int          `json:"weight"` // hectograms
	Sprite    string       `json:"sprite"` // front sprite URL (PokeAPI sprites repo)
	// SpriteFemale is set when the female looks different (e.g. Pikachu's heart-shaped tail).
	SpriteFemale string     `json:"sprite_female"`
	Learnsets    []Learnset `json:"learnsets"` // newest game first

	// From the species endpoint.
	Species    string `json:"species"`
	Genus      string `json:"genus"`
	Flavor     string `json:"flavor"`
	Generation string `json:"generation"`
	Legendary  bool   `json:"legendary"`
	Mythical   bool   `json:"mythical"`

	Evolution *EvoNode `json:"evolution"` // whole chain; nil if unknown
	// Varieties are all Pokémon of this species (default form first): Mega, Gmax,
	// regional and gender forms, …
	Varieties []string `json:"varieties"`
}

func (p Pokemon) AbilityNames() []string {
	out := make([]string, len(p.Abilities))
	for i, a := range p.Abilities {
		out[i] = a.Name
	}
	return out
}

// UniqueMoveCount counts distinct moves across all games.
func (p Pokemon) UniqueMoveCount() int {
	seen := map[string]bool{}
	for _, ls := range p.Learnsets {
		for _, m := range ls.Moves {
			seen[m.Name] = true
		}
	}
	return len(seen)
}

func (p Pokemon) StatTotal() int {
	t := 0
	for _, s := range p.Stats {
		t += s.Base
	}
	return t
}

type AbilityInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	ShortEffect string `json:"short_effect"`
	Effect      string `json:"effect"`
	Flavor      string `json:"flavor"`
	Generation  string `json:"generation"`
}

type MoveInfo struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	Type         string `json:"type"`
	Class        string `json:"class"` // physical, special, status
	Power        *int   `json:"power"`
	Accuracy     *int   `json:"accuracy"`
	PP           *int   `json:"pp"`
	Priority     int    `json:"priority"`
	EffectChance *int   `json:"effect_chance"`
	ShortEffect  string `json:"short_effect"`
	Effect       string `json:"effect"`
	Flavor       string `json:"flavor"`
	Generation   string `json:"generation"`
}

// ----------------------------------------------------------------------------- raw API shapes

type langText struct {
	Language namedURL `json:"language"`
}

type apiEffectEntry struct {
	langText
	Effect      string `json:"effect"`
	ShortEffect string `json:"short_effect"`
}

type apiFlavorEntry struct {
	langText
	FlavorText string `json:"flavor_text"`
}

type apiName struct {
	langText
	Name string `json:"name"`
}

func englishEffect(entries []apiEffectEntry) (short, long string) {
	for _, e := range entries {
		if e.Language.Name == "en" {
			return cleanText(e.ShortEffect), cleanText(e.Effect)
		}
	}
	return "", ""
}

// latestEnglishFlavor returns the last English flavor text (entries are oldest first).
func latestEnglishFlavor(entries []apiFlavorEntry) string {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Language.Name == "en" {
			return cleanText(entries[i].FlavorText)
		}
	}
	return ""
}

func englishName(names []apiName, fallback string) string {
	for _, n := range names {
		if n.Language.Name == "en" {
			return n.Name
		}
	}
	return DisplayName(fallback)
}

func cleanText(s string) string {
	s = strings.NewReplacer("­\n", "", "\f", " ", "\n", " ", "\r", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// ----------------------------------------------------------------------------- version groups

var (
	vgOrderOnce sync.Once
	vgOrder     map[string]int
)

// versionGroupOrder returns the chronological order of every version Group (cached).
func (d *Client) versionGroupOrder() map[string]int {
	vgOrderOnce.Do(func() {
		order, err := cached(d, "version_groups", func() (map[string]int, error) {
			var listing struct{ Results []namedURL }
			if err := d.get(apiBase+"/version-Group?limit=200", &listing); err != nil {
				return nil, err
			}
			out := map[string]int{}
			var mu sync.Mutex
			var wg sync.WaitGroup
			var firstErr error
			for _, vg := range listing.Results {
				wg.Add(1)
				go func(vg namedURL) {
					defer wg.Done()
					var r struct{ Order int }
					err := d.get(vg.URL, &r)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						firstErr = err
						return
					}
					out[vg.Name] = r.Order
				}(vg)
			}
			wg.Wait()
			return out, firstErr
		})
		if err != nil {
			order = map[string]int{}
		}
		vgOrder = order
	})
	return vgOrder
}

// ----------------------------------------------------------------------------- loaders

const cacheVersion = "v7_" // bump when the cached shapes change

func (d *Client) Pokemon(name string) (Pokemon, error) {
	return cached(d, cacheVersion+"pokemon_"+name, func() (Pokemon, error) {
		var raw struct {
			Name   string
			ID     int
			Height int
			Weight int
			Types  []struct {
				Slot int
				Type namedURL
			}
			Abilities []struct {
				Ability  namedURL
				IsHidden bool `json:"is_hidden"`
				Slot     int
			}
			Stats []struct {
				BaseStat int `json:"base_stat"`
				Stat     namedURL
			}
			Moves []struct {
				Move    namedURL
				Details []struct {
					Level        int      `json:"level_learned_at"`
					Method       namedURL `json:"move_learn_method"`
					VersionGroup namedURL `json:"version_group"`
				} `json:"version_group_details"`
			}
			Species namedURL
			Sprites struct {
				FrontDefault string `json:"front_default"`
				FrontFemale  string `json:"front_female"`
			}
		}
		if err := d.get(apiBase+"/pokemon/"+name, &raw); err != nil {
			return Pokemon{}, err
		}

		p := Pokemon{Name: raw.Name, ID: raw.ID, Height: raw.Height, Weight: raw.Weight, Species: raw.Species.Name,
			Sprite: raw.Sprites.FrontDefault, SpriteFemale: raw.Sprites.FrontFemale}
		sort.Slice(raw.Types, func(i, j int) bool { return raw.Types[i].Slot < raw.Types[j].Slot })
		for _, t := range raw.Types {
			p.Types = append(p.Types, t.Type.Name)
		}
		sort.Slice(raw.Abilities, func(i, j int) bool { return raw.Abilities[i].Slot < raw.Abilities[j].Slot })
		for _, a := range raw.Abilities {
			p.Abilities = append(p.Abilities, AbilityRef{a.Ability.Name, a.IsHidden})
		}
		for _, s := range raw.Stats {
			p.Stats = append(p.Stats, Stat{s.Stat.Name, s.BaseStat})
		}

		// Group moves by game, newest game first.
		order := d.versionGroupOrder()
		byVG := map[string][]MoveRef{}
		seen := map[MoveRef]bool{}
		for _, m := range raw.Moves {
			for _, det := range m.Details {
				ref := MoveRef{m.Move.Name, det.Method.Name, det.Level, det.VersionGroup.Name}
				if !seen[ref] {
					seen[ref] = true
					byVG[ref.VersionGroup] = append(byVG[ref.VersionGroup], ref)
				}
			}
		}
		for vg, moves := range byVG {
			sortMoves(moves)
			p.Learnsets = append(p.Learnsets, Learnset{vg, moves})
		}
		sort.Slice(p.Learnsets, func(i, j int) bool {
			oi, oj := order[p.Learnsets[i].VersionGroup], order[p.Learnsets[j].VersionGroup]
			if oi != oj {
				return oi > oj
			}
			return p.Learnsets[i].VersionGroup < p.Learnsets[j].VersionGroup
		})

		var species struct {
			Genera []struct {
				langText
				Genus string
			}
			FlavorTextEntries []apiFlavorEntry `json:"flavor_text_entries"`
			Generation        namedURL
			IsLegendary       bool                  `json:"is_legendary"`
			IsMythical        bool                  `json:"is_mythical"`
			EvolutionChain    *struct{ URL string } `json:"evolution_chain"`
			Varieties         []struct {
				IsDefault bool `json:"is_default"`
				Pokemon   namedURL
			}
		}
		if raw.Species.URL != "" && d.get(raw.Species.URL, &species) == nil {
			for _, g := range species.Genera {
				if g.Language.Name == "en" {
					p.Genus = g.Genus
				}
			}
			p.Flavor = latestEnglishFlavor(species.FlavorTextEntries)
			p.Generation = species.Generation.Name
			p.Legendary, p.Mythical = species.IsLegendary, species.IsMythical
			for _, v := range species.Varieties {
				if v.IsDefault {
					p.Varieties = append([]string{v.Pokemon.Name}, p.Varieties...)
				} else {
					p.Varieties = append(p.Varieties, v.Pokemon.Name)
				}
			}
			if species.EvolutionChain != nil && species.EvolutionChain.URL != "" {
				var chain struct{ Chain apiEvoChain }
				if d.get(species.EvolutionChain.URL, &chain) == nil {
					tree := buildEvoTree(chain.Chain)
					p.Evolution = &tree
				}
			}
		}
		return p, nil
	})
}

var methodOrder = map[string]int{"level-up": 0, "machine": 1, "tutor": 2, "egg": 3}

func methodRank(m string) int {
	if r, ok := methodOrder[m]; ok {
		return r
	}
	return 4
}

func sortMoves(moves []MoveRef) {
	sort.SliceStable(moves, func(i, j int) bool {
		a, b := moves[i], moves[j]
		if ra, rb := methodRank(a.Method), methodRank(b.Method); ra != rb {
			return ra < rb
		}
		if a.Method == "level-up" && a.Level != b.Level {
			return a.Level < b.Level
		}
		return a.Name < b.Name
	})
}

func (d *Client) Ability(name string) (AbilityInfo, error) {
	return cached(d, "ability_"+name, func() (AbilityInfo, error) {
		var raw struct {
			Name              string
			Names             []apiName
			EffectEntries     []apiEffectEntry `json:"effect_entries"`
			FlavorTextEntries []apiFlavorEntry `json:"flavor_text_entries"`
			Generation        namedURL
		}
		if err := d.get(apiBase+"/ability/"+name, &raw); err != nil {
			return AbilityInfo{}, err
		}
		a := AbilityInfo{
			Name:        raw.Name,
			DisplayName: englishName(raw.Names, raw.Name),
			Flavor:      latestEnglishFlavor(raw.FlavorTextEntries),
			Generation:  raw.Generation.Name,
		}
		a.ShortEffect, a.Effect = englishEffect(raw.EffectEntries)
		return a, nil
	})
}

func (d *Client) Move(name string) (MoveInfo, error) {
	return cached(d, "move_"+name, func() (MoveInfo, error) {
		var raw struct {
			Name              string
			Names             []apiName
			Type              namedURL
			DamageClass       namedURL `json:"damage_class"`
			Power             *int
			Accuracy          *int
			PP                *int
			Priority          int
			EffectChance      *int             `json:"effect_chance"`
			EffectEntries     []apiEffectEntry `json:"effect_entries"`
			FlavorTextEntries []apiFlavorEntry `json:"flavor_text_entries"`
			Generation        namedURL
		}
		if err := d.get(apiBase+"/move/"+name, &raw); err != nil {
			return MoveInfo{}, err
		}
		m := MoveInfo{
			Name:         raw.Name,
			DisplayName:  englishName(raw.Names, raw.Name),
			Type:         raw.Type.Name,
			Class:        raw.DamageClass.Name,
			Power:        raw.Power,
			Accuracy:     raw.Accuracy,
			PP:           raw.PP,
			Priority:     raw.Priority,
			EffectChance: raw.EffectChance,
			Flavor:       latestEnglishFlavor(raw.FlavorTextEntries),
			Generation:   raw.Generation.Name,
		}
		m.ShortEffect, m.Effect = englishEffect(raw.EffectEntries)
		return m, nil
	})
}

// ----------------------------------------------------------------------------- formatting helpers

var romanGen = map[string]string{
	"generation-i": "I", "generation-ii": "II", "generation-iii": "III", "generation-iv": "IV",
	"generation-v": "V", "generation-vi": "VI", "generation-vii": "VII", "generation-viii": "VIII",
	"generation-ix": "IX", "generation-x": "X",
}

func GenLabel(g string) string {
	if r, ok := romanGen[g]; ok {
		return "Gen " + r
	}
	return DisplayName(g)
}

// WithChance fills in the "$effect_chance" placeholder used in PokeAPI effect texts.
func WithChance(text string, chance *int) string {
	if chance == nil {
		return strings.ReplaceAll(text, "$effect_chance% ", "")
	}
	return strings.ReplaceAll(text, "$effect_chance", strconv.Itoa(*chance))
}

var StatLabels = map[string]string{
	"hp": "HP", "attack": "Attack", "defense": "Defense",
	"special-attack": "Sp. Atk", "special-defense": "Sp. Def", "speed": "Speed",
}

func MethodLabel(m MoveRef) string {
	switch m.Method {
	case "level-up":
		if m.Level <= 1 {
			return "Lv 1"
		}
		return "Lv " + strconv.Itoa(m.Level)
	case "machine":
		return "TM"
	case "egg":
		return "Egg"
	case "tutor":
		return "Tutor"
	default:
		return Capitalize(strings.SplitN(m.Method, "-", 2)[0])
	}
}
