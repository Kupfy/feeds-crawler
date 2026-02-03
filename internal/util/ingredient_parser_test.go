package util

import (
	"testing"

	"github.com/Kupfy/feeds-crawler/internal/data/dto"
	"github.com/Kupfy/feeds-crawler/internal/data/entity"
	"github.com/Kupfy/feeds-crawler/internal/data/enum/unit"
)

var ingredientsDict map[string]*entity.Ingredient

func loadIngredients() {
	ingredientsDict = map[string]*entity.Ingredient{
		"cloves of garlic": ToPtr(entity.Ingredient{CanonicalName: "garlic"}),
		"eggplant":         ToPtr(entity.Ingredient{CanonicalName: "eggplant"}),
		"apple":            ToPtr(entity.Ingredient{CanonicalName: "apple"}),
		"thai basil":       ToPtr(entity.Ingredient{CanonicalName: "thai basil"}),
		"salt":             ToPtr(entity.Ingredient{CanonicalName: "salt"}),
		"clove":            ToPtr(entity.Ingredient{CanonicalName: "clove"}),
	}

	ingredientsDict["garlic cloves"] = ingredientsDict["cloves of garlic"]
	ingredientsDict["apples"] = ingredientsDict["apple"]
}

func TestParseIngredient(t *testing.T) {
	tests := []struct {
		Name     string
		Input    string
		ExpIng   dto.IngredientsItem
		ExpQuant float64
	}{
		{
			Name:  "instructions and pieces 1",
			Input: "5 - garlic cloves, finely chopped",
			ExpIng: dto.IngredientsItem{
				Name:        "cloves of garlic",
				Quantity:    5,
				Unit:        unit.Piece,
				Instruction: ToPtr("finely chopped"),
			},
		},
		{
			Name:  "instructions and pieces 2",
			Input: "5 cloves of garlic, crushed",
			ExpIng: dto.IngredientsItem{
				Name:        "cloves of garlic",
				Quantity:    5,
				Unit:        unit.Piece,
				Instruction: ToPtr("crushed"),
			},
		},
		{
			Name:  "quantity and unit",
			Input: "15g thai basil",
			ExpIng: dto.IngredientsItem{
				Name:     "thai basil",
				Quantity: 15,
				Unit:     unit.Gram,
			},
		},
		{
			Name:  "vernacular",
			Input: "a pinch of salt",
			ExpIng: dto.IngredientsItem{
				Name:     "salt",
				Quantity: 1,
				Unit:     unit.Pinch,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if parsed, err := ParseIngredient(test.Input, ingredientsDict); err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if parsed != test.ExpIng {
				t.Errorf("expected %s, got %s", test.ExpIng, parsed)
			}
		})
	}
}
