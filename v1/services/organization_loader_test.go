package services_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/testing/mocks"
	"github.com/cloudfoundry-incubator/notifications/v1/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrganizationLoader", func() {
	Describe("Load", func() {
		var loader services.OrganizationLoader
		var token string
		var cc *mocks.CloudController

		BeforeEach(func() {
			cc = mocks.NewCloudController()
			cc.Orgs = map[string]cf.CloudControllerOrganization{
				"org-001": {
					GUID: "org-001",
					Name: "org-name",
				},
				"org-123": {
					GUID: "org-123",
					Name: "org-piggies",
				},
			}
			loader = services.NewOrganizationLoader(cc)
		})

		It("returns the org", func() {
			org, err := loader.Load("org-001", token)
			if err != nil {
				panic(err)
			}

			Expect(org).To(Equal(cf.CloudControllerOrganization{
				GUID: "org-001",
				Name: "org-name",
			}))
		})

		Context("when the org cannot be found", func() {
			It("returns an error object", func() {
				_, err := loader.Load("org-doesnotexist", token)

				Expect(err).To(BeAssignableToTypeOf(services.CCNotFoundError("")))
				Expect(err.Error()).To(Equal(`CloudController Error: CloudController Failure (404): {"code":30003,"description":"The organization could not be found: org-doesnotexist","error_code":"CF-OrganizationNotFound"}`))
			})
		})

		Context("when Load returns any other type of error", func() {
			It("returns a CCDownError when the error is cf.Failure", func() {
				failure := cf.NewFailure(401, "BOOM!")
				cc.LoadOrganizationError = failure
				_, err := loader.Load("org-001", token)

				Expect(err).To(Equal(services.CCDownError(failure.Error())))
			})

			It("returns the same error for all other cases", func() {
				cc.LoadOrganizationError = errors.New("BOOM!")
				_, err := loader.Load("org-001", token)

				Expect(err).To(Equal(errors.New("BOOM!")))
			})
		})
	})
})
