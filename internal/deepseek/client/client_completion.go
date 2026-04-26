package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsprotocol "ds2api/internal/deepseek/protocol"
	trans "ds2api/internal/deepseek/transport"
)

func (c *Client) CallCompletion(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, maxAttempts int) (*http.Response, error) {
	if maxAttempts <= 0 {
		maxAttempts = c.maxRetries
	}
	clients := c.requestClientsForAuth(ctx, a)
	headers := c.authHeaders(a.DeepSeekToken)
	headers["x-ds-pow-response"] = powResp
	captureSession := c.capture.Start("deepseek_completion", dsprotocol.DeepSeekCompletionURL, a.AccountID, payload)
	attempts := 0
	var lastErrDetail string
	for attempts < maxAttempts {
		resp, err := c.streamPost(ctx, clients.stream, dsprotocol.DeepSeekCompletionURL, headers, payload)
		if err != nil {
			lastErrDetail = "transport: " + err.Error()
			config.Logger.Warn("[deepseek] completion stream post failed", "url", dsprotocol.DeepSeekCompletionURL, "error", err, "account", a.AccountID, "attempt", attempts+1)
			attempts++
			time.Sleep(time.Second)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			if captureSession != nil {
				resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
			}
			resp = c.wrapCompletionWithAutoContinue(ctx, a, payload, powResp, resp)
			return resp, nil
		}
		if captureSession != nil {
			resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
		}
		body, readErr := readResponseBody(resp)
		if closeErr := resp.Body.Close(); closeErr != nil {
			config.Logger.Warn("[deepseek] completion non-OK body close failed", "error", closeErr)
		}
		prev := ""
		if readErr != nil {
			prev = "read: " + readErr.Error()
		} else {
			prev = preview(body)
		}
		lastErrDetail = fmt.Sprintf("status=%d preview=%q", resp.StatusCode, prev)
		config.Logger.Warn("[deepseek] completion upstream non-OK", "status", resp.StatusCode, "content_encoding", resp.Header.Get("Content-Encoding"), "preview", prev, "account", a.AccountID, "attempt", attempts+1)
		attempts++
		time.Sleep(time.Second)
	}
	if lastErrDetail == "" {
		return nil, fmt.Errorf("completion failed after %d attempts", maxAttempts)
	}
	return nil, fmt.Errorf("completion failed after %d attempts, last: %s", maxAttempts, lastErrDetail)
}

func (c *Client) streamPost(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers = c.jsonHeaders(headers)
	clients := c.requestClientsFromContext(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := doer.Do(req)
	if err != nil {
		config.Logger.Warn("[deepseek] fingerprint stream request failed, fallback to std transport", "url", url, "error", err)
		req2, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if reqErr != nil {
			return nil, reqErr
		}
		for k, v := range headers {
			req2.Header.Set(k, v)
		}
		return clients.fallbackS.Do(req2)
	}
	return resp, nil
}
