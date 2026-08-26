package xmlq

import (
	"bytes"
	"encoding/xml"
	"strings"
	"unicode"
	"unicode/utf8"
)

type pathSeg struct {
	Space, Local string
}

func parseMaskPath(name string) []pathSeg {
	var out []pathSeg
	for _, part := range strings.Split(name, "/") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.LastIndex(part, ":"); i >= 0 {
			out = append(out, pathSeg{Space: part[:i], Local: part[i+1:]})
		} else {
			out = append(out, pathSeg{Local: part})
		}
	}
	return out
}

func nameMatches(name xml.Name, local, space string) bool {
	if !strings.EqualFold(name.Local, local) {
		return false
	}
	if name.Space != "" && space != "" {
		return strings.EqualFold(name.Space, space)
	}
	return true
}

func pathMatches(path []xml.Name, parts []pathSeg, targetSpace string) bool {
	if len(parts) == 0 || len(path) < len(parts) {
		return false
	}

	cur := path[len(path)-1]
	last := parts[len(parts)-1]
	space := last.Space
	if targetSpace != "" {
		space = targetSpace
	}
	if !nameMatches(cur, last.Local, space) {
		return false
	}

	ei := len(path) - 2
	for pi := len(parts) - 2; pi >= 0; pi-- {
		found := false
		for ei >= 0 {
			if nameMatches(path[ei], parts[pi].Local, parts[pi].Space) {
				found = true
				ei--
				break
			}
			ei--
		}
		if !found {
			return false
		}
	}
	return true
}

func findMask(path []xml.Name, masks []Mask) *Mask {
	var best *Mask
	bestLen := 0
	for i := range masks {
		parts := parseMaskPath(masks[i].Name)
		if !pathMatches(path, parts, masks[i].Space) {
			continue
		}
		if best == nil || len(parts) > bestLen {
			best = &masks[i]
			bestLen = len(parts)
		}
	}
	return best
}

func applyMask(elm xml.CharData, mask *Mask) xml.CharData {
	if mask == nil {
		return elm
	}

	runes := bytesToRunes(elm)

	switch mask.Mask {
	case ShowLastFour:
		if len(runes) < 5 {
			return xml.CharData(stars(len(runes)))
		}
		return xml.CharData(append(stars(len(runes)-4), string(runes[len(runes)-4:])...))

	case ShowMiddle:
		if len(runes) < 2 {
			return xml.CharData(stars(len(runes)))
		}
		quarter := (len(runes) / 4) + 1
		if len(runes) == 4 {
			quarter = 1
		}
		var out []byte
		out = append(out, stars(quarter)...)
		out = append(out, string(runes[quarter:len(runes)-quarter])...)
		out = append(out, stars(quarter)...)
		return xml.CharData(out)

	case ShowWordStart:
		return xml.CharData(maskWordStart(runes))

	case ShowNone:
		return xml.CharData(stars(len(runes)))
	}
	return elm
}

func bytesToRunes(b []byte) []rune {
	if !utf8.Valid(b) {
		// Replace invalid sequences so masking cannot emit illegal UTF-8.
		return []rune(strings.ToValidUTF8(string(b), "\uFFFD"))
	}
	return []rune(string(b))
}

func stars(n int) []byte {
	if n <= 0 {
		return nil
	}
	return bytes.Repeat([]byte("*"), n)
}

func maskWordStart(runes []rune) []byte {
	if len(runes) == 0 {
		return nil
	}
	out := make([]byte, 0, len(runes))
	inWord := false
	for _, r := range runes {
		if unicode.IsSpace(r) {
			out = utf8.AppendRune(out, r)
			inWord = false
			continue
		}
		if !inWord {
			out = utf8.AppendRune(out, r)
			inWord = true
			continue
		}
		out = append(out, '*')
	}
	return out
}
