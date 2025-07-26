package types

import (
	"math/big"
	"strconv"
	"strings"
)

// Helper function for decimal addition using big.Int with 10^20 expansion
func addDecimals(nums ...string) string {
	if len(nums) == 0 {
		return "0.00"
	}

	// Use 10^20 as the scaling factor for precision
	scale := new(big.Int)
	scale.SetString("100000000000000000000", 10) // 10^20

	sum := big.NewInt(0)

	for _, num := range nums {
		if num == "" {
			continue
		}

		// Parse the decimal string
		val, err := parseDecimalToBigInt(num, scale)
		if err != nil {
			continue // Skip invalid numbers
		}

		sum.Add(sum, val)
	}

	// Convert back to decimal string
	return bigIntToDecimalString(sum, scale)
}

// parseDecimalToBigInt converts a decimal string to big.Int scaled by the given factor
func parseDecimalToBigInt(s string, scale *big.Int) (*big.Int, error) {
	// Handle empty or zero values
	if s == "" || s == "0" || s == "0.00" {
		return big.NewInt(0), nil
	}

	// Split by decimal point
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, strconv.ErrSyntax
	}

	// Parse integer part
	intPart := parts[0]
	intVal, ok := new(big.Int).SetString(intPart, 10)
	if !ok {
		return nil, strconv.ErrSyntax
	}

	// Scale the integer part
	result := new(big.Int).Mul(intVal, scale)

	// Handle fractional part if present
	if len(parts) == 2 {
		fracPart := parts[1]
		// Pad or truncate to 20 decimal places (matching our scale)
		for len(fracPart) < 20 {
			fracPart += "0"
		}
		if len(fracPart) > 20 {
			fracPart = fracPart[:20]
		}

		fracVal, ok := new(big.Int).SetString(fracPart, 10)
		if !ok {
			return nil, strconv.ErrSyntax
		}

		result.Add(result, fracVal)
	}

	return result, nil
}

// bigIntToDecimalString converts a scaled big.Int back to decimal string
func bigIntToDecimalString(val *big.Int, scale *big.Int) string {
	if val.Sign() == 0 {
		return "0.00"
	}

	// Divide by scale to get integer and remainder
	intPart := new(big.Int)
	fracPart := new(big.Int)
	intPart.DivMod(val, scale, fracPart)

	// Convert integer part
	intStr := intPart.String()

	// Convert fractional part
	fracStr := fracPart.String()
	// Pad with leading zeros to make it 20 digits
	for len(fracStr) < 20 {
		fracStr = "0" + fracStr
	}

	// Trim trailing zeros, but keep at least 2 decimal places
	fracStr = strings.TrimRight(fracStr, "0")
	if len(fracStr) < 2 {
		for len(fracStr) < 2 {
			fracStr += "0"
		}
	}

	return intStr + "." + fracStr
}
