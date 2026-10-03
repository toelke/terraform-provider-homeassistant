package client

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mozillazg/go-unidecode"
	"golang.org/x/text/unicode/norm"
)

var (
	slugQuotes     = regexp.MustCompile(`'+`)
	slugDecimalRef = regexp.MustCompile(`&#([0-9]+);`)
	slugHexRef     = regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	slugEntityRef  = regexp.MustCompile(`&([A-Za-z0-9]+);`)
	slugDisallowed = regexp.MustCompile(`[^-a-zA-Z0-9]+`)
	slugDashes     = regexp.MustCompile(`-{2,}`)
)

// Slugify returns the ID Home Assistant derives from name, mirroring HA's `util.slugify`, which
// wraps python-slugify with `separator="_"`.
func Slugify(text string) string {
	if text == "" {
		return ""
	}
	// The steps and their order follow python-slugify 8.
	s := slugQuotes.ReplaceAllString(text, "-")
	s = transliterate(norm.NFKD.String(s))
	s = slugEntityRef.ReplaceAllStringFunc(s, func(m string) string {
		if r, ok := htmlEntities[m[1:len(m)-1]]; ok {
			return string(r)
		}
		return m
	})
	s = replaceCharRefs(s, slugDecimalRef, 10)
	s = replaceCharRefs(s, slugHexRef, 16)
	s = norm.NFKD.String(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "'", "")
	s = removeDigitCommas(s)
	s = slugDisallowed.ReplaceAllString(s, "-")
	s = strings.Trim(slugDashes.ReplaceAllString(s, "-"), "-")
	s = strings.ReplaceAll(s, "-", "_")
	if s == "" {
		return "unknown"
	}
	return s
}

// translitOverride gives the transliteration of the code points lo through hi.
type translitOverride struct {
	lo, hi rune
	s      string
}

// transliterate converts s to ASCII the way text-unidecode, which HA's python-slugify uses, does.
// It is go-unidecode, corrected by the generated translitOverrides.
func transliterate(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		case r > translitMax:
			// text-unidecode drops code points outside its table.
		default:
			i := sort.Search(len(translitOverrides), func(i int) bool { return translitOverrides[i].hi >= r })
			if i < len(translitOverrides) && translitOverrides[i].lo <= r {
				b.WriteString(translitOverrides[i].s)
			} else {
				b.WriteString(unidecode.Unidecode(string(r)))
			}
		}
	}
	return b.String()
}

// replaceCharRefs decodes numeric character references. Like python-slugify, it leaves s
// unchanged if any reference is not a valid code point.
func replaceCharRefs(s string, re *regexp.Regexp, base int) string {
	valid := true
	out := re.ReplaceAllStringFunc(s, func(m string) string {
		digits := re.FindStringSubmatch(m)[1]
		cp, err := strconv.ParseUint(digits, base, 32)
		if err != nil || cp > unicode.MaxRune {
			valid = false
			return m
		}
		return string(rune(cp))
	})
	if !valid {
		return s
	}
	return out
}

// removeDigitCommas drops every comma between two digits, as in "1,000".
func removeDigitCommas(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if r == ',' && i > 0 && i < len(runes)-1 && unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[i+1]) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
