package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestLanguagePacksRoundTripGeneratedSounds(t *testing.T) {
	for _, tag := range supportedLanguageTags {
		pack := languagePacks[strings.ToLower(tag)]
		for i := 0; i < 1000; i++ {
			generated, family := generateForLanguage(pack)
			analysis := analyzeForLanguage(generated, pack)
			if !analysis.isStrict || analysis.family != family {
				t.Fatalf("%s generated %q did not strictly round-trip as %q: %#v", tag, generated, family, analysis)
			}
		}
	}
}

func TestLanguageWholeInputRecognition(t *testing.T) {
	tests := []struct {
		name     string
		language string
		input    string
		strict   bool
		fuzzy    bool
	}{
		{name: "English bark", language: "en", input: "bark", strict: true, fuzzy: true},
		{name: "English woof", language: "en", input: "woof", strict: true, fuzzy: true},
		{name: "English stretched woof", language: "en", input: "woooofff", strict: true, fuzzy: true},
		{name: "English punctuation", language: "en", input: "w.o.o.f", strict: true, fuzzy: true},
		{name: "French ouaf", language: "fr", input: "ouaf", strict: true, fuzzy: true},
		{name: "French wouf", language: "fr", input: "wouf", strict: true, fuzzy: true},
		{name: "French growl", language: "fr", input: "grrr", strict: true, fuzzy: true},
		{name: "Japanese hiragana", language: "ja", input: "わん", strict: true, fuzzy: true},
		{name: "Japanese repeated hiragana", language: "ja", input: "わんわん", strict: true, fuzzy: true},
		{name: "Japanese katakana", language: "ja", input: "ワンワン", strict: true, fuzzy: true},
		{name: "Japanese half-width", language: "ja", input: "ﾜﾝﾜﾝ", strict: true, fuzzy: true},
		{name: "Japanese growl", language: "ja", input: "ガルル", strict: true, fuzzy: true},
		{name: "Turkish hav", language: "tr", input: "hav", strict: true, fuzzy: true},
		{name: "Turkish hev", language: "tr", input: "hev-hev", strict: true, fuzzy: true},
		{name: "Turkish growl", language: "tr", input: "hırr", strict: true, fuzzy: true},
		{name: "Russian gav", language: "ru", input: "гав", strict: true, fuzzy: true},
		{name: "Russian tyav", language: "ru", input: "тяв-тяв", strict: true, fuzzy: true},
		{name: "Russian growl", language: "ru", input: "ррр", strict: true, fuzzy: true},
		{name: "Ukrainian gav", language: "uk", input: "гав-гав", strict: true, fuzzy: true},
		{name: "Ukrainian dzyav", language: "uk", input: "дзяв", strict: true, fuzzy: true},
		{name: "Ukrainian growl", language: "uk", input: "гар", strict: true, fuzzy: true},
		{name: "Simplified Chinese short form", language: "zh-hans", input: "汪", strict: true, fuzzy: true},
		{name: "Traditional Chinese repeated form", language: "zh-hant", input: "汪汪", strict: true, fuzzy: true},
		{name: "Chinese alternate bark", language: "zh-hans", input: "狺狺", strict: true, fuzzy: true},
		{name: "Spanish guau", language: "es", input: "guau", strict: true, fuzzy: true},
		{name: "Spanish repeated guau", language: "es", input: "guau-guau", strict: true, fuzzy: true},
		{name: "Spanish growl", language: "es", input: "grrr", strict: true, fuzzy: true},
		{name: "Dutch woef", language: "nl", input: "woef", strict: true, fuzzy: true},
		{name: "Dutch waf", language: "nl", input: "waf", strict: true, fuzzy: true},
		{name: "Dutch blaf", language: "nl", input: "blaf", strict: true, fuzzy: true},
		{name: "Dutch growl", language: "nl", input: "grr", strict: true, fuzzy: true},
		{name: "German wuff", language: "de", input: "wuff", strict: true, fuzzy: true},
		{name: "German wau", language: "de", input: "wau", strict: true, fuzzy: true},
		{name: "German growl", language: "de", input: "knurr", strict: true, fuzzy: true},
		{name: "Armenian hav", language: "hy", input: "հավ", strict: true, fuzzy: true},
		{name: "Armenian haf", language: "hy", input: "հաֆ", strict: true, fuzzy: true},
		{name: "Esperanto boj", language: "eo", input: "boj", strict: true, fuzzy: true},
		{name: "Esperanto ŭaŭ", language: "eo", input: "ŭaŭ", strict: true, fuzzy: true},
		{name: "Esperanto growl", language: "eo", input: "grrr", strict: true, fuzzy: true},
		{name: "Arabic aw", language: "ar", input: "عو", strict: true, fuzzy: true},
		{name: "Arabic haw", language: "ar", input: "هو-هو", strict: true, fuzzy: true},
		{name: "Arabic alternate haw", language: "ar", input: "هاو-هاو", strict: true, fuzzy: true},
		{name: "Hindi bhaun", language: "hi", input: "भौं", strict: true, fuzzy: true},
		{name: "Hindi bhon", language: "hi", input: "भों", strict: true, fuzzy: true},
		{name: "Hindi growl", language: "hi", input: "गुर्र", strict: true, fuzzy: true},
		{name: "other pack does not see English", language: "fr", input: "bark", strict: false, fuzzy: false},
		{name: "digits are not part of a new-pack sound", language: "fr", input: "ouaf1", strict: false, fuzzy: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis := analyzeForLanguage(test.input, languagePacks[strings.ToLower(test.language)])
			if analysis.isStrict != test.strict || analysis.isFuzzy != test.fuzzy {
				t.Fatalf("unexpected verdict for %q in %s: strict=%t fuzzy=%t scores=%.1f/%.1f", test.input, test.language, analysis.isStrict, analysis.isFuzzy, analysis.strictScore, analysis.fuzzyScore)
			}
		})
	}
}

