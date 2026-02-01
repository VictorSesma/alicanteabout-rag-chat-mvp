package chat

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	reNonLetters = regexp.MustCompile(`[^\p{L}]+`)
	enStopwords  = map[string]struct{}{
		"the": {}, "and": {}, "of": {}, "to": {}, "in": {}, "is": {}, "for": {}, "on": {}, "with": {}, "from": {},
		"how": {}, "what": {}, "where": {}, "when": {}, "can": {}, "do": {}, "does": {}, "a": {}, "an": {}, "please": {},
	}
	esStopwords = map[string]struct{}{
		"hola": {}, "gracias": {}, "por": {}, "favor": {}, "como": {}, "que": {}, "donde": {}, "cuando": {}, "para": {},
		"con": {}, "sin": {}, "del": {}, "la": {}, "el": {}, "los": {}, "las": {}, "un": {}, "una": {}, "unos": {},
		"unas": {}, "y": {}, "o": {}, "pero": {}, "porque": {}, "quien": {}, "quienes": {}, "cual": {}, "cuanto": {},
		"cuantos": {}, "cuanta": {}, "cuantas": {}, "al": {}, "de": {}, "en": {},
	}
	frStopwords = map[string]struct{}{
		"bonjour": {}, "merci": {}, "svp": {}, "comment": {}, "quand": {}, "pour": {}, "avec": {}, "sans": {},
		"du": {}, "de": {}, "la": {}, "le": {}, "les": {}, "un": {}, "une": {}, "des": {}, "et": {}, "ou": {},
		"mais": {}, "parce": {}, "qui": {}, "quoi": {}, "quel": {}, "quelle": {}, "quels": {}, "quelles": {},
		"a": {}, "au": {}, "aux": {}, "en": {}, "aller": {}, "plage": {},
	}
)

func isEnglishQuestion(question string) bool {
	text := strings.ToLower(strings.TrimSpace(question))
	if text == "" {
		return true
	}
	clean := reNonLetters.ReplaceAllString(text, " ")
	tokens := strings.Fields(clean)
	if len(tokens) <= 2 {
		return true
	}

	englishHits := 0
	spanishHits := 0
	frenchHits := 0
	for _, tok := range tokens {
		if _, ok := enStopwords[tok]; ok {
			englishHits++
		}
		if _, ok := esStopwords[tok]; ok {
			spanishHits++
		}
		if _, ok := frStopwords[tok]; ok {
			frenchHits++
		}
	}

	nonASCII := 0
	for _, r := range text {
		if r > 127 && unicode.IsLetter(r) {
			nonASCII++
		}
	}

	if englishHits >= 2 {
		return true
	}
	if spanishHits >= 2 && spanishHits > englishHits && spanishHits >= frenchHits {
		return false
	}
	if frenchHits >= 2 && frenchHits > englishHits && frenchHits >= spanishHits {
		return false
	}
	if (spanishHits > 0 || frenchHits > 0) && englishHits == 0 {
		return false
	}
	if nonASCII > 0 && englishHits == 0 {
		return false
	}
	return true
}
