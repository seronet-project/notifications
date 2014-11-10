package fakes

import (
    "github.com/cloudfoundry-incubator/notifications/models"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/cloudfoundry-incubator/notifications/postal/strategies"
)

type MailStrategy struct {
    DispatchArguments []interface{}
    Responses         []strategies.Response
    Error             error
    TrimCalled        bool
}

func NewMailStrategy() *MailStrategy {
    return &MailStrategy{}
}

func (fake *MailStrategy) Dispatch(clientID string, guid postal.TypedGUID,
    options postal.Options, conn models.ConnectionInterface) ([]strategies.Response, error) {

    fake.DispatchArguments = []interface{}{clientID, guid, options}
    return fake.Responses, fake.Error
}

func (fake *MailStrategy) Trim(response []byte) []byte {
    fake.TrimCalled = true
    return response
}