func TestResearchBackedVariantsStayInTheirSoundFamily(t *testing.T) {
	tests := []struct {
		language string
		input    string
		family   vocalizationFamily
	}{
		{language: "tr", input: "hav-hav", family: familyWaf},
		{language: "tr", input: "hev-hev", family: familyWaf},
		{language: "tr", input: "hırr", family: familyGrowl},
		{language: "ru", input: "гав-гав", family: familyBark},
		{language: "ru", input: "тяв-тяв", family: familyWaf},
		{language: "ru", input: "ррр", family: familyGrowl},
		{language: "uk", input: "гав-гав", family: familyBark},
		{language: "uk", input: "дзяв-дзяв", family: familyWaf},
		{language: "uk", input: "гар", family: familyGrowl},
		{language: "zh-hans", input: "汪汪", family: familyBark},
		{language: "zh-hans", input: "狺狺", family: familyBark},
		{language: "es", input: "guau-guau", family: familyWoof},
		{language: "es", input: "grrr", family: familyGrowl},
		{language: "nl", input: "woef", family: familyWoof},
		{language: "nl", input: "waf", family: familyWaf},
		{language: "nl", input: "grr", family: familyGrowl},
		{language: "de", input: "wuff", family: familyWoof},
		{language: "de", input: "wau", family: familyWaf},
		{language: "de", input: "knurr", family: familyGrowl},
		{language: "hy", input: "հավ-հավ", family: familyWaf},
		{language: "hy", input: "հաֆ", family: familyWaf},
		{language: "eo", input: "boj-boj", family: familyBark},
		{language: "eo", input: "ŭaŭ", family: familyWoof},
		{language: "eo", input: "grrr", family: familyGrowl},
		{language: "ar", input: "عو-عو", family: familyWaf},
		{language: "ar", input: "هو", family: familyWaf},
		{language: "ar", input: "هاو", family: familyWaf},
		{language: "hi", input: "भौं-भौं", family: familyWoof},
		{language: "hi", input: "गुर्र", family: familyGrowl},
		{language: "fr", input: "grrr", family: familyGrowl},
		{language: "ja", input: "ガルル", family: familyGrowl},
	}
	for _, test := range tests {
		t.Run(test.language+"/"+test.input, func(t *testing.T) {
			analysis := analyzeForLanguage(test.input, languagePacks[test.language])
			if !analysis.isStrict || analysis.family != test.family {
				t.Fatalf("%q in %s classified as strict=%t family=%q, want %q", test.input, test.language, analysis.isStrict, analysis.family, test.family)
			}
		})
	}
}

