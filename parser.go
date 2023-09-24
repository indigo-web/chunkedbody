package chunkedbody

import (
	"fmt"
	"github.com/indigo-web/utils/hex"
	"io"
)

// Parser parses chunked-encoded bodies in streaming mode. This means, that it returns
// chunk value, that may be incomplete. Or it can return contents of one chunk, even if there's
// more data to parse (and the next chunk is available). So always check, whether there's something
// returned as extra
type Parser struct {
	state parserState

	settings    Settings
	chunkLength int64
}

// NewParser returns new *ChunkedBodyParser
func NewParser(settings Settings) *Parser {
	return &Parser{
		state:    eChunkLength1Char,
		settings: settings,
	}
}

// Parse a stream of chunked body. When body is parsed till the end, io.EOF is returned.
// Other data, not belonging to the body, will be returned as extra
func (p *Parser) Parse(data []byte, trailer bool) (chunk, extra []byte, err error) {
	var offset int64

	switch p.state {
	case eChunkLength1Char:
		goto chunkLength1Char
	case eChunkLength:
		goto chunkLength
	case eChunkLengthCR:
		goto chunkLengthCR
	case eChunkLengthCRLF:
		goto chunkLengthCRLF
	case eChunkBody:
		goto chunkBody
	case eChunkBodyEnd:
		goto chunkBodyEnd
	case eChunkBodyCR:
		goto chunkBodyCR
	case eChunkBodyCRLF:
		goto chunkBodyCRLF
	case eLastChunkCR:
		goto lastChunkCR
	case eFooter:
		goto footer
	case eFooterCR:
		goto footerCR
	case eFooterCRLF:
		goto footerCRLF
	case eFooterCRLFCR:
		goto footerCRLFCR
	default:
		panic(fmt.Sprintf("BUG: unknown state: %v", p.state))
	}

chunkLength1Char:
	if !hex.Is(data[offset]) {
		return nil, nil, ErrBadRequest
	}

	p.chunkLength = int64(hex.Un(data[offset]))
	offset++
	p.state = eChunkLength
	goto chunkLength

chunkLength:
	for ; offset < int64(len(data)); offset++ {
		switch data[offset] {
		case '\r':
			offset++
			p.state = eChunkLengthCR
			goto chunkLengthCR
		case '\n':
			offset++
			p.state = eChunkLengthCRLF
			goto chunkLengthCRLF
		default:
			if !hex.Is(data[offset]) {
				return nil, nil, ErrBadRequest
			}

			p.chunkLength = (p.chunkLength << 4) | int64(hex.Un(data[offset]))
			if p.chunkLength > p.settings.MaxChunkSize {
				return nil, nil, ErrTooLarge
			}
		}
	}

	return nil, nil, nil

chunkLengthCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		offset++
		p.state = eChunkLengthCRLF
		goto chunkLengthCRLF
	default:
		return nil, nil, ErrBadRequest
	}

chunkLengthCRLF:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch p.chunkLength {
	case 0:
		switch data[offset] {
		case '\r':
			offset++
			p.state = eLastChunkCR
			goto lastChunkCR
		case '\n':
			p.state = eChunkLength1Char

			return nil, data[offset+1:], io.EOF
		default:
			if !trailer {
				return nil, nil, ErrBadRequest
			}

			offset++
			p.state = eFooter
			goto footer
		}
	default:
		p.state = eChunkBody
		goto chunkBody
	}

chunkBody:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	if int64(len(data[offset:])) > p.chunkLength {
		p.state = eChunkBodyEnd

		return data[offset : offset+p.chunkLength], data[offset+p.chunkLength:], nil
	}

	p.chunkLength -= int64(len(data[offset:]))

	return data[offset:], nil, nil

chunkBodyEnd:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\r':
		offset++
		p.state = eChunkBodyCR
		goto chunkBodyCR
	case '\n':
		offset++
		p.state = eChunkBodyCRLF
		goto chunkBodyCRLF
	default:
		return nil, nil, ErrBadRequest
	}

chunkBodyCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		offset++
		p.state = eChunkBodyCRLF
		goto chunkBodyCRLF
	default:
		return nil, nil, ErrBadRequest
	}

chunkBodyCRLF:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\r':
		offset++
		p.state = eLastChunkCR
		goto lastChunkCR
	case '\n':
		if !trailer {
			p.state = eChunkLength1Char

			return nil, data[offset+1:], io.EOF
		}

		offset++
		p.state = eFooter
		goto footer
	default:
		p.chunkLength = int64(hex.Un(data[offset]))
		if p.chunkLength > p.settings.MaxChunkSize {
			return nil, nil, ErrTooLarge
		}

		offset++
		p.state = eChunkLength
		goto chunkLength
	}

lastChunkCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		if !trailer {
			p.state = eChunkLength1Char

			return nil, data[offset+1:], io.EOF
		}

		offset++
		p.state = eFooter
		goto footer
	default:
		return nil, nil, ErrBadRequest
	}

footer:
	for ; offset < int64(len(data)); offset++ {
		switch data[offset] {
		case '\r':
			offset++
			p.state = eFooterCR
			goto footerCR
		case '\n':
			offset++
			p.state = eFooterCRLF
			goto footerCRLF
		}
	}

	return nil, nil, nil

footerCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		offset++
		p.state = eFooterCRLF
		goto footerCRLF
	default:
		return nil, nil, ErrBadRequest
	}

footerCRLF:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\r':
		offset++
		p.state = eFooterCRLFCR
		goto footerCRLFCR
	case '\n':
		p.state = eChunkLength1Char

		return nil, data[offset+1:], io.EOF
	default:
		offset++
		p.state = eFooter
		goto footer
	}

footerCRLFCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		p.state = eChunkLength1Char

		return nil, data[offset+1:], io.EOF
	default:
		return nil, nil, ErrBadRequest
	}
}
