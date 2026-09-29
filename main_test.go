package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestBarkConfidence(t *testing.T) {
	iterations := 100000

	fmt.Printf("Commencing bulk test of %d barks...\n", iterations)

	for i := 0; i < iterations; i++ {
		reqGen := httptest.NewRequest(http.MethodGet, "/bark", nil)
		recGen := httptest.NewRecorder()
		generateBark(recGen, reqGen)

		var genResult map[string]string
		json.NewDecoder(recGen.Body).Decode(&genResult)
		generatedBark := genResult["bark"]

		reqDetect := httptest.NewRequest(http.MethodGet, "/isbark?text="+generatedBark, nil)
		recDetect := httptest.NewRecorder()
		detectBark(recDetect, reqDetect)

		var detectResult map[string]interface{}
		json.NewDecoder(recDetect.Body).Decode(&detectResult)

		isBark := detectResult["is_bark"].(bool)
		if !isBark {
			t.Fatalf("Failed on iteration %d: '%s' was NOT classified as a bark!", i, generatedBark)
		}

		percStr := detectResult["bark_percentage"].(string)
		percStr = strings.TrimSuffix(percStr, "%")
		confidence, err := strconv.ParseFloat(percStr, 64)

		if err != nil {
			t.Fatalf("Failed to parse percentage: %v", err)
		}

		if confidence < 80.0 {
			t.Fatalf("Low confidence on iteration %d: '%s' scored only %.1f%%", i, generatedBark, confidence)
		}
	}
}
