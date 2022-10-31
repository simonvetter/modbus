package mbserver

type WordOrder uint

const (
	// word order of 32-bit registers
	HIGH_WORD_FIRST WordOrder = 1
	LOW_WORD_FIRST  WordOrder = 2
)
