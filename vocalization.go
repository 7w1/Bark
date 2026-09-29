package main

import (
	"math/rand"
	"net/http"
	"strings"
	"unicode"
)

type vocalizationFamily string

const (
	familyBark vocalizationFamily = "bark"
	familyWoof  vocalizationFamily = "woof"
	familyWaf vocalizationFamily = "waf"
	familyGrowl vocalizationFamily = "growl"

	strictVocalizationThreshold = 75.0
	barkLikeThreshold           = 55.0
	maxVocalizationInputLetters = 512
)

type vocalizationForm struct {
	text      string
	family    vocalizationFamily
	exactOnly bool
	anchor    string
	embeddedOK bool
	embeddedOnly bool
}

var vocalizationForms = []vocalizationForm{
	{family: familyBark, text: "bark", embeddedOK: true},
	{family: familyBark, text: "barks", exactOnly: true, embeddedOK: true, embeddedOnly: true},
	{family: familyBark, text: "barking", exactOnly: true, embeddedOK: true, embeddedOnly: true},
	{family: familyBark, text: "barkbark", exactOnly: true, embeddedOK: true},
	{family: familyBark, text: "bowow", exactOnly: true, embeddedOK: true},
	{family: familyWoof, text: "wof", embeddedOK: true},
	{family: familyWoof, text: "wofs", exactOnly: true, embeddedOK: true, embeddedOnly: true},
	{family: familyWoof, text: "wuf", embeddedOK: true},
	{family: familyWoof, text: "ruf", embeddedOK: true},
	{family: familyWaf, text: "waf", embeddedOK: true},
	{family: familyWaf, text: "arf", embeddedOK: true},
	{family: familyWaf, text: "yap", embeddedOK: true},
	{family: familyWaf, text: "yip", embeddedOK: true},
	{family: familyGrowl, text: "grr", embeddedOK: true},
	{family: familyGrowl, text: "growl", embeddedOK: true},
	{family: familyGrowl, text: "growling", exactOnly: true, embeddedOK: true, embeddedOnly: true},
}

var vocalizationFormsByFamily = indexVocalizationForms()

func indexVocalizationForms() map[vocalizationFamily][]vocalizationForm {
	indexed := make(map[vocalizationFamily][]vocalizationForm)
	for _, form := range vocalizationForms {
		if !form.embeddedOnly {
			indexed[form.family] = append(indexed[form.family], form)
		}
	}
	return indexed
}

type generationFamily struct {
	family        vocalizationFamily
	variants      []string
	weight        int
	stretchChars  string
	stretchChance float64
	maxExtra      int
}

var generationFamilies = []generationFamily{
	{
		family:        familyBark,
		variants:      []string{"bark", "bowwow"},
		weight:        2,
		stretchChars:  "ao",
		stretchChance: 0.65,
		maxExtra:      4,
	},
	{
		family:        familyWoof,
		variants:      []string{"woof", "wuff", "ruff"},
		weight:        6,
		stretchChars:  "ou",
		stretchChance: 0.65,
		maxExtra:      4,
	},
	{
		family:        familyWaf,
		variants:      []string{"waf", "arf", "yap"},
		weight:        2,
		stretchChars:  "a",
		stretchChance: 0.55,
		maxExtra:      3,
	},
	{
		family:        familyGrowl,
		variants:      []string{"grr", "growl"},
		weight:        1,
		stretchChars:  "r",
		stretchChance: 0.55,
		maxExtra:      4,
	},
}

func stretchVocalization(text string, allowed string, chance float64, maxExtra int) string {
	var builder strings.Builder
	builder.Grow(len(text) + maxExtra*len(text))

	for _, char := range text {
		builder.WriteRune(char)
		if maxExtra > 0 && strings.ContainsRune(allowed, char) && rand.Float64() < chance {
			extra := rand.Intn(maxExtra) + 1
			builder.WriteString(strings.Repeat(string(char), extra))
		}
	}

	return builder.String()
}

func generateVocalization() (string, vocalizationFamily) {
	totalWeight := 0
	for _, family := range generationFamilies {
		totalWeight += family.weight
	}

	roll := rand.Intn(totalWeight)
	for _, family := range generationFamilies {
		if roll < family.weight {
			variant := family.variants[rand.Intn(len(family.variants))]
			return stretchVocalization(
				variant,
				family.stretchChars,
				family.stretchChance,
				family.maxExtra,
			), family.family
		}
		roll -= family.weight
	}

	return "bark", familyBark
}

func generateBark(w http.ResponseWriter, r *http.Request) {
	serveGenerateBark(w, r)
}

type normalizedVocalization struct {
	letters      string
	squeezed     string
	unknownCount int
}

