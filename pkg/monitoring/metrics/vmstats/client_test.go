package vmstats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("VMStatsClient", func() {
	It("should fetch and parse bulk VMStats", func() {
		expected := map[string]*VMStatsResult{
			"ns1/vm1": {
				Stats: &VMStats{
					DomainStats: DomainStats{
						Name: "ns1_vm1",
						Cpu:  &DomainStatsCPU{TimeSet: true, Time: 1_000_000_000},
					},
				},
			},
			"ns1/vm2": {
				Stats: &VMStats{
					DomainStats: DomainStats{
						Name: "ns1_vm2",
						Cpu:  &DomainStatsCPU{TimeSet: true, Time: 2_000_000_000},
					},
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.URL.Path).To(Equal("/v1/vmstats"))
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(expected)).To(Succeed())
		}))
		defer server.Close()

		client := NewVMStatsClient(server.Client(), 0)
		client.baseURLOverride = server.URL

		results, err := client.FetchNodeVMStats(context.Background(), "unused")
		Expect(err).ToNot(HaveOccurred())
		Expect(results).To(HaveLen(2))
		Expect(results["ns1/vm1"].Stats.DomainStats.Name).To(Equal("ns1_vm1"))
		Expect(results["ns1/vm2"].Stats.DomainStats.Cpu.Time).To(Equal(uint64(2_000_000_000)))
	})

	It("should handle partial failures in response", func() {
		expected := map[string]*VMStatsResult{
			"ns1/vm1": {
				Stats: &VMStats{
					DomainStats: DomainStats{Name: "ns1_vm1"},
				},
			},
			"ns1/vm2": {
				Error: "failed to connect to cmd client socket",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(expected)).To(Succeed())
		}))
		defer server.Close()

		client := NewVMStatsClient(server.Client(), 0)
		client.baseURLOverride = server.URL

		results, err := client.FetchNodeVMStats(context.Background(), "unused")
		Expect(err).ToNot(HaveOccurred())
		Expect(results).To(HaveLen(2))
		Expect(results["ns1/vm1"].Stats).ToNot(BeNil())
		Expect(results["ns1/vm1"].Error).To(BeEmpty())
		Expect(results["ns1/vm2"].Stats).To(BeNil())
		Expect(results["ns1/vm2"].Error).To(ContainSubstring("failed to connect"))
	})

	It("should return error on non-200 response", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		client := NewVMStatsClient(server.Client(), 0)
		client.baseURLOverride = server.URL

		_, err := client.FetchNodeVMStats(context.Background(), "unused")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("403"))
	})

	It("should return error on invalid JSON", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer server.Close()

		client := NewVMStatsClient(server.Client(), 0)
		client.baseURLOverride = server.URL

		_, err := client.FetchNodeVMStats(context.Background(), "unused")
		Expect(err).To(HaveOccurred())
	})

	Describe("NewTLSConfigFromCA", func() {
		It("should return error for invalid CA data", func() {
			_, err := NewTLSConfigFromCA([]byte("not a cert"))
			Expect(err).To(HaveOccurred())
		})
	})
})
