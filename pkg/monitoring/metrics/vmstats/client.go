/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package vmstats

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type VMStatsClient struct {
	httpClient      *http.Client
	port            int
	baseURLOverride string
}

func NewVMStatsClient(httpClient *http.Client, port int) *VMStatsClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &VMStatsClient{httpClient: httpClient, port: port}
}

var defaultQueryParams = []string{
	"domainStats",
	"dirtyRate",
	"guestGetOsInfo",
	"guestGetHostName",
	"guestGetTimezone",
	"guestGetUsers",
	"guestGetDiskStats",
	"guestNetworkGetInterfaces",
}

func (c *VMStatsClient) FetchNodeVMStats(ctx context.Context, podIP string) (map[string]*VMStatsResult, error) {
	baseURL := c.baseURLOverride
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s:%d", podIP, c.port)
	}
	url := fmt.Sprintf("%s/v1/vmstats?%s", baseURL, buildQueryString(defaultQueryParams))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, podIP)
	}

	var results map[string]*VMStatsResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return results, nil
}

func buildQueryString(params []string) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		parts = append(parts, p+"=true")
	}
	return strings.Join(parts, "&")
}

func NewTLSConfigFromCA(caData []byte) (*tls.Config, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}, nil
}
