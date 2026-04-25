package openai

import "net/http"

func shouldWriteUpstreamEmptyOutputError(text string) bool {
	return text == ""
}

func upstreamEmptyOutputDetail(contentFilter bool, text, thinking string) (int, string, string) {
	_ = text
	if contentFilter {
		return http.StatusBadRequest, "Upstream content filtered the response and returned no output.", "content_filter"
	}
	if thinking != "" {
		return http.StatusTooManyRequests, "Upstream account hit a rate limit and returned reasoning without visible output.", "upstream_empty_output"
	}
	return http.StatusTooManyRequests, "Upstream account hit a rate limit and returned empty output.", "upstream_empty_output"
}

func writeUpstreamEmptyOutputError(w http.ResponseWriter, text, thinking string, contentFilter bool) bool {
	if !shouldWriteUpstreamEmptyOutputError(text) {
		return false
	}
	status, message, code := upstreamEmptyOutputDetail(contentFilter, text, thinking)
	writeOpenAIErrorWithCode(w, status, message, code)
	return true
}
