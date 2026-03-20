package notifications

import "testing"

func TestPow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		base     int
		exp      int
		expected int
	}{
		{name: "zero exponent", base: 10, exp: 0, expected: 1},
		{name: "small power", base: 2, exp: 5, expected: 32},
		{name: "decimal power", base: 10, exp: 3, expected: 1000},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := pow(tc.base, tc.exp); got != tc.expected {
				t.Fatalf("pow(%d, %d) = %d, expected %d", tc.base, tc.exp, got, tc.expected)
			}
		})
	}
}

func TestGenerateRandomIntWithNDigits(t *testing.T) {
	t.Parallel()

	if got := generateRandomIntWithNDigits(0); got != 0 {
		t.Fatalf("generateRandomIntWithNDigits(0) = %d, expected 0", got)
	}

	for i := 0; i < 50; i++ {
		n := generateRandomIntWithNDigits(4)
		if n < 1000 || n > 9999 {
			t.Fatalf("generated value %d is not a 4-digit integer", n)
		}
	}
}
