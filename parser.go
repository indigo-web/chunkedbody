package chunkedbodyparser

import (
	"fmt"
	"github.com/indigo-web/utils/hex"
	"io"
)

// ChunkedBodyParser parses chunked-encoded bodies in streaming mode. This means, that it returns
// chunk value, that may be incomplete. Or it can return contents of one chunk, even if there's
// more data to parse (and the next chunk is available). So always check, whether there's something
// returned as extra
type ChunkedBodyParser struct {
	state parserState

	settings    Settings
	chunkLength int64
}

// NewChunkedBodyParser returns new *ChunkedBodyParser
func NewChunkedBodyParser(settings Settings) *ChunkedBodyParser {
	return &ChunkedBodyParser{
		state:    eChunkLength1Char,
		settings: settings,
	}
}

// Parse a stream of chunked body. When body is parsed till the end, io.EOF is returned.
// Other data, not belonging to the body, will be returned as extra
func (c *ChunkedBodyParser) Parse(data []byte, trailer bool) (chunk, extra []byte, err error) {
	var offset int64

	switch c.state {
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
		panic(fmt.Sprintf("BUG: unknown state: %v", c.state))
	}

chunkLength1Char:
	if !hex.Is(data[offset]) {
		return nil, nil, ErrBadRequest
	}

	c.chunkLength = int64(hex.Un(data[offset]))
	offset++
	c.state = eChunkLength
	goto chunkLength

chunkLength:
	for ; offset < int64(len(data)); offset++ {
		switch data[offset] {
		case '\r':
			offset++
			c.state = eChunkLengthCR
			goto chunkLengthCR
		case '\n':
			offset++
			c.state = eChunkLengthCRLF
			goto chunkLengthCRLF
		default:
			if !hex.Is(data[offset]) {
				return nil, nil, ErrBadRequest
			}

			c.chunkLength = (c.chunkLength << 4) | int64(hex.Un(data[offset]))
			if c.chunkLength > c.settings.MaxChunkSize {
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
		c.state = eChunkLengthCRLF
		goto chunkLengthCRLF
	default:
		return nil, nil, ErrBadRequest
	}

chunkLengthCRLF:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch c.chunkLength {
	case 0:
		switch data[offset] {
		case '\r':
			offset++
			c.state = eLastChunkCR
			goto lastChunkCR
		case '\n':
			c.state = eChunkLength1Char

			return nil, data[offset+1:], io.EOF
		default:
			if !trailer {
				return nil, nil, ErrBadRequest
			}

			offset++
			c.state = eFooter
			goto footer
		}
	default:
		c.state = eChunkBody
		goto chunkBody
	}

chunkBody:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	if int64(len(data[offset:])) > c.chunkLength {
		c.state = eChunkBodyEnd

		return data[offset : offset+c.chunkLength], data[offset+c.chunkLength:], nil
	}

	c.chunkLength -= int64(len(data[offset:]))

	return data[offset:], nil, nil

chunkBodyEnd:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\r':
		offset++
		c.state = eChunkBodyCR
		goto chunkBodyCR
	case '\n':
		offset++
		c.state = eChunkBodyCRLF
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
		c.state = eChunkBodyCRLF
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
		c.state = eLastChunkCR
		goto lastChunkCR
	case '\n':
		if !trailer {
			c.state = eChunkLength1Char

			return nil, data[offset+1:], io.EOF
		}

		offset++
		c.state = eFooter
		goto footer
	default:
		c.chunkLength = int64(hex.Un(data[offset]))
		if c.chunkLength > c.settings.MaxChunkSize {
			return nil, nil, ErrTooLarge
		}

		offset++
		c.state = eChunkLength
		goto chunkLength
	}

lastChunkCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		if !trailer {
			c.state = eChunkLength1Char

			return nil, data[offset+1:], io.EOF
		}

		offset++
		c.state = eFooter
		goto footer
	default:
		return nil, nil, ErrBadRequest
	}

footer:
	for ; offset < int64(len(data)); offset++ {
		switch data[offset] {
		case '\r':
			offset++
			c.state = eFooterCR
			goto footerCR
		case '\n':
			offset++
			c.state = eFooterCRLF
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
		c.state = eFooterCRLF
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
		c.state = eFooterCRLFCR
		goto footerCRLFCR
	case '\n':
		c.state = eChunkLength1Char

		return nil, data[offset+1:], io.EOF
	default:
		offset++
		c.state = eFooter
		goto footer
	}

footerCRLFCR:
	if offset >= int64(len(data)) {
		return nil, nil, nil
	}

	switch data[offset] {
	case '\n':
		c.state = eChunkLength1Char

		return nil, data[offset+1:], io.EOF
	default:
		return nil, nil, ErrBadRequest
	}
}
