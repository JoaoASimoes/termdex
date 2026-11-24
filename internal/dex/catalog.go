package dex

import (
	"fmt"
	"sort"
)

// Catalogs: the full lists of moves and items, for browsing them on their own.

// ItemInfo describes one item.
type ItemInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Category    string   `json:"category"`
	Attributes  []string `json:"attributes"`
	FlingPower  *int     `json:"fling_power"`
	ShortEffect string   `json:"short_effect"`
	Effect      string   `json:"effect"`
	Flavor      string   `json:"flavor"`
	Sprite      string   `json:"sprite"`
	HeldBy      []string `json:"held_by"` // wild Pokémon that may hold it
}

// MoveNames returns every move name (cached).
func (d *Client) MoveNames() ([]string, error) { return d.listNames("move_names", "/move?limit=5000") }

// ItemNames returns every item name (cached).
func (d *Client) ItemNames() ([]string, error) { return d.listNames("item_names", "/item?limit=10000") }

func (d *Client) listNames(cacheName, path string) ([]string, error) {
	return cached(d, cacheName, func() ([]string, error) {
		var listing struct{ Results []namedURL }
		if err := d.get(apiBase+path, &listing); err != nil {
			return nil, err
		}
		out := make([]string, len(listing.Results))
		for i, r := range listing.Results {
			out[i] = r.Name
		}
		sort.Strings(out)
		return out, nil
	})
}

// Item loads one item (cached).
func (d *Client) Item(name string) (ItemInfo, error) {
	return cached(d, "item_"+name, func() (ItemInfo, error) {
		var raw struct {
			Name              string
			Names             []apiName
			Category          namedURL
			Attributes        []namedURL
			FlingPower        *int             `json:"fling_power"`
			EffectEntries     []apiEffectEntry `json:"effect_entries"`
			FlavorTextEntries []struct {
				langText
				Text string `json:"text"`
			} `json:"flavor_text_entries"`
			Sprites struct {
				Default string `json:"default"`
			}
			HeldByPokemon []struct {
				Pokemon namedURL
			} `json:"held_by_pokemon"`
		}
		if err := d.get(apiBase+"/item/"+name, &raw); err != nil {
			return ItemInfo{}, err
		}
		it := ItemInfo{
			Name:        raw.Name,
			DisplayName: englishName(raw.Names, raw.Name),
			Category:    raw.Category.Name,
			FlingPower:  raw.FlingPower,
			Sprite:      raw.Sprites.Default,
		}
		it.ShortEffect, it.Effect = englishEffect(raw.EffectEntries)
		for i := len(raw.FlavorTextEntries) - 1; i >= 0; i-- { // newest English text
			if e := raw.FlavorTextEntries[i]; e.Language.Name == "en" {
				it.Flavor = cleanText(e.Text)
				break
			}
		}
		for _, a := range raw.Attributes {
			it.Attributes = append(it.Attributes, a.Name)
		}
		for _, h := range raw.HeldByPokemon {
			it.HeldBy = append(it.HeldBy, h.Pokemon.Name)
		}
		return it, nil
	})
}

// PokemonOfType returns every Pokémon (forms included) that has type t (cached).
func (d *Client) PokemonOfType(t string) ([]string, error) {
	return cached(d, "type_members_"+t, func() ([]string, error) {
		var raw struct {
			Pokemon []struct {
				Pokemon namedURL
			}
		}
		if err := d.get(apiBase+"/type/"+t, &raw); err != nil {
			return nil, err
		}
		out := make([]string, len(raw.Pokemon))
		for i, p := range raw.Pokemon {
			out[i] = p.Pokemon.Name
		}
		return out, nil
	})
}

// SpeciesOfGeneration returns the species introduced in generation n (1-9…, cached).
func (d *Client) SpeciesOfGeneration(n int) ([]string, error) {
	return cached(d, fmt.Sprintf("generation_%d", n), func() ([]string, error) {
		var raw struct {
			PokemonSpecies []namedURL `json:"pokemon_species"`
		}
		if err := d.get(fmt.Sprintf("%s/generation/%d", apiBase, n), &raw); err != nil {
			return nil, err
		}
		out := make([]string, len(raw.PokemonSpecies))
		for i, s := range raw.PokemonSpecies {
			out[i] = s.Name
		}
		return out, nil
	})
}
