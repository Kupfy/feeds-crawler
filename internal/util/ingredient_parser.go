package util

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
)

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)

	replacements := map[string]string{
		"½": "0.5",
		"¼": "0.25",
		"¾": "0.75",
		"⅛": "0.125",
		"–": "-",
	}

	for k, v := range replacements {
		s = strings.ReplaceAll(s, k, v)
	}

	return s
}

func tokenize(s string) []string {
	s = strings.ReplaceAll(s, ",", " , ")
	s = strings.ReplaceAll(s, "(", " ( ")
	s = strings.ReplaceAll(s, ")", " ) ")
	return strings.Fields(s)
}

func extractQuantity(tokens []string) (float64, int, bool) {
	for i, t := range tokens {
		if t == "a" || t == "an" {
			return 1, i, true
		}
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			return v, i, true
		}
	}
	return 0, -1, false
}

func extractUnit(tokens []string, start int) (unit.Unit, int) {
	if start+1 >= len(tokens) {
		return unit.Piece, -1
	}

	return unit.NewUnit(tokens[start+1]), start + 1
}

func extractInstruction(s string) *string {
	if idx := strings.Index(s, ","); idx != -1 {
		instr := strings.TrimSpace(s[idx+1:])
		if instr != "" {
			return &instr
		}
	}
	return nil
}

func extractComponent(s string) *string {
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		comp := strings.TrimSpace(parts[0])
		if strings.HasPrefix(comp, "for ") {
			return &comp
		}
	}
	return nil
}

func ParseIngredient(raw string, ingredientsDict map[string]entity.Ingredient) (dto.IngredientsItem, error) {
	s := normalize(raw)
	tokens := tokenize(s)

	qty, qtyIdx, ok := extractQuantity(tokens)
	if !ok {
		return dto.IngredientsItem{}, errors.New("no quantity found")
	}

	u, unitIdx := extractUnit(tokens, qtyIdx)

	instr := extractInstruction(s)
	comp := extractComponent(s)

	nameTokens := tokens
	if unitIdx != -1 {
		nameTokens = tokens[unitIdx+1:]
	}

	name := nameTokens[len(nameTokens)-1]
	nameCandidate := strings.Join(nameTokens, " ")
	if ingredient, ok := ingredientsDict[nameCandidate]; ok {
		name = ingredient.Name
	}

	return dto.IngredientsItem{
		Name:        name,
		Quantity:    qty,
		Unit:        u,
		Component:   comp,
		Instruction: instr,
	}, nil
}