func normalizeVocalization(text string) normalizedVocalization {
	var letters strings.Builder
	var squeezed strings.Builder
	letters.Grow(len(text))
	squeezed.Grow(len(text))

	var lastChar byte
	runLength := 0
	unknownCount := 0

	for _, char := range strings.ToLower(text) {
		if char >= 'a' && char <= 'z' {
			letter := byte(char)
			letters.WriteByte(letter)

			if letter != lastChar {
				squeezed.WriteByte(letter)
				lastChar = letter
				runLength = 1
				continue
			}

			if letter == 'r' && runLength < 2 {
				squeezed.WriteByte(letter)
			}
			runLength++
			continue
		}

		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			unknownCount++
		}
	}

	return normalizedVocalization{
		letters:      letters.String(),
		squeezed:     squeezed.String(),
		unknownCount: unknownCount,
	}
}

func squeezeLetters(text string) string {
	return normalizeVocalization(text).squeezed
}

func countLetters(text string) int {
	return len(normalizeVocalization(text).letters)
}

type segmentationState struct {
	valid         bool
	distance      int
	segments      int
	exactSegments int
}

type familyMatch struct {
	family        vocalizationFamily
	matched       bool
	distance      int
	segments      int
	exactSegments int
}

func betterSegmentation(candidate, current segmentationState) bool {
	if !current.valid {
		return true
	}
	if candidate.distance != current.distance {
		return candidate.distance < current.distance
	}
	if candidate.exactSegments != current.exactSegments {
		return candidate.exactSegments > current.exactSegments
	}
	return candidate.segments < current.segments
}

func levenshteinDistance(left, right string) int {
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}

	for leftIndex := 1; leftIndex <= len(left); leftIndex++ {
		current[0] = leftIndex
		for rightIndex := 1; rightIndex <= len(right); rightIndex++ {
			cost := 0
			if left[leftIndex-1] != right[rightIndex-1] {
				cost = 1
			}

			deletion := previous[rightIndex] + 1
			insertion := current[rightIndex-1] + 1
			substitution := previous[rightIndex-1] + cost
			current[rightIndex] = minInt(deletion, insertion, substitution)
		}
		previous, current = current, previous
	}

	return previous[len(right)]
}

func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func formsForFamily(family vocalizationFamily) []vocalizationForm {
	return vocalizationFormsByFamily[family]
}

func bestFamilyMatch(squeezed string, family vocalizationFamily, maxEditPerSegment int) familyMatch {
	result := familyMatch{family: family}
	if squeezed == "" {
		return result
	}

	states := make([]segmentationState, len(squeezed)+1)
	states[0] = segmentationState{valid: true}

	for position := 0; position < len(squeezed); position++ {
		if !states[position].valid {
			continue
		}

		for _, form := range formsForFamily(family) {
			editLimit := maxEditPerSegment
			if form.exactOnly {
				editLimit = 0
			}

			minLength := len(form.text) - editLimit
			if minLength < 1 {
				minLength = 1
			}
			maxLength := len(form.text) + editLimit

			for length := minLength; length <= maxLength && position+length <= len(squeezed); length++ {
				candidateText := squeezed[position : position+length]
				distance := levenshteinDistance(candidateText, form.text)
				if distance > editLimit {
					continue
				}

				candidate := segmentationState{
					valid:         true,
					distance:      states[position].distance + distance,
					segments:      states[position].segments + 1,
					exactSegments: states[position].exactSegments,
				}
				if distance == 0 {
					candidate.exactSegments++
				}

				nextPosition := position + length
				if betterSegmentation(candidate, states[nextPosition]) {
					states[nextPosition] = candidate
				}
			}
		}
	}

	finalState := states[len(squeezed)]
	if !finalState.valid {
		return result
	}

	result.matched = true
	result.distance = finalState.distance
	result.segments = finalState.segments
	result.exactSegments = finalState.exactSegments
	return result
}

func familyAnchorOK(squeezed string, match familyMatch) bool {
	if !match.matched || len(squeezed) < 3 {
		return false
	}

	switch match.family {
	case familyBark:
		return strings.HasPrefix(squeezed, "b")
	case familyWoof:
		return strings.HasPrefix(squeezed, "w") || strings.HasPrefix(squeezed, "r")
	case familyWaf:
		return strings.HasPrefix(squeezed, "w") || strings.HasPrefix(squeezed, "a") || strings.HasPrefix(squeezed, "y")
	case familyGrowl:
		return strings.HasPrefix(squeezed, "g")
	default:
		return false
	}
}

