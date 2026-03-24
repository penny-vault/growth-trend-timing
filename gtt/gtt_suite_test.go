package gtt_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGrowthTrendTiming(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Growth-Trend Timing Suite")
}