func TestChineseHanSuffixAndCanonicalLanguageTags(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/isbark?text=%E6%B1%AA%E6%B1%AA&lang=ZH-hAnT", nil)
	recorder := httptest.NewRecorder()
	detectBark(recorder, request)
	response := decodeJSONResponse(t, recorder)
	if recorder.Code != http.StatusOK || response["language"] != "zh-Hant" || response["is_bark"] != true {
		t.Fatalf("Chinese canonical tag/form did not resolve: status=%d response=%#v", recorder.Code, response)
	}

	text := "喜欢狗汪汪"
	analysis := applyEmbeddedAnalysis(text, languagePacks["zh-hans"], analyzePackWhole(text, languagePacks["zh-hans"]))
	if analysis.isStrict || !analysis.isEmbedded || analysis.matchedText != "汪汪" {
		t.Fatalf("expected exact guarded Han suffix, got %#v", analysis)
	}

	for _, incidental := range []string{"你好汪", "汉汪字", "abc汪"} {
		analysis := applyEmbeddedAnalysis(incidental, languagePacks["zh-hans"], analyzePackWhole(incidental, languagePacks["zh-hans"]))
		if analysis.isFuzzy {
			t.Errorf("unqualified Chinese substring %q matched: %#v", incidental, analysis)
		}
	}
}

func TestEmbeddedBarkLikeCoverageAndBoundaries(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "You’re cute bark", want: true},
		{input: "You’re cute woooofff", want: true},
		{input: "That was grrbark", want: true},
		{input: "That was ruffwoof", want: true},
		{input: "You are cute bark", want: true},
		{input: "bark bark you", want: true},
		{input: "This is a long unrelated sentence with a single bark at the very end", want: false},
		{input: "barking", want: true},
		{input: "barks", want: true},
		{input: "woofs", want: true},
		{input: "somewhere", want: false},
		{input: "waffle", want: false},
		{input: "barkeeper", want: false},
		{input: "You're cute now", want: false},
		{input: "cute \x60bark\x60", want: false},
		{input: "cute https://example.com/bark", want: false},
		{input: "🙂 123 ...", want: false},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			whole := analyzeVocalization(test.input)
			analysis := applyEmbeddedAnalysis(test.input, languagePacks["en"], whole)
			if analysis.isFuzzy != test.want {
				t.Fatalf("input %q got fuzzy=%t, score=%.1f match=%q", test.input, analysis.isFuzzy, analysis.fuzzyScore, analysis.matchedText)
			}
		})
	}

	analysis := applyEmbeddedAnalysis("You’re cute bark", languagePacks["en"], analyzeVocalization("You’re cute bark"))
	if analysis.isStrict || !analysis.isEmbedded || analysis.matchedText != "bark" {
		t.Fatalf("unexpected embedded analysis: %#v", analysis)
	}
	if math.Abs(analysis.coverage-30.7692307692) > 0.1 || analysis.matchType != "embedded" {
		t.Fatalf("unexpected coverage or match type: coverage=%.3f type=%q", analysis.coverage, analysis.matchType)
	}

	japanese := applyEmbeddedAnalysis("かわいいわん", languagePacks["ja"], analyzePackWhole("かわいいわん", languagePacks["ja"]))
	if japanese.isStrict || !japanese.isEmbedded || japanese.matchedText != "わん" || math.Abs(japanese.coverage-100.0*2.0/6.0) > 0.1 {
		t.Fatalf("guarded Japanese suffix did not match as expected: %#v", japanese)
	}

	chineseText := "喜欢狗汪汪"
	chineseWhole := analyzePackWhole(chineseText, languagePacks["zh-hans"])
	chinese := applyEmbeddedAnalysis(chineseText, languagePacks["zh-hans"], chineseWhole)
	if chinese.isStrict || !chinese.isEmbedded || chinese.matchedText != "汪汪" || math.Abs(chinese.coverage-40) > 0.1 {
		t.Fatalf("guarded Chinese suffix did not match as expected: %#v", chinese)
	}
	chineseShort := applyEmbeddedAnalysis("你好汪", languagePacks["zh-hans"], analyzePackWhole("你好汪", languagePacks["zh-hans"]))
	if chineseShort.isFuzzy {
		t.Fatalf("single-character Chinese suffix should not match: %#v", chineseShort)
	}
}

func TestEnglishWordVariantsAreFuzzyOnly(t *testing.T) {
	for _, input := range []string{"barks", "barking", "woofs", "growling"} {
		strict := httptest.NewRecorder()
		fuzzy := httptest.NewRecorder()
		requestURL := "?text=" + url.QueryEscape(input)
		detectBark(strict, httptest.NewRequest(http.MethodGet, "/isbark"+requestURL, nil))
		detectBarkLike(fuzzy, httptest.NewRequest(http.MethodGet, "/barklike"+requestURL, nil))
		if decodeJSONResponse(t, strict)["is_bark"] != false {
			t.Errorf("%q passed strict detection", input)
		}
		response := decodeJSONResponse(t, fuzzy)
		if response["is_bark_like"] != true || response["matched_text"] != input {
			t.Errorf("%q was not recognized as a fuzzy word variant: %#v", input, response)
		}
	}
}

