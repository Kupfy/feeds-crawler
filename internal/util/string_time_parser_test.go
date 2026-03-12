package util

import "testing"

func TestParseServing(t *testing.T) {
	tests := []struct {
		input            string
		expectedQuantity int
		expectedCourse   string
		expectedMakes    string
	}{
		{input: "Serves 6", expectedQuantity: 6, expectedCourse: "", expectedMakes: ""},
		{input: "Serves 6 as a main", expectedQuantity: 6, expectedCourse: "main", expectedMakes: ""},
		{input: "Makes 12 juicy buns", expectedQuantity: 0, expectedCourse: "", expectedMakes: "12 juicy buns"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			serving := ParseServing(test.input)
			if serving == nil {
				t.Errorf("expected a serving, got nil")
				return
			}
			if serving.Quantity != test.expectedQuantity {
				t.Errorf("expected quantity %d, got %d", test.expectedQuantity, serving.Quantity)
			}
			if serving.Course != test.expectedCourse {
				t.Errorf("expected course %s, got %s", test.expectedCourse, serving.Course)
			}
			if serving.Makes != test.expectedMakes {
				t.Errorf("expected makes %s, got %s", test.expectedMakes, serving.Makes)
			}
		})
	}
}
