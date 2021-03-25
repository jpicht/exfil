package exfil

var (
	Mapping = map[rune]rune{
		'A': 'α', 'B': 'β', 'C': 'π', 'D': 'δ', 'E': 'ε', 'F': 'ϝ',
		'G': 'γ', 'H': 'σ', 'I': 'ι', 'J': 'φ', 'K': 'κ', 'L': 'λ',
		'M': 'χ', 'N': 'ν', 'O': 'ο', 'P': 'θ', 'Q': 'ψ', 'R': 'ρ',
		'S': 'ς', 'T': 'τ', 'U': 'μ', 'V': 'ω', 'W': 'Ϟ', 'X': 'ξ',
		'Y': 'υ', 'Z': 'ζ', '+': 'ƕ', '/': 'η',
	}

	ReverseMapping map[rune]rune
)

func init() {
	ReverseMapping = make(map[rune]rune, len(Mapping))
	for r, rr := range Mapping {
		ReverseMapping[rr] = r
	}
}