func TestMixedDogSoundsAreFuzzyOnly(t *testing.T) {
	for _, input := range []string{"grrbark", "grrwoof", "arfwoof", "woofgrr", "barkwaf"} {
		strict := httptest.NewRecorder()
		fuzzy := httptest.NewRecorder()
		requestURL := "?text=" + url.QueryEscape(input) + "&lang=auto"
		detectBark(strict, httptest.NewRequest(http.MethodGet, "/isbark"+requestURL, nil))
		detectBarkLike(fuzzy, httptest.NewRequest(http.MethodGet, "/barklike"+requestURL, nil))
		if decodeJSONResponse(t, strict)["is_bark"] != false {
			t.Errorf("%q passed strict detection", input)
		}
		response := decodeJSONResponse(t, fuzzy)
		if response["is_bark_like"] != true || response["language"] != "en" {
			t.Errorf("%q missed fuzzy English detection: %#v", input, response)
		}
	}
}

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func TestLanguageQueryValidationOnBarkEndpoints(t *testing.T) {
	handlers := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "bark", handler: generateBark},
		{name: "isbark", handler: detectBark},
		{name: "barklike", handler: detectBarkLike},
	}
	for _, endpoint := range handlers {
		for _, suffix := range []string{"?lang=xx", "?lang=", "?lang=fr&lang=ja"} {
			t.Run(endpoint.name+suffix, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				endpoint.handler(recorder, httptest.NewRequest(http.MethodGet, "/"+endpoint.name+suffix, nil))
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
				}
				response := decodeJSONResponse(t, recorder)
				if response["error"] != "unsupported language" {
					t.Fatalf("unexpected error response: %#v", response)
				}
			})
		}
	}
}