func scoreFamilyMatch(match familyMatch, normalized normalizedVocalization) float64 {
	if !familyAnchorOK(normalized.squeezed, match) {
		return 0
	}

	score := 100.0
	if len(normalized.squeezed) > 0 {
		score -= 100.0 * float64(match.distance) / float64(len(normalized.squeezed))
	}
	score -= 35.0 * float64(normalized.unknownCount)

	if len(normalized.letters) > maxVocalizationInputLetters {
		score -= 40.0
	}

	if match.segments == 1 && len(normalized.squeezed) < 4 && match.distance > 0 {
		score -= 15.0
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func allowedStrictErrors(segments int) int {
	allowed := segments / 4
	if allowed < 1 {
		return 1
	}
	return allowed
}

func allowedFuzzyErrors(segments int) int {
	allowed := segments / 2
	if allowed < 1 {
		return 1
	}
	return allowed
}

type vocalizationAnalysis struct {
	normalized  normalizedVocalization
	family      vocalizationFamily
	matchType   string
	strictScore float64
	fuzzyScore  float64
	isStrict    bool
	isFuzzy     bool
	isEmbedded  bool
	matchedText string
	coverage    float64
}

func analyzeVocalization(text string) vocalizationAnalysis {
	normalized := normalizeVocalization(text)
	analysis := vocalizationAnalysis{
		normalized: normalized,
		family:     "unknown",
		matchType:  "none",
	}
	for _, form := range vocalizationForms {
		if form.embeddedOnly && normalized.squeezed == form.text {
			return analysis
		}
	}
	if isBarkEcho(normalized.squeezed) {
		score := 100 - 35*float64(normalized.unknownCount)
		if len(normalized.letters) > maxVocalizationInputLetters {
			score -= 40
		}
		analysis.family = familyBark
		analysis.matchType = "stretched"
		analysis.strictScore = max(0, score)
		analysis.fuzzyScore = analysis.strictScore
		analysis.isStrict = normalized.unknownCount == 0 && len(normalized.letters) <= maxVocalizationInputLetters && score >= strictVocalizationThreshold
		analysis.isFuzzy = normalized.unknownCount <= 2 && score >= barkLikeThreshold
		return analysis
	}

	var bestMatch familyMatch
	bestScore := 0.0
	for _, family := range []vocalizationFamily{familyBark, familyWoof, familyWaf, familyGrowl} {
		match := bestFamilyMatch(normalized.squeezed, family, 1)
		score := scoreFamilyMatch(match, normalized)
		if score > bestScore {
			bestScore = score
			bestMatch = match
			analysis.family = match.family
		}

		if score >= analysis.strictScore {
			analysis.strictScore = score
		}
		if score >= analysis.fuzzyScore {
			analysis.fuzzyScore = score
		}

		strictEligible := match.matched &&
			familyAnchorOK(normalized.squeezed, match) &&
			normalized.unknownCount == 0 &&
			len(normalized.letters) <= maxVocalizationInputLetters &&
			match.distance <= allowedStrictErrors(match.segments)
		fuzzyEligible := match.matched &&
			familyAnchorOK(normalized.squeezed, match) &&
			normalized.unknownCount <= 2 &&
			match.distance <= allowedFuzzyErrors(match.segments)

		if strictEligible && score >= strictVocalizationThreshold {
			analysis.isStrict = true
		}
		if fuzzyEligible && score >= barkLikeThreshold {
			analysis.isFuzzy = true
		}
	}

	if bestScore > 0 {
		if bestMatch.distance > 0 || normalized.unknownCount > 0 {
			analysis.matchType = "typo"
		} else if normalized.letters != normalized.squeezed {
			analysis.matchType = "stretched"
		} else {
			analysis.matchType = "exact"
		}
	}
	if !analysis.isStrict && !analysis.isFuzzy && normalized.unknownCount == 0 && isMixedBark(normalized.squeezed) {
		analysis.family = familyBark
		analysis.matchType = "mixed"
		analysis.fuzzyScore = 85
		analysis.isFuzzy = true
	}

	return analysis
}

func isMixedBark(squeezed string) bool {
	for _, growl := range []string{"grr", "ruf", "arf", "waf"} {
		if strings.HasPrefix(squeezed, growl) && isLooseBarkSyllable(strings.TrimPrefix(squeezed, growl)) ||
			strings.HasSuffix(squeezed, growl) && isLooseBarkSyllable(strings.TrimSuffix(squeezed, growl)) {
			return true
		}
	}
	return false
}

func isLooseBarkSyllable(syllable string) bool {
	return syllable == "bark" || syllable == "wof" || syllable == "bowow"
}

func isBarkEcho(squeezed string) bool {
	if !strings.HasPrefix(squeezed, "bowow") {
		return false
	}
	tail := strings.TrimPrefix(squeezed, "bowow")
	if len(tail) < 2 || len(tail)%2 != 0 {
		return false
	}
	for len(tail) > 0 {
		if !strings.HasPrefix(tail, "ow") {
			return false
		}
		tail = tail[2:]
	}
	return true
}

func detectBark(w http.ResponseWriter, r *http.Request) {
	serveDetectBark(w, r)
}

func detectBarkLike(w http.ResponseWriter, r *http.Request) {
	serveDetectBarkLike(w, r)
}
