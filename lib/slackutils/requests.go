package slackutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/graytonio/slack-cli/lib/config"
)

func RawSlackRequestFormData(method string, path string, body map[string]string, query map[string]string) ([]byte, int, error) {
	reqUrl, err := url.JoinPath("https://slack.com/api/", path)
	if err != nil {
		return nil, -1, err
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, v := range body {
		var fw io.Writer
		var err error
		if fw, err = w.CreateFormField(k); err != nil {
			return nil, -1, err
		}
		if _, err := io.Copy(fw, bytes.NewBufferString(v)); err != nil {
			return nil, -1, err
		}
	}
	w.Close()

	req, err := http.NewRequest(method, reqUrl, &b)
	if err != nil {
		return nil, -1, err
	}

	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.GetConfig().SlackCredentials.UserToken))

	resp, err := config.SlackHTTPClient.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, err
	}

	return respBody, resp.StatusCode, nil
}

func RawSlackRequest(method string, path string, body any, query map[string]string) ([]byte, int, error) {
	reqUrl, err := url.JoinPath("https://slack.com/api/", path)
	if err != nil {
		return nil, -1, err
	}

	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, -1, err
	}

	req, err := http.NewRequest(method, reqUrl, bytes.NewBuffer(bodyData))
	if err != nil {
		return nil, -1, err
	}

	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.GetConfig().SlackCredentials.UserToken))

	resp, err := config.SlackHTTPClient.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, err
	}

	return respBody, resp.StatusCode, nil
}