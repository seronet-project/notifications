package handlers_test

import (
    "bytes"
    "encoding/json"
    "errors"
    "log"
    "net/http"
    "net/http/httptest"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/web/handlers"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

type FakeCloudController struct {
    UsersBySpaceGuid         map[string][]cf.CloudControllerUser
    CurrentToken             string
    GetUsersBySpaceGuidError error
}

func NewFakeCloudController() *FakeCloudController {
    return &FakeCloudController{
        UsersBySpaceGuid: make(map[string][]cf.CloudControllerUser),
    }
}

func (fake *FakeCloudController) GetUsersBySpaceGuid(guid, token string) ([]cf.CloudControllerUser, error) {
    fake.CurrentToken = token

    if users, ok := fake.UsersBySpaceGuid[guid]; ok {
        return users, fake.GetUsersBySpaceGuidError
    } else {
        return make([]cf.CloudControllerUser, 0), fake.GetUsersBySpaceGuidError
    }
}

var _ = Describe("NotifySpace", func() {
    Describe("ServeHTTP", func() {
        var handler handlers.NotifySpace
        var writer *httptest.ResponseRecorder
        var request *http.Request
        var buffer *bytes.Buffer
        var fakeCC *FakeCloudController

        BeforeEach(func() {
            var err error

            writer = httptest.NewRecorder()
            body, err := json.Marshal(map[string]string{
                "kind": "test_email",
                "text": "This is the body of the email",
            })
            if err != nil {
                panic(err)
            }

            request, err = http.NewRequest("POST", "/spaces/space-001", bytes.NewBuffer(body))
            if err != nil {
                panic(err)
            }

            buffer = bytes.NewBuffer([]byte{})
            logger := log.New(buffer, "", 0)
            fakeCC = NewFakeCloudController()

            fakeUAA := FakeUAAClient{
                ClientToken: uaa.Token{
                    Access: "the-app-token",
                },
            }

            handler = handlers.NewNotifySpace(logger, fakeCC, fakeUAA)
        })

        It("logs the UUIDs of all users in the space", func() {
            fakeCC.UsersBySpaceGuid["space-001"] = []cf.CloudControllerUser{
                cf.CloudControllerUser{Guid: "user-123"},
                cf.CloudControllerUser{Guid: "user-456"},
                cf.CloudControllerUser{Guid: "user-789"},
            }

            handler.ServeHTTP(writer, request)

            Expect(fakeCC.CurrentToken).To(Equal("the-app-token"))

            lines := strings.Split(buffer.String(), "\n")

            Expect(lines).To(ContainElement("user-123"))
            Expect(lines).To(ContainElement("user-456"))
            Expect(lines).To(ContainElement("user-789"))
        })

        It("validates the presence of required fields", func() {
            request, err := http.NewRequest("POST", "/spaces/space-001", strings.NewReader(""))
            if err != nil {
                panic(err)
            }

            handler.ServeHTTP(writer, request)

            Expect(writer.Code).To(Equal(422))
            body := make(map[string]interface{})
            err = json.Unmarshal(writer.Body.Bytes(), &body)
            if err != nil {
                panic(err)
            }

            Expect(body["errors"]).To(ContainElement(`"kind" is a required field`))
            Expect(body["errors"]).To(ContainElement(`"text" is a required field`))
        })

        It("returns a 502 when CloudController fails to respond", func() {
            fakeCC.GetUsersBySpaceGuidError = errors.New("BOOM!")

            handler.ServeHTTP(writer, request)

            Expect(writer.Code).To(Equal(http.StatusBadGateway))
            body := make(map[string]interface{})
            err := json.Unmarshal(writer.Body.Bytes(), &body)
            if err != nil {
                panic(err)
            }

            Expect(body["errors"]).To(ContainElement("Cloud Controller is unavailable"))
        })
    })
})