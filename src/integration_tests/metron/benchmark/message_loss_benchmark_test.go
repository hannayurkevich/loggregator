package benchmark_test

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("MessageLossBenchmark", func() {
	BeforeEach(func() {
		command := exec.Command(pathToMetronExecutable, "--config=fixtures/metron.json")
		var err error
		metronSession, err = gexec.Start(command, gexec.NewPrefixedWriter("[o][metron]", GinkgoWriter), gexec.NewPrefixedWriter("[e][metron]", GinkgoWriter))
		Expect(err).ToNot(HaveOccurred())

		Eventually(metronSession.Buffer).Should(gbytes.Say("metron started"))
	})

	AfterEach(func() {
		metronSession.Kill().Wait()
	})

	Measure("message loss and throughput", func(b Benchmarker) {
		command := exec.Command(pathToMetronBenchmarkExec, "-writeRate", "5000", "-interval",
			"10s", "-stopAfter", "11s")

		outBuffer := bytes.NewBuffer(nil)
		errBuffer := bytes.NewBuffer(nil)
		benchmarkSession, err := gexec.Start(command, outBuffer, errBuffer)
		Expect(err).ToNot(HaveOccurred())
		Eventually(benchmarkSession, 15).Should(gexec.Exit())

		out := outBuffer.String()
		Expect(out).To(ContainSubstring("PercentLoss"))
		lines := strings.Split(out, "\n")
		Expect(lines).To(HaveLen(4))
		values := strings.Split(lines[1], ", ")
		Expect(values).To(HaveLen(5))

		value := strings.Split(values[4], "%")[0]
		percentLoss, err := strconv.ParseFloat(value, 64)
		Expect(err).ToNot(HaveOccurred())
		Expect(percentLoss).To(BeNumerically("<", 1.0))
		b.RecordValue("message loss (percent)", percentLoss)

		messageThroughput, err := strconv.ParseFloat(values[1], 64)
		Expect(err).ToNot(HaveOccurred())
		b.RecordValue("message throughput", messageThroughput)
	}, 3)

	Measure("concurrent message loss and throughput", func(b Benchmarker) {
		command := exec.Command(pathToMetronBenchmarkExec, "-writeRate", "500", "-interval",
			"10s", "-stopAfter", "15s", "-concurrentWriters", "10")
		outBuffer := bytes.NewBuffer(nil)
		errBuffer := bytes.NewBuffer(nil)
		benchmarkSession, err := gexec.Start(command, outBuffer, errBuffer)
		Expect(err).ToNot(HaveOccurred())
		Eventually(benchmarkSession, 18).Should(gexec.Exit())

		out := outBuffer.String()
		Expect(out).To(ContainSubstring("PercentLoss"))

		lines := strings.Split(out, "\n")
		Expect(lines).To(HaveLen(4))

		values := strings.Split(lines[1], ", ")
		Expect(values).To(HaveLen(5))

		value := strings.Split(values[4], "%")[0]
		percentLoss, err := strconv.ParseFloat(value, 64)
		Expect(err).ToNot(HaveOccurred())
		Expect(percentLoss).To(BeNumerically("<", 1.0))

		b.RecordValue("message loss (percent)", percentLoss)

		messageThroughput, err := strconv.ParseFloat(values[1], 64)
		Expect(err).ToNot(HaveOccurred())
		b.RecordValue("message throughput", messageThroughput)
	}, 3)
})
