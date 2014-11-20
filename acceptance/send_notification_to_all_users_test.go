package acceptance

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/notifications/acceptance/servers"
	"github.com/cloudfoundry-incubator/notifications/config"
	"github.com/pivotal-cf/uaa-sso-golang/uaa"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Send a notification to all users of UAA", func() {
	BeforeEach(func() {
		TruncateTables()
	})

	It("sends an email notification to all users of UAA", func() {
		// Boot Fake SMTP Server
		smtpServer := servers.NewSMTP()
		smtpServer.Boot()

		// Boot Fake UAA Server
		uaaServer := servers.NewUAA()
		uaaServer.Boot()
		defer uaaServer.Close()

		// Boot Real Notifications Server
		notificationsServer := servers.NewNotifications()
		notificationsServer.Boot()
		defer notificationsServer.Close()

		// Retrieve UAA token
		env := config.NewEnvironment()
		uaaClient := uaa.NewUAA("", env.UAAHost, "notifications-sender", "secret", "")
		clientToken, err := uaaClient.GetClientToken()
		if err != nil {
			panic(err)
		}

		t := SendNotificationToAllUsers{}
		t.RegisterClientNotification(notificationsServer, clientToken)
		t.SendNotificationToAllUsers(notificationsServer, clientToken, smtpServer)
	})
})

type SendNotificationToAllUsers struct{}

// Make request to /registation
func (t SendNotificationToAllUsers) RegisterClientNotification(notificationsServer servers.Notifications, clientToken uaa.Token) {
	body, err := json.Marshal(map[string]interface{}{
		"source_description": "Notifications Sender",
		"kinds": []map[string]string{
			{
				"id":          "acceptance-test",
				"description": "Acceptance Test",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("PUT", notificationsServer.RegistrationPath(), bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	request.Header.Set("Authorization", "Bearer "+clientToken.Access)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	// Confirm response status code looks ok
	Expect(response.StatusCode).To(Equal(http.StatusOK))
}

func (t SendNotificationToAllUsers) SendNotificationToAllUsers(notificationsServer servers.Notifications, clientToken uaa.Token, smtpServer *servers.SMTP) {
	body, err := json.Marshal(map[string]string{
		"kind_id": "acceptance-test",
		"html":    "<p>this is an acceptance%40test</p>",
		"subject": "my-spammy-subject",
	})
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("POST", notificationsServer.EveryonePath(), bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	request.Header.Set("Authorization", "Bearer "+clientToken.Access)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	// Confirm the request response looks correct
	Expect(response.StatusCode).To(Equal(http.StatusOK))

	responseJSON := []map[string]string{}
	err = json.Unmarshal(body, &responseJSON)
	if err != nil {
		panic(err)
	}

	Expect(len(responseJSON)).To(Equal(2))

	indexedResponses := map[string]map[string]string{}
	for _, resp := range responseJSON {
		indexedResponses[resp["recipient"]] = resp
	}

	responseItem := indexedResponses["091b6583-0933-4d17-a5b6-66e54666c88e"]
	Expect(responseItem["recipient"]).To(Equal("091b6583-0933-4d17-a5b6-66e54666c88e"))
	Expect(responseItem["status"]).To(Equal("queued"))
	Expect(GUIDRegex.MatchString(responseItem["notification_id"])).To(BeTrue())

	responseItem = indexedResponses["943e6076-b1a5-4404-811b-a1ee9253bf56"]
	Expect(responseItem["recipient"]).To(Equal("943e6076-b1a5-4404-811b-a1ee9253bf56"))
	Expect(responseItem["status"]).To(Equal("queued"))
	Expect(GUIDRegex.MatchString(responseItem["notification_id"])).To(BeTrue())

	Eventually(func() int {
		return len(smtpServer.Deliveries)
	}, 5*time.Second).Should(Equal(2))
	delivery := smtpServer.Deliveries[0]

	env := config.NewEnvironment()
	Expect(delivery.Sender).To(Equal(env.Sender))
	Expect(delivery.Recipients).To(Equal([]string{"why-email@example.com"}))

	data := strings.Split(string(delivery.Data), "\n")
	Expect(data).To(ContainElement("X-CF-Client-ID: notifications-sender"))
	Expect(data).To(ContainElement("X-CF-Notification-ID: " + indexedResponses["091b6583-0933-4d17-a5b6-66e54666c88e"]["notification_id"]))
	Expect(data).To(ContainElement("Subject: CF Notification: my-spammy-subject"))
	Expect(data).To(ContainElement(`<p>The following "Acceptance Test" notification was sent to you directly by the`))
	Expect(data).To(ContainElement(`    "Notifications Sender" component of Cloud Foundry:</p>`))
	Expect(data).To(ContainElement("<p>this is an acceptance%40test</p>"))
}