func TestAutomaticDetectionFindsLanguageAndKeepsAmbiguousFormsStable(t *testing.T) {
	tests := []struct {
		input    string
		language string
	}{
		{input: "bark", language: "en"},
		{input: "ouaf", language: "fr"},
		{input: "わん", language: "ja"},
		{input: "ワンワン", language: "ja"},
		{input: "hav", language: "tr"},
		{input: "тяв", language: "ru"},
		{input: "дзяв", language: "uk"},
		{input: "汪汪", language: "zh-Hans"},
		{input: "guau", language: "es"},
		{input: "blaf", language: "nl"},
		{input: "wuff", language: "de"},
		{input: "հաֆ", language: "hy"},
		{input: "boj", language: "eo"},
		{input: "عو", language: "ar"},
		{input: "भौं", language: "hi"},
	}
	for _, test := range tests {
		for _, endpoint := range []struct {
			path string
			fn   http.HandlerFunc
		}{
			{path: "/isbark", fn: detectBark},
			{path: "/barklike", fn: detectBarkLike},
		} {
			t.Run(endpoint.path+"/"+test.input, func(t *testing.T) {
				requestURL := endpoint.path + "?text=" + url.QueryEscape(test.input) + "&lang=auto"
				recorder := httptest.NewRecorder()
				endpoint.fn(recorder, httptest.NewRequest(http.MethodGet, requestURL, nil))
				response := decodeJSONResponse(t, recorder)
				if recorder.Code != http.StatusOK || response["language"] != test.language {
					t.Fatalf("automatic detection returned status=%d language=%v for %q", recorder.Code, response["language"], test.input)
				}
				if endpoint.path == "/isbark" && response["is_bark"] != true {
					t.Fatalf("strict auto detection missed %q: %#v", test.input, response)
				}
				if endpoint.path == "/barklike" && response["is_bark_like"] != true {
					t.Fatalf("fuzzy auto detection missed %q: %#v", test.input, response)
				}
			})
		}
	}

	recorder := httptest.NewRecorder()
	generateBark(recorder, httptest.NewRequest(http.MethodGet, "/bark?lang=auto", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("/bark accepted detection-only lang=auto: %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	detectBarkLike(recorder, httptest.NewRequest(http.MethodGet, "/barklike?text=ordinary+words&lang=auto", nil))
	response := decodeJSONResponse(t, recorder)
	if recorder.Code != http.StatusOK || response["language"] != "auto" || response["is_bark_like"] != false {
		t.Fatalf("unexpected no-match auto response: status=%d response=%#v", recorder.Code, response)
	}
}

func TestDetectionHTTPFieldsAndLimits(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/barklike?text=You%E2%80%99re+cute+bark", nil)
	recorder := httptest.NewRecorder()
	detectBarkLike(recorder, request)
	response := decodeJSONResponse(t, recorder)
	if recorder.Code != http.StatusOK || response["language"] != "en" || response["is_bark_like"] != true {
		t.Fatalf("unexpected embedded HTTP response: status=%d response=%#v", recorder.Code, response)
	}
	if response["matched_text"] != "bark" || response["match_type"] != "embedded" {
		t.Fatalf("missing embedded diagnostics: %#v", response)
	}
	if math.Abs(response["coverage_percentage"].(float64)-30.7692307692) > 0.1 {
		t.Fatalf("unexpected HTTP coverage: %#v", response["coverage_percentage"])
	}

	for _, target := range []struct {
		path string
		fn   http.HandlerFunc
	}{
		{path: "/isbark", fn: detectBark},
		{path: "/barklike", fn: detectBarkLike},
	} {
		recorder := httptest.NewRecorder()
		target.fn(recorder, httptest.NewRequest(http.MethodGet, target.path+"?text="+strings.Repeat("a", 513), nil))
		if recorder.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("%s accepted oversized text: status=%d", target.path, recorder.Code)
		}
	}

	for _, query := range []string{"?text=", "?text=%20", "?text=%FF", "?text=" + strings.Repeat("!", maxRequestTextBytes+1)} {
		recorder := httptest.NewRecorder()
		detectBarkLike(recorder, httptest.NewRequest(http.MethodGet, "/barklike"+query, nil))
		if recorder.Code == http.StatusOK {
			t.Fatalf("invalid or oversized query was accepted: %q", query[:minInt(len(query), 40)])
		}
	}

	recorder = httptest.NewRecorder()
	detectBark(recorder, httptest.NewRequest(http.MethodPost, "/isbark?text=bark", nil))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("unexpected method handling: status=%d allow=%q", recorder.Code, recorder.Header().Get("Allow"))
	}
}

func TestRejectedCandidateKeepsClosestClassification(t *testing.T) {
	for _, endpoint := range []struct {
		path string
		fn http.HandlerFunc
		verdict string
	}{
		{path: "/isbark", fn: detectBark, verdict: "is_bark"},
		{path: "/barklike", fn: detectBarkLike, verdict: "is_bark_like"},
	} {
		recorder := httptest.NewRecorder()
		endpoint.fn(recorder, httptest.NewRequest(http.MethodGet, endpoint.path+"?text=b.a.r.k.12&lang=auto", nil))
		response := decodeJSONResponse(t, recorder)
		if response[endpoint.verdict] != false || response["language"] != "auto" ||
			response["closest_language"] != "en" || response["family"] != "bark" ||
			response["match_type"] != "typo" || response["bark_percentage"] != "30.0%" {
			t.Errorf("%s lost its closest candidate: %#v", endpoint.path, response)
		}
	}
}

func TestAskBarkRemainsEnglishOnly(t *testing.T) {
	recorder := httptest.NewRecorder()
	askBark(recorder, httptest.NewRequest(http.MethodGet, "/askbark?text=Is+it+time+for+food", nil))
	response := decodeJSONResponse(t, recorder)
	if response["answer"] != "You may rely on woof." {
		t.Fatalf("English 8-ball fixture changed: got %q", response["answer"])
	}
	if _, ok := response["language"]; ok {
		t.Fatal("8-ball response advertises a language")
	}
	for _, tag := range []string{"en", "fr", "auto"} {
		recorder = httptest.NewRecorder()
		askBark(recorder, httptest.NewRequest(http.MethodGet, "/askbark?text=hello&lang="+tag, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("8-ball accepted lang=%s: %d", tag, recorder.Code)
		}
	}
}

func TestConcurrentLanguageDetection(t *testing.T) {
	const workers = 24
	var group sync.WaitGroup
	errors := make(chan string, workers)
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			analysis := applyEmbeddedAnalysis("You’re cute bark", languagePacks["en"], analyzeVocalization("You’re cute bark"))
			if !analysis.isEmbedded || analysis.matchedText != "bark" {
				errors <- "concurrent English detection returned a different result"
			}
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
