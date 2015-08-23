package services_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/testing/mocks"
	"github.com/cloudfoundry-incubator/notifications/v1/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SpaceLoader", func() {
	Describe("Load", func() {
		var loader services.SpaceLoader
		var token string
		var cc *mocks.CloudController

		BeforeEach(func() {
			cc = mocks.NewCloudController()
			cc.Spaces = map[string]cf.CloudControllerSpace{
				"space-001": {
					GUID:             "space-001",
					Name:             "space-name",
					OrganizationGUID: "org-001",
				},
			}
			loader = services.NewSpaceLoader(cc)
		})

		It("returns the space", func() {
			space, err := loader.Load("space-001", token)
			if err != nil {
				panic(err)
			}

			Expect(space).To(Equal(cf.CloudControllerSpace{
				GUID:             "space-001",
				Name:             "space-name",
				OrganizationGUID: "org-001",
			}))
		})

		Context("when the space cannot be found", func() {
			It("returns an error object", func() {
				_, err := loader.Load("space-doesnotexist", token)

				Expect(err).To(BeAssignableToTypeOf(services.CCNotFoundError("")))
				Expect(err.Error()).To(Equal(`CloudController Error: CloudController Failure (404): {"code":40004,"description":"The app space could not be found: space-doesnotexist","error_code":"CF-SpaceNotFound"}`))
			})
		})

		Context("when Load returns any other type of error", func() {
			It("returns a CCDownError when the error is cf.Failure", func() {
				failure := cf.NewFailure(401, "BOOM!")
				cc.LoadSpaceError = failure
				_, err := loader.Load("space-001", token)

				Expect(err).To(Equal(services.CCDownError(failure.Error())))
			})

			It("returns the same error for all other cases", func() {
				cc.LoadSpaceError = errors.New("BOOM!")
				_, err := loader.Load("space-001", token)

				Expect(err).To(Equal(errors.New("BOOM!")))
			})
		})
	})
})
