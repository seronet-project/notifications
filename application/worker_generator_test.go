package application_test

import (
	"github.com/cloudfoundry-incubator/notifications/application"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockWorker int

func (m *mockWorker) Work() {
	*m++
}

var _ = Describe("WorkerGenerator", func() {
	Describe("#Work", func() {
		var (
			workerIDs []int
			worker    mockWorker
		)

		BeforeEach(func() {
			worker = mockWorker(0)
			application.WorkerGenerator{Count: 5, InstanceIndex: 2}.Work(func(id int) application.Worker {
				workerIDs = append(workerIDs, id)
				return &worker
			})
		})

		It("should allow the consumer to generate a Worker using an ID that is unique across instances", func() {
			Expect(workerIDs).To(Equal([]int{11, 12, 13, 14, 15}))
		})

		It("should do work on each worker", func() {
			Expect(worker).To(BeEquivalentTo(5))
		})
	})
})
