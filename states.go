package chunkedbody

type parserState int

const (
	eChunkLength1Char parserState = iota + 1
	eChunkLength
	eChunkLengthCR
	eChunkLengthCRLF
	eChunkBody
	eChunkBodyEnd
	eChunkBodyCR
	eChunkBodyCRLF
	eLastChunkCR
	eFooter
	eFooterCR
	eFooterCRLF
	eFooterCRLFCR
